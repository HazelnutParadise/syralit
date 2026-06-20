// Hello example — exercises the rerun spine: text input, button, session state.
package main

import sy "github.com/HazelnutParadise/syralit"

func main() {
	sy.App(func() {
		sy.Title("Hello, Syralit")

		// Milestone 1: typing reruns the app and the text updates live.
		name := sy.TextInput("Your name", sy.Key("name"))
		if name == "" {
			sy.Caption("Type your name above.")
		} else {
			sy.Text("Hello, " + name + "!")
		}

		// State + button + rerun: a working counter.
		count := sy.State("count", 0)
		if sy.Button("Add", sy.Key("add")) {
			count.Set(count.Get() + 1)
		}
		sy.Textf("Count: %d", count.Get())

		if sy.Checkbox("Show a tip", sy.Key("tip")) {
			sy.Info("Tip: every widget here carries an explicit Key.")
		}
	})
}
