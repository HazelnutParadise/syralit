package main

import (
	"fmt"
	"strings"

	sy "github.com/HazelnutParadise/syralit"
)

type registration struct {
	Name        string
	Email       string
	Company     string
	Role        string
	Topics      []string
	Experience  string
	Expectation float64
	Dietary     string
}

func main() {
	sy.App(func() {
		sy.SetPageConfig(sy.PageTitle("Conference Registration"), sy.ConfigIcon("📋"))

		sy.Title("Conference Registration")
		sy.Markdown("Fill out the form below to register for **GopherCon 2026**. All fields inside the form are batched — nothing reruns until you hit **Submit**.")

		sy.Divider()

		submissions := sy.State("submissions", []registration{})

		sy.Form("registration", func() {
			sy.Subheader("Personal Information")
			cols := sy.Columns(2)
			var name, email string
			cols[0](func() {
				name = sy.TextInput("Full Name", sy.Key("reg_name"), sy.Placeholder("Jane Doe"))
			})
			cols[1](func() {
				email = sy.TextInput("Email Address", sy.Key("reg_email"), sy.Placeholder("jane@example.com"))
			})

			company := sy.TextInput("Company / Organization", sy.Key("reg_company"), sy.Placeholder("Acme Corp"))

			sy.Divider()
			sy.Subheader("Conference Details")

			role := sy.SelectBox("Role", []string{"Attendee", "Speaker", "Sponsor"}, sy.Key("reg_role"))

			topics := sy.MultiSelect("Topics of Interest", []string{
				"Web Development",
				"Cloud & Infrastructure",
				"AI / Machine Learning",
				"DevOps & SRE",
				"Security",
				"Open Source",
				"Performance",
				"Testing & QA",
			}, sy.Key("reg_topics"))

			experience := sy.Radio("Experience Level", []string{"Beginner", "Intermediate", "Advanced", "Expert"}, sy.Key("reg_exp"))

			expectation := sy.Slider("Satisfaction Expectation (1-10)", 1, 10, sy.Key("reg_expect"), sy.DefaultValue(7.0))

			sy.Divider()
			sy.Subheader("Additional Info")

			dietary := sy.TextArea("Dietary Requirements or Notes", sy.Key("reg_dietary"), sy.Placeholder("e.g. vegetarian, allergies..."), sy.Height(80))

			agreed := sy.Checkbox("I agree to the conference terms and code of conduct", sy.Key("reg_agree"))

			if sy.FormSubmitButton("Submit Registration") {
				if name == "" || email == "" {
					sy.Toast("Please fill in your name and email.", "error")
				} else if !agreed {
					sy.Toast("You must agree to the terms before submitting.", "warning")
				} else {
					entry := registration{
						Name:        name,
						Email:       email,
						Company:     company,
						Role:        role,
						Topics:      topics,
						Experience:  experience,
						Expectation: expectation,
						Dietary:     dietary,
					}
					submissions.Set(append(submissions.Get(), entry))
					sy.Toast(fmt.Sprintf("Welcome, %s! Registration received.", name), "success")
				}
			}
		})

		sy.Divider()

		// --- Summary section ---
		all := submissions.Get()
		count := len(all)

		sy.Header("Submissions Overview")

		cols := sy.Columns(3)
		cols[0](func() {
			sy.Metric("Total Registrations", fmt.Sprintf("%d", count))
		})
		cols[1](func() {
			speakers := 0
			for _, r := range all {
				if r.Role == "Speaker" {
					speakers++
				}
			}
			sy.Metric("Speakers", fmt.Sprintf("%d", speakers))
		})
		cols[2](func() {
			if count > 0 {
				var sum float64
				for _, r := range all {
					sum += r.Expectation
				}
				sy.Metric("Avg Expectation", fmt.Sprintf("%.1f / 10", sum/float64(count)))
			} else {
				sy.Metric("Avg Expectation", "-")
			}
		})

		if count > 0 {
			sy.Expander(fmt.Sprintf("View all %d submission(s)", count), func() {
				rows := make([][]any, len(all))
				for i, r := range all {
					rows[i] = []any{
						i + 1,
						r.Name,
						r.Email,
						r.Company,
						r.Role,
						strings.Join(r.Topics, ", "),
						r.Experience,
						fmt.Sprintf("%.0f", r.Expectation),
					}
				}
				sy.DataFrame(
					[]string{"#", "Name", "Email", "Company", "Role", "Topics", "Experience", "Expectation"},
					rows,
				)
			})
		} else {
			sy.Info("No submissions yet. Fill out the form above to get started!")
		}
	})
}
