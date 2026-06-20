package main

import (
	"fmt"
	"time"

	sy "github.com/HazelnutParadise/syralit"
)

func init() {
	sy.AddPage("Home", homePage, sy.PageIcon("🏠"), sy.PageOrder(1))
	sy.AddPage("Profile", profilePage, sy.PageIcon("👤"), sy.PageOrder(2))
	sy.AddPage("Admin", adminPage, sy.PageIcon("🔒"), sy.PageOrder(3))
}

func main() { sy.App(nil) }

var validUsers = map[string]string{
	"admin": "admin123",
	"user":  "pass",
	"demo":  "demo",
}

var userRoles = map[string]string{
	"admin": "admin",
	"user":  "member",
	"demo":  "viewer",
}

func homePage() {
	sy.SetPageConfig(sy.PageTitle("Auth Demo"), sy.ConfigIcon("🔐"))

	user := sy.User()
	if user == nil {
		sy.Title("Welcome to the Auth Demo")
		sy.Markdown("This example demonstrates Syralit's authentication features: `LoginGate`, `User`, `Login`, `Logout`, and role-based access control.")

		sy.Divider()
		sy.Info("Please log in to continue. Try one of these accounts:")
		sy.Table(
			[]string{"Username", "Password", "Role"},
			[][]string{
				{"admin", "admin123", "admin"},
				{"user", "pass", "member"},
				{"demo", "demo", "viewer"},
			},
		)

		sy.Divider()

		username := sy.LoginGate(func(user, pass string) bool {
			expected, ok := validUsers[user]
			return ok && pass == expected
		})

		role := userRoles[username]
		sy.Login(map[string]string{
			"name": username,
			"role": role,
		})
		sy.Rerun()
		return
	}

	sy.Title(fmt.Sprintf("Hello, %s!", user["name"]))
	sy.Success(fmt.Sprintf("You are logged in as **%s** with the **%s** role.", user["name"], user["role"]))

	sy.Sidebar(func() {
		sy.Caption(fmt.Sprintf("Logged in as: %s", user["name"]))
		sy.Caption(fmt.Sprintf("Role: %s", user["role"]))
		sy.Divider()
		if sy.Button("Logout", sy.Key("sidebar_logout")) {
			sy.Logout()
			sy.Rerun()
		}
	})

	sy.Divider()
	sy.Header("Features Demonstrated")
	cols := sy.Columns(3)
	cols[0](func() {
		sy.Container(func() {
			sy.Subheader("LoginGate")
			sy.Text("Blocks page rendering until the user authenticates with valid credentials.")
		}, sy.Border())
	})
	cols[1](func() {
		sy.Container(func() {
			sy.Subheader("Role-Based Access")
			sy.Text("The Admin page is restricted to users with the 'admin' role.")
		}, sy.Border())
	})
	cols[2](func() {
		sy.Container(func() {
			sy.Subheader("Session State")
			sy.Text("User info persists across page navigation within the same session.")
		}, sy.Border())
	})
}

func profilePage() {
	user := sy.User()
	if user == nil {
		sy.Warning("You must log in first.")
		sy.PageLink("Go to Home", "Home")
		sy.Stop()
	}

	sy.Title("Profile")

	sy.Header("User Information")
	sy.DataFrame(
		[]string{"Field", "Value"},
		[][]any{
			{"Username", user["name"]},
			{"Role", user["role"]},
			{"Session Active", "Yes"},
			{"Login Time", time.Now().Format("2006-01-02 15:04:05")},
		},
	)

	sy.Divider()

	sy.Header("Account Actions")
	if sy.Button("Logout", sy.Key("profile_logout")) {
		sy.Logout()
		sy.Toast("You have been logged out.", "info")
		sy.SwitchPage("Home")
	}
}

func adminPage() {
	user := sy.User()
	if user == nil {
		sy.Warning("You must log in first.")
		sy.PageLink("Go to Home", "Home")
		sy.Stop()
	}

	if user["role"] != "admin" {
		sy.Error("Access Denied")
		sy.Text(fmt.Sprintf("Your role is '%s'. Only users with the 'admin' role can access this page.", user["role"]))
		sy.Stop()
	}

	sy.Title("Admin Panel")
	sy.Success("You have admin access.")

	sy.Header("Registered Users")
	rows := make([][]any, 0, len(validUsers))
	for name, _ := range validUsers {
		rows = append(rows, []any{name, userRoles[name], "Active"})
	}
	sy.DataFrame(
		[]string{"Username", "Role", "Status"},
		rows,
	)

	sy.Divider()
	sy.Header("System Status")
	cols := sy.Columns(3)
	cols[0](func() { sy.Metric("Active Sessions", "3") })
	cols[1](func() { sy.Metric("Total Users", fmt.Sprintf("%d", len(validUsers))) })
	cols[2](func() { sy.Metric("Uptime", "99.9%", sy.Delta("+0.1%")) })
}
