package syralit

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"
)

// User-supplied static file systems, registered before the server starts.
// rootStatics back files served at the site root (a project's public/ dir);
// assetOverlays shadow the built-in front-end assets at /_syralit/assets/.
var (
	staticMu      sync.RWMutex
	rootStatics   []fs.FS
	assetOverlays []fs.FS
)

// Static mounts a filesystem whose files are served at the site root, so an
// embedded public/ directory becomes reachable at "/...". Call it before App or
// Run. Multiple calls stack (first match wins); the reserved /_syralit/* routes
// always take precedence, and "/" always renders the app shell.
//
//	//go:embed all:public
//	var public embed.FS
//
//	func main() {
//	    sub, _ := fs.Sub(public, "public")
//	    sy.Static(sub)
//	    sy.App(myApp)
//	}
//
// The `syralit build` command writes this wiring for you automatically.
func Static(fsys fs.FS) {
	if fsys == nil {
		return
	}
	staticMu.Lock()
	rootStatics = append(rootStatics, fsys)
	staticMu.Unlock()
}

// StaticAssets overlays a filesystem on the built-in front-end assets served at
// /_syralit/assets/. A file here (e.g. a custom runtime.css) shadows the
// framework's copy — the production equivalent of `syralit dev`'s assets/ dir.
func StaticAssets(fsys fs.FS) {
	if fsys == nil {
		return
	}
	staticMu.Lock()
	assetOverlays = append(assetOverlays, fsys)
	staticMu.Unlock()
}

// openStatic looks up name (a slash path with no leading slash) across the given
// filesystems, returning the first match. Directories are rejected so a stray
// directory open can't be served as a file.
func openStatic(layers []fs.FS, name string) (fs.File, bool) {
	if name == "" || strings.Contains(name, "..") {
		return nil, false
	}
	name = path.Clean(name)
	staticMu.RLock()
	defer staticMu.RUnlock()
	for _, fsys := range layers {
		f, err := fsys.Open(name)
		if err != nil {
			continue
		}
		if st, err := f.Stat(); err != nil || st.IsDir() {
			_ = f.Close()
			continue
		}
		return f, true
	}
	return nil, false
}

// serveRootStatic serves a user public/ file for the request path, returning
// true if a file was written. The leading slash is stripped; "/" never matches
// (the app shell owns it).
func serveRootStatic(w http.ResponseWriter, r *http.Request) bool {
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" {
		return false
	}
	staticMu.RLock()
	layers := rootStatics
	staticMu.RUnlock()
	f, ok := openStatic(layers, name)
	if !ok {
		return false
	}
	defer f.Close()
	serveFSFile(w, r, name, f)
	return true
}

// serveOverlayAsset serves a user override for a /_syralit/assets/<name> request
// (name already stripped of the prefix), returning true if written.
func serveOverlayAsset(w http.ResponseWriter, r *http.Request, name string) bool {
	staticMu.RLock()
	layers := assetOverlays
	staticMu.RUnlock()
	if len(layers) == 0 {
		return false
	}
	f, ok := openStatic(layers, name)
	if !ok {
		return false
	}
	defer f.Close()
	serveFSFile(w, r, name, f)
	return true
}

// serveFSFile writes an fs.File to the response, using http.ServeContent when the
// file is seekable (for Range/conditional support) and a plain copy otherwise.
func serveFSFile(w http.ResponseWriter, r *http.Request, name string, f fs.File) {
	if rs, ok := f.(io.ReadSeeker); ok {
		modTime := time.Time{}
		if st, err := f.Stat(); err == nil {
			modTime = st.ModTime()
		}
		http.ServeContent(w, r, path.Base(name), modTime, rs)
		return
	}
	if ctype := mime.TypeByExtension(path.Ext(name)); ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	_, _ = io.Copy(w, f)
}
