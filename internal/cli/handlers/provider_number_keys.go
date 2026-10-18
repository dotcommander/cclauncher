package handlers

import (
	"fmt"
	"strconv"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

// providerNumberKeys adds number + Enter shortcuts without taking digits away
// from huh's active text filter. Numbers refer to the labels shown in the picker.
type providerNumberKeys struct {
	field    *huh.Select[string]
	options  []huh.Option[string]
	selected *string
	digits   string
}

func (n *providerNumberKeys) filter(_ tea.Model, msg tea.Msg) tea.Msg {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok || n.field.GetFiltering() {
		return msg
	}
	pressed := key.String()
	switch {
	case len(pressed) == 1 && pressed[0] >= '0' && pressed[0] <= '9':
		// Keep one overflow digit to mark an invalid number, but bound input.
		if len(n.digits) <= len(strconv.Itoa(len(n.options))) {
			n.digits += pressed
		}
	case pressed == "backspace" && n.digits != "":
		n.digits = n.digits[:len(n.digits)-1]
	case pressed == "enter" && n.digits != "":
		if !n.selectNumber() {
			return nil
		}
		n.reset()
		return msg
	default:
		n.reset()
		return msg
	}
	if n.digits == "" {
		n.reset()
	} else {
		n.selectNumber()
		n.field.Title(fmt.Sprintf("Select provider [%s]", n.digits))
	}
	return msg
}

func (n *providerNumberKeys) selectNumber() bool {
	number, err := strconv.Atoi(n.digits)
	if err != nil || number < 1 || number > len(n.options) {
		return false
	}
	*n.selected = n.options[number-1].Value
	// A completed text search can leave a filtered subset while GetFiltering
	// is false. Reset options before selecting a stable, numbered row.
	n.field.Options(n.options...).Value(n.selected)
	return true
}

func (n *providerNumberKeys) reset() {
	n.digits = ""
	n.field.Title("Select provider")
}
