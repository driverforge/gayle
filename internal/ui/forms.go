package ui

import (
	"errors"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// PromptValue asks for one required value: the key as the title, the yml's
// prompt description underneath, pre-filled with the current remote value so
// Enter keeps it — the Node `prompt` package flow, huh-flavoured. The value
// may not be left empty (Node's required: true).
func PromptValue(key, description, initial string) (string, error) {
	v := initial
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(key).
				Description("Description: " + description).
				Value(&v).
				Validate(func(s string) error {
					if s == "" {
						return errors.New("a value is required")
					}
					return nil
				}),
		),
	).WithTheme(formTheme()).WithWidth(formWidth())
	if err := form.Run(); err != nil {
		return "", ExpectedIfAborted(err)
	}
	return v, nil
}

// formTheme is the huh theme for gayle's prompts: green key titles (the Node
// prompt painted the key green), readable descriptions, quiet chrome.
func formTheme() huh.Theme {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		t := huh.ThemeBase(isDark)

		t.Form.Base = t.Form.Base.MarginTop(1)

		t.Focused.Base = t.Focused.Base.BorderForeground(lipgloss.Color("2"))
		t.Focused.Card = t.Focused.Base
		t.Focused.Title = t.Focused.Title.Foreground(lipgloss.Color("2")).Bold(true)
		t.Focused.Description = t.Focused.Description.Foreground(lipgloss.Color("250")).MarginBottom(1)
		t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(lipgloss.Color("196"))
		t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(lipgloss.Color("196"))

		t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(lipgloss.Color("2"))
		t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(lipgloss.Color("10"))
		t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(lipgloss.Color("241"))

		// Blurred mirrors focused but hides the bar (huh convention).
		t.Blurred = t.Focused
		t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
		t.Blurred.Card = t.Blurred.Base

		t.Group.Title = t.Focused.Title
		t.Group.Description = t.Focused.Description

		return t
	})
}
