package handlers

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/stretchr/testify/require"
)

func numberPickerFixture(count int) *providerNumberKeys {
	options := make([]huh.Option[string], count)
	for i := range options {
		options[i] = huh.NewOption(fmt.Sprintf("%d. provider-%d", i+1, i+1), fmt.Sprintf("provider-%d", i+1))
	}
	selected := options[count-1].Value
	field := huh.NewSelect[string]().Title("Select provider").Options(options...).Value(&selected)
	field.WithKeyMap(huh.NewDefaultKeyMap())
	field.WithWidth(80)
	field.WithHeight(15)
	field.Focus()
	return &providerNumberKeys{field: field, options: options, selected: &selected}
}

func applyPickerKey(n *providerNumberKeys, code rune, text string) tea.Cmd {
	msg := n.filter(nil, tea.KeyPressMsg{Code: code, Text: text})
	if msg == nil {
		return nil
	}
	_, cmd := n.field.Update(msg)
	return cmd
}

func TestNumberedProviderSelection(t *testing.T) {
	t.Parallel()
	picker := numberPickerFixture(12)
	require.Equal(t, "provider-12", *picker.selected, "keep the remembered default until input")
	require.Nil(t, applyPickerKey(picker, '1', "1"))
	require.Equal(t, "provider-1", *picker.selected)
	require.Nil(t, applyPickerKey(picker, '0', "0"))
	require.Equal(t, "provider-10", *picker.selected, "support multi-digit choices")
	require.Contains(t, picker.field.View(), "Select provider [10]")
	require.NotNil(t, applyPickerKey(picker, tea.KeyEnter, ""), "Enter submits the numbered selection")
}

func TestProviderNumberRejectsInvalidAndAllowsCorrection(t *testing.T) {
	t.Parallel()
	picker := numberPickerFixture(5)
	applyPickerKey(picker, '9', "9")
	require.Equal(t, "provider-5", *picker.selected)
	require.Nil(t, applyPickerKey(picker, tea.KeyEnter, ""), "invalid numbers cannot submit the previous row")
	applyPickerKey(picker, tea.KeyBackspace, "")
	applyPickerKey(picker, '2', "2")
	require.Equal(t, "provider-2", *picker.selected)
	require.NotNil(t, applyPickerKey(picker, tea.KeyEnter, ""))
}

func TestProviderNumbersPreserveNavigationAndFiltering(t *testing.T) {
	t.Parallel()
	picker := numberPickerFixture(5)
	applyPickerKey(picker, '2', "2")
	applyPickerKey(picker, tea.KeyDown, "")
	require.Empty(t, picker.digits)
	require.Equal(t, "provider-3", *picker.selected)

	applyPickerKey(picker, '/', "/")
	require.True(t, picker.field.GetFiltering())
	msg := tea.KeyPressMsg{Code: '5', Text: "5"}
	require.Equal(t, msg, picker.filter(nil, msg), "digits in text search must not become shortcuts")
	picker.field.Update(msg)
	hovered, ok := picker.field.Hovered()
	require.True(t, ok)
	require.Equal(t, "provider-5", hovered)
	applyPickerKey(picker, tea.KeyEnter, "") // finish text filtering, not the form
	require.False(t, picker.field.GetFiltering())
	applyPickerKey(picker, '2', "2")
	require.Equal(t, "provider-2", *picker.selected, "number labels stay stable after filtering")
	require.NotNil(t, applyPickerKey(picker, tea.KeyEnter, ""))
}
