package main

import (
	"fmt"
	"strings"
	"time"

	sy "github.com/HazelnutParadise/syralit"
)

func main() {
	sy.App(func() {
		sy.SetPageConfig(sy.PageTitle("Syralit Chatbot"), sy.ConfigIcon("💬"))

		sy.Title("Syralit Chatbot")
		sy.Caption("A chatbot UI demo with simulated streaming responses.")

		// Chat history stored as []map[string]string with "role" and "content" keys.
		msgs := sy.State("msgs", []map[string]string{})

		// Sidebar with clear chat button.
		sy.Sidebar(func() {
			sy.Header("Settings")
			if sy.Button("Clear chat", sy.Key("clear_chat")) {
				msgs.Set([]map[string]string{})
				sy.Rerun()
			}
			sy.Divider()
			sy.Caption(fmt.Sprintf("Messages: %d", len(msgs.Get())))
		})

		// Display all previous messages.
		for i, msg := range msgs.Get() {
			role := msg["role"]
			content := msg["content"]
			sy.ChatMessage(role, func() {
				sy.Markdown(content)
				// Show feedback widget on assistant messages.
				if role == "assistant" {
					sy.Feedback(sy.Key(fmt.Sprintf("fb_%d", i)))
				}
			})
		}

		// Chat input.
		prompt := sy.ChatInput("Type a message...")

		if prompt != "" {
			// Append user message to history.
			history := msgs.Get()
			history = append(history, map[string]string{
				"role":    "user",
				"content": prompt,
			})
			msgs.Set(history)

			// Display the user message immediately.
			sy.ChatMessage("user", func() {
				sy.Markdown(prompt)
			})

			// Generate and stream a simulated assistant response.
			reply := generateReply(prompt)

			sy.ChatMessage("assistant", func() {
				sy.WriteStream(func(yield func(string)) {
					words := strings.Fields(reply)
					for i, word := range words {
						if i > 0 {
							yield(" ")
						}
						yield(word)
						time.Sleep(50 * time.Millisecond)
					}
				}, sy.Key(fmt.Sprintf("stream_%d", len(msgs.Get()))))
			})

			// Append assistant message to history.
			history = msgs.Get()
			history = append(history, map[string]string{
				"role":    "assistant",
				"content": reply,
			})
			msgs.Set(history)
		}
	})
}

// generateReply returns a simulated AI response based on the user's input.
func generateReply(prompt string) string {
	lower := strings.ToLower(prompt)

	switch {
	case strings.Contains(lower, "hello") || strings.Contains(lower, "hi"):
		return "Hello! I'm a demo chatbot built with Syralit. How can I help you today?"
	case strings.Contains(lower, "syralit"):
		return "Syralit is a Go-native framework inspired by Streamlit. It lets you build interactive web apps using pure Go, with real-time state management, streaming output, and a rich set of built-in widgets."
	case strings.Contains(lower, "how are you"):
		return "I'm doing great, thanks for asking! I'm just a simulated chatbot, but I appreciate the sentiment."
	case strings.Contains(lower, "help"):
		return "I can answer basic questions about Syralit. Try asking about features, widgets, state management, or streaming. Keep in mind I'm just a demo with canned responses!"
	case strings.Contains(lower, "feature"):
		return "Syralit supports columns, tabs, charts (line, bar, pie, scatter, and more), forms, file uploads, maps, data tables, dialogs, popovers, and of course this chat interface with streaming output."
	case strings.Contains(lower, "state"):
		return "Syralit provides `sy.State[T](key, default)` for typed session state. It returns a `*StateValue[T]` with `.Get()` and `.Set()` methods. State persists across reruns within the same session."
	case strings.Contains(lower, "stream"):
		return "Streaming in Syralit uses `sy.WriteStream`. You pass a function that receives a `yield` callback. Each call to `yield` sends a chunk of text to the client in real-time, perfect for token-by-token AI responses."
	case strings.Contains(lower, "bye") || strings.Contains(lower, "goodbye"):
		return "Goodbye! Thanks for trying the Syralit chatbot demo. Have a great day!"
	default:
		return fmt.Sprintf("You said: \"%s\". This is a demo chatbot with limited responses. Try saying hello, asking about Syralit features, state management, or streaming!", prompt)
	}
}
