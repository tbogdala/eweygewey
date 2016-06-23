// Copyright 2016, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package eweygewey

const (
	EweyKeyUnknown = iota
	EweyKeyWorld1
	EweyKeyWorld2
	EweyKeyEscape
	EweyKeyEnter
	EweyKeyTab
	EweyKeyBackspace
	EweyKeyInsert
	EweyKeyDelete
	EweyKeyRight
	EweyKeyLeft
	EweyKeyDown
	EweyKeyUp
	EweyKeyPageUp
	EweyKeyPageDown
	EweyKeyHome
	EweyKeyEnd
	EweyKeyCapsLock
	EweyKeyScrollLock
	EweyKeyNumLock
	EweyKeyPrintScreen
	EweyKeyPause
	EweyKeyF1
	EweyKeyF2
	EweyKeyF3
	EweyKeyF4
	EweyKeyF5
	EweyKeyF6
	EweyKeyF7
	EweyKeyF8
	EweyKeyF9
	EweyKeyF10
	EweyKeyF11
	EweyKeyF12
	EweyKeyF13
	EweyKeyF14
	EweyKeyF15
	EweyKeyF16
	EweyKeyF17
	EweyKeyF18
	EweyKeyF19
	EweyKeyF20
	EweyKeyF21
	EweyKeyF22
	EweyKeyF23
	EweyKeyF24
	EweyKeyF25
	EweyKeyLeftShift
	EweyKeyLeftControl
	EweyKeyLeftAlt
	EweyKeyLeftSuper
	EweyKeyRightShift
	EweyKeyRightControl
	EweyKeyRightAlt
	EweyKeyRightSuper
)

// KeyPressEvent represents the data associated with a single key-press event
// from whatever input library is used in conjunction with this package.
type KeyPressEvent struct {
	// the key that was hit if it is alpha-numeric or otherwise able to be
	// stored as a character
	Rune rune

	// if the key was not something that can be stored as a rune, then
	// use the corresponding key enum value here (e.g. eweyKeyF1)
	KeyCode int

	// IsRune indicates if the event for a rune or non-rune key
	IsRune bool

	// ShiftDown indicates if the shift key was down at time of key press
	ShiftDown bool

	// CtrlDown indicates if the ctrl key was down at time of key press
	CtrlDown bool

	// AltDown indicates if the alt key was down at time of key press
	AltDown bool

	// SuperDown indicates if the super key was down at time of key press
	SuperDown bool
}
