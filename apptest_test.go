package syralit

import "testing"

func TestAppTestBasics(t *testing.T) {
	at := NewAppTest(func() {
		Title("Demo")
		name := TextInput("Name", Key("name"))
		if Button("Greet", Key("greet")) {
			Success("Hello, " + name)
		}
		Space()
		if choice := MenuButton("Actions", []string{"Export", "Delete"}, Key("menu")); choice != "" {
			Text("chose: " + choice)
		}
		dt := DatetimeInput("When", Key("when"))
		if dt != "" {
			Text("at: " + dt)
		}
		Bottom(func() {
			Caption("pinned")
		})
	})

	at.Run()
	if got := at.Texts("title"); len(got) != 1 || got[0] != "Demo" {
		t.Fatalf("title = %v", got)
	}
	if len(at.FindAll("status")) != 0 {
		t.Fatal("greeting should not render before click")
	}
	if len(at.FindAll("space")) != 1 || len(at.FindAll("bottom")) != 1 {
		t.Fatal("space/bottom nodes missing")
	}

	// Type a name and click the button (by label to exercise ClickLabel).
	at.SetValue("name", "Ada")
	if err := at.ClickLabel("Greet"); err != nil {
		t.Fatal(err)
	}
	at.Run()
	if got := at.Texts("status"); len(got) != 1 || got[0] != "Hello, Ada" {
		t.Fatalf("greeting = %v", got)
	}
	// Button presses are one-shot.
	at.Run()
	if len(at.FindAll("status")) != 0 {
		t.Fatal("greeting persisted past one rerun")
	}

	// MenuButton is one-shot too.
	at.SetValue("menu", "Export")
	at.Run()
	if got := at.Texts("text"); len(got) != 1 || got[0] != "chose: Export" {
		t.Fatalf("menu choice = %v", got)
	}
	at.Run()
	if len(at.Texts("text")) != 0 {
		t.Fatal("menu choice persisted past one rerun")
	}

	// DatetimeInput normalizes the browser's T separator.
	at.SetValue("when", "2026-07-04T12:30")
	at.Run()
	if got := at.Texts("text"); len(got) != 1 || got[0] != "at: 2026-07-04 12:30" {
		t.Fatalf("datetime = %v", got)
	}
}

func TestGetOption(t *testing.T) {
	resolvedConfig = Config{Title: "X", Host: "127.0.0.1", Port: 9000}
	if GetOption("title") != "X" || GetOption("server.port") != 9000 {
		t.Fatalf("GetOption mismatch: %v %v", GetOption("title"), GetOption("server.port"))
	}
	if GetOption("server.max_upload_size_mb") != 10 {
		t.Fatalf("default upload option = %v", GetOption("server.max_upload_size_mb"))
	}
	if GetOption("nope") != nil {
		t.Fatal("unknown key should be nil")
	}
}
