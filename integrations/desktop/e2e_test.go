//go:build windows

package sydesktop

// End-to-end tests that open real native windows. Gated behind
// SYRALIT_DESKTOP_E2E=1 so a plain `go test ./...` stays headless; the CI
// desktop job (windows-latest) and release checks set the variable.

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

func requireE2E(t *testing.T) {
	t.Helper()
	if os.Getenv("SYRALIT_DESKTOP_E2E") == "" {
		t.Skip("set SYRALIT_DESKTOP_E2E=1 to run desktop e2e tests (opens real windows)")
	}
}

// --- win32 helpers ---

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procPostMessageW             = user32.NewProc("PostMessageW")
)

const wmClose = 0x0010

// findWindows returns the handles and owning PIDs of all top-level windows
// whose title equals title.
func findWindows(title string) (hwnds []uintptr, pids []uint32) {
	cb := syscall.NewCallback(func(h, _ uintptr) uintptr {
		var buf [256]uint16
		n, _, _ := procGetWindowTextW.Call(h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if n > 0 && syscall.UTF16ToString(buf[:n]) == title {
			var pid uint32
			procGetWindowThreadProcessId.Call(h, uintptr(unsafe.Pointer(&pid)))
			hwnds = append(hwnds, h)
			pids = append(pids, pid)
		}
		return 1 // continue enumeration
	})
	procEnumWindows.Call(cb, 0)
	return
}

func waitForWindow(t *testing.T, title string, timeout time.Duration) (uintptr, uint32) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if hs, ps := findWindows(title); len(hs) > 0 {
			return hs[0], ps[0]
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("window %q did not appear within %s", title, timeout)
	return 0, 0
}

func processAlive(pid uint32) bool {
	const processQueryLimitedInformation = 0x1000
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, pid)
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	const stillActive = 259
	return code == stillActive
}

// --- log capture ---

type logBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *logBuf) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *logBuf) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

func (l *logBuf) waitMatch(t *testing.T, re *regexp.Regexp, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if m := re.FindStringSubmatch(l.String()); m != nil {
			return m[len(m)-1]
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("log did not match %v within %s; log:\n%s", re, timeout, l.String())
	return ""
}

func buildFixture(t *testing.T, pkg, out string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", out, pkg)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", pkg, err, b)
	}
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// --- production mode ---

func TestE2EProductionWindow(t *testing.T) {
	requireE2E(t)

	exe := filepath.Join(t.TempDir(), "e2eapp.exe")
	buildFixture(t, "./testdata/e2eapp", exe)

	logs := &logBuf{}
	cmd := exec.Command(exe)
	cmd.Stdout = logs
	cmd.Stderr = logs
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}()

	url := logs.waitMatch(t, regexp.MustCompile(`running on (http://127\.0\.0\.1:\d+)`), 30*time.Second)

	// Browser lockdown: outsiders are rejected with 403.
	for _, u := range []string{url + "/", url + "/?" + lockdownParam + "=wrong"} {
		resp, err := http.Get(u)
		if err != nil {
			t.Fatalf("GET %s: %v", u, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("GET %s = %d, want 403", u, resp.StatusCode)
		}
	}

	// Agent artifact endpoint: passes the lockdown, keeps its own bearer auth.
	post := func(bearer string) int {
		body := strings.NewReader(`{"expected_revision":1,"spec":{"version":"v1","layout":{"columns":1,"gap":8,"padding":8},"nodes":[{"id":"headline","component":"text","props":{"text":"updated by e2e"}}]}}`)
		req, _ := http.NewRequest(http.MethodPost, url+"/api/agent/artifacts/main", body)
		req.Header.Set("Content-Type", "application/json")
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("artifact POST: %v", err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}
	if got := post(""); got != http.StatusUnauthorized {
		t.Fatalf("artifact POST without bearer = %d, want 401 (own auth, not lockdown 403)", got)
	}
	if got := post("e2e-secret"); got != http.StatusOK {
		t.Fatalf("artifact POST with bearer = %d, want 200", got)
	}

	// The native window exists and belongs to our process.
	hwnd, pid := waitForWindow(t, "SyralitE2EWindow", 30*time.Second)
	if int(pid) != cmd.Process.Pid {
		t.Fatalf("window owned by pid %d, app is %d", pid, cmd.Process.Pid)
	}

	// Let WebView2 finish embedding before closing — a WM_CLOSE during
	// controller creation makes Wails treat the aborted creation as fatal
	// (80004004), which is a startup race, not the shutdown path under test.
	time.Sleep(5 * time.Second)

	// Closing the window shuts the whole app down cleanly.
	procPostMessageW.Call(hwnd, wmClose, 0, 0)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("app exited non-zero after window close: %v\n%s", err, logs.String())
		}
	case <-time.After(15 * time.Second):
		t.Fatalf("app did not exit within 15s of window close; log:\n%s", logs.String())
	}
}

// --- dev hot reload ---

const devMainV1 = `package main

import (
	sy "github.com/HazelnutParadise/syralit"
	sydesktop "github.com/HazelnutParadise/syralit/integrations/desktop"
)

func main() {
	sydesktop.App(func() {
		sy.Title("E2E dev MARKER")
	})
}
`

func TestE2EDevHotReload(t *testing.T) {
	requireE2E(t)

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	desktopDir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}

	// A throwaway project using the local checkout. Reuse the desktop-demo
	// example's go.mod/go.sum (full require blocks incl. indirect, so plain
	// `go build` works offline) with the module path and replaces rewritten.
	proj := t.TempDir()
	writeFile(t, filepath.Join(proj, "main.go"), strings.ReplaceAll(devMainV1, "MARKER", "v1"))
	demo := filepath.Join(repoRoot, "examples", "desktop-demo")
	mod, err := os.ReadFile(filepath.Join(demo, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	modTxt := string(mod)
	modTxt = strings.Replace(modTxt,
		"module github.com/HazelnutParadise/syralit/examples/desktop-demo",
		"module e2edev", 1)
	modTxt = strings.Replace(modTxt, "../../integrations/desktop", filepath.ToSlash(desktopDir), 1)
	modTxt = strings.Replace(modTxt, "../../", filepath.ToSlash(repoRoot), 1)
	writeFile(t, filepath.Join(proj, "go.mod"), modTxt)
	sum, err := os.ReadFile(filepath.Join(demo, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(proj, "go.sum"), string(sum))

	sup := filepath.Join(t.TempDir(), "devsup.exe")
	buildFixture(t, "./testdata/devsup", sup)

	port := freeTCPPort(t)
	logs := &logBuf{}
	cmd := exec.Command(sup, proj, fmt.Sprint(port))
	cmd.Stdout = logs
	cmd.Stderr = logs
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
		// The supervisor's hard kill orphans its current dev child; reap it.
		_ = exec.Command("taskkill", "/F", "/IM", fmt.Sprintf("syralit-dev-%d.exe", cmd.Process.Pid)).Run()
	}()

	// Initial build (cold cache on CI) can take a while.
	childListen := regexp.MustCompile(`dev-child]: listening`)
	logs.waitMatch(t, childListen, 3*time.Minute)

	hwnd1, hostPID := waitForWindow(t, "Syralit App (dev)", 60*time.Second)

	// Trigger a rebuild and wait for the new child to come up.
	writeFile(t, filepath.Join(proj, "main.go"), strings.ReplaceAll(devMainV1, "MARKER", "v2"))
	countListens := func() int { return len(childListen.FindAllString(logs.String(), -1)) }
	deadline := time.Now().Add(2 * time.Minute)
	for countListens() < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("no rebuild within 2m; log:\n%s", logs.String())
		}
		time.Sleep(500 * time.Millisecond)
	}
	time.Sleep(2 * time.Second) // let any (buggy) duplicate window appear

	// The same window survived the rebuild — same HWND, same host process,
	// and exactly one of it.
	hs, ps := findWindows("Syralit App (dev)")
	if len(hs) != 1 {
		t.Fatalf("expected exactly 1 dev window after rebuild, found %d", len(hs))
	}
	if hs[0] != hwnd1 || ps[0] != hostPID {
		t.Fatalf("dev window was replaced across rebuild: hwnd %x→%x pid %d→%d", hwnd1, hs[0], hostPID, ps[0])
	}

	// Killing the supervisor makes the window host quit on its own.
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
	deadline = time.Now().Add(20 * time.Second)
	for processAlive(hostPID) {
		if time.Now().After(deadline) {
			t.Fatalf("window host (pid %d) still alive 20s after supervisor died", hostPID)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
