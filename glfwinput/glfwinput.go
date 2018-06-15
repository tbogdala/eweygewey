// Copyright 2016, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package glfwinput

import (
	"time"

	glfw "github.com/go-gl/glfw/v3.1/glfw"
	mgl "github.com/go-gl/mathgl/mgl32"
	gui "github.com/tbogdala/eweygewey"
)

// mouseButtonData is used to track button presses between frames.
type mouseButtonData struct {
	// lastPress is the time the last UP->DOWN transition took place
	lastPress time.Time

	// lastPressLocation is the position of the mouse when the last UP->DOWN
	// transition took place
	lastPressLocation mgl.Vec2

	// lastAction was the last detected action for the button
	lastAction int

	// doubleClickDetected should be set to true if the last UP->DOWN->UP
	// sequence was fast enough to be a double click.
	doubleClickDetected bool

	// lastCheckedAt should be set to the time the functions last checked
	// for action. This way, input can be polled only once per frame.
	lastCheckedAt time.Time
}

// SetInputHandlers sets the input callbacks for the GUI Manager to work with
// GLFW. This function takes advantage of closures to track input across
// multiple calls.
func SetInputHandlers(uiman *gui.Manager, window *glfw.Window) {
	lastMouseX := -1.0
	lastMouseY := -1.0
	lastDeltaX := -1.0
	lastDeltaY := -1.0

	needsMousePosCheck := true

	// at the start of a new frame, reset some flags
	uiman.AddConstructionStartCallback(func(startTime time.Time) {
		needsMousePosCheck = true
	})

	uiman.GetMousePosition = func() (float32, float32) {
		// if we've already checked the position this frame, then just return
		// the old coordinates using math
		if needsMousePosCheck == false {
			return float32(lastMouseX), float32(lastMouseY)
		}

		x, y := window.GetCursorPos()

		// in this package, we reverse the Y location so that the origin
		// is in the lower left corner and not the top left corner.
		_, resY := uiman.GetResolution()
		y = float64(resY) - y

		lastDeltaX = x - lastMouseX
		lastDeltaY = y - lastMouseY
		lastMouseX = x
		lastMouseY = y
		needsMousePosCheck = false
		return float32(x), float32(y)
	}

	uiman.GetMousePositionDelta = func() (float32, float32) {
		// test to see if we polled the delta this frame
		if needsMousePosCheck {
			// if not, then update the location data
			uiman.GetMousePosition()
		}
		return float32(lastDeltaX), float32(lastDeltaY)
	}

	const doubleClickThreshold = 0.5 // seconds
	mouseButtonTracker := make(map[int]mouseButtonData)

	uiman.GetMouseButtonAction = func(button int) int {
		var action int
		var mbData mouseButtonData
		var tracked bool

		// get the mouse button data and return the stale result if we're
		// in the same frame.
		mbData, tracked = mouseButtonTracker[button]
		if tracked == true && mbData.lastCheckedAt == uiman.FrameStart {
			return mbData.lastAction
		}

		// poll the button action
		glfwAction := window.GetMouseButton(glfw.MouseButton(int(glfw.MouseButton1) + button))
		if glfwAction == glfw.Release {
			action = gui.MouseUp
		} else if glfwAction == glfw.Press {
			action = gui.MouseDown
		} else if glfwAction == glfw.Repeat {
			action = gui.MouseDown
		}

		// see if we're tracking this button yet
		if tracked == false {
			// create a new mouse button tracker data object
			if action == gui.MouseDown {
				mx, my := uiman.GetMousePosition()
				mbData.lastPressLocation = mgl.Vec2{mx, my}
				mbData.lastPress = uiman.FrameStart
			} else {
				mbData.lastPress = time.Unix(0, 0)
			}
		} else {
			if action == gui.MouseDown {
				// check to see if there was a transition from UP to DOWN
				if mbData.lastAction == gui.MouseUp {
					// check to see the time between the last UP->DOWN transition
					// and this one. If it's less than the double click threshold
					// then change the doubleClickDetected member so that the
					// next DOWN->UP will return a double click instead.
					if uiman.FrameStart.Sub(mbData.lastPress).Seconds() < doubleClickThreshold {
						mbData.doubleClickDetected = true
					}

					// count this as a press and log the time
					mx, my := uiman.GetMousePosition()
					mbData.lastPressLocation = mgl.Vec2{mx, my}
					mbData.lastPress = uiman.FrameStart
				}
			} else {
				// check to see if there was a transition from DOWN to UP
				if mbData.lastAction == gui.MouseDown {
					if mbData.doubleClickDetected {
						// return the double click
						action = gui.MouseDoubleClick

						// reset the tracker
						mbData.doubleClickDetected = false
					} else {
						// return the single click
						action = gui.MouseClick
					}
				}
			}
		}

		// put the updated data back into the map and return the action
		mbData.lastAction = action
		mbData.lastCheckedAt = uiman.FrameStart
		mouseButtonTracker[button] = mbData
		return action
	}

	uiman.ClearMouseButtonAction = func(buttonNumber int) {
		// get the mouse button data and return the stale result if we're
		// in the same frame.
		mbData, tracked := mouseButtonTracker[buttonNumber]
		if tracked == true {
			mbData.lastAction = gui.MouseUp
			mbData.doubleClickDetected = false
			mouseButtonTracker[buttonNumber] = mbData
		}
	}

	uiman.GetMouseDownPosition = func(button int) (float32, float32) {
		// test to see if we polled the delta this frame
		if needsMousePosCheck {
			// if not, then update the location data
			uiman.GetMousePosition()
		}

		// is the mouse button down?
		if uiman.GetMouseButtonAction(button) != gui.MouseUp {
			var tracked bool
			var mbData mouseButtonData

			// get the mouse button data and return the stale result if we're
			// in the same frame.
			mbData, tracked = mouseButtonTracker[button]
			if tracked == true {
				return mbData.lastPressLocation[0], mbData.lastPressLocation[1]
			}
		}

		// mouse not down or not tracked.
		return -1.0, -1.0
	}

	scrollWheelDelta := float32(0.0)
	scrollWheelCache := float32(0.0)
	uiman.GetScrollWheelDelta = func(useCached bool) float32 {
		if useCached {
			return scrollWheelCache
		}
		scrollWheelCache = scrollWheelDelta
		scrollWheelDelta = 0.0
		return scrollWheelCache
	}

	// create our own handler for the scroll wheel which then passes the
	// correct data to our own scroll wheel handler function
	window.SetScrollCallback(func(w *glfw.Window, xoff float64, yoff float64) {
		scrollWheelDelta += float32(yoff) * uiman.ScrollSpeed
	})

	// stores all of the key press events
	keyBuffer := []gui.KeyPressEvent{}

	// make a translation table from GLFW->EweyGewey key codes
	keyTranslation := make(map[glfw.Key]int)
	keyTranslation[glfw.KeyWorld1] = gui.EweyKeyWorld1
	keyTranslation[glfw.KeyWorld2] = gui.EweyKeyWorld2
	keyTranslation[glfw.KeyEscape] = gui.EweyKeyEscape
	keyTranslation[glfw.KeyEnter] = gui.EweyKeyEnter
	keyTranslation[glfw.KeyTab] = gui.EweyKeyTab
	keyTranslation[glfw.KeyBackspace] = gui.EweyKeyBackspace
	keyTranslation[glfw.KeyInsert] = gui.EweyKeyInsert
	keyTranslation[glfw.KeyDelete] = gui.EweyKeyDelete
	keyTranslation[glfw.KeyRight] = gui.EweyKeyRight
	keyTranslation[glfw.KeyLeft] = gui.EweyKeyLeft
	keyTranslation[glfw.KeyDown] = gui.EweyKeyDown
	keyTranslation[glfw.KeyUp] = gui.EweyKeyUp
	keyTranslation[glfw.KeyPageUp] = gui.EweyKeyPageUp
	keyTranslation[glfw.KeyPageDown] = gui.EweyKeyPageDown
	keyTranslation[glfw.KeyHome] = gui.EweyKeyHome
	keyTranslation[glfw.KeyEnd] = gui.EweyKeyEnd
	keyTranslation[glfw.KeyCapsLock] = gui.EweyKeyCapsLock
	keyTranslation[glfw.KeyNumLock] = gui.EweyKeyNumLock
	keyTranslation[glfw.KeyPrintScreen] = gui.EweyKeyPrintScreen
	keyTranslation[glfw.KeyPause] = gui.EweyKeyPause
	keyTranslation[glfw.KeyF1] = gui.EweyKeyF1
	keyTranslation[glfw.KeyF2] = gui.EweyKeyF2
	keyTranslation[glfw.KeyF3] = gui.EweyKeyF3
	keyTranslation[glfw.KeyF4] = gui.EweyKeyF4
	keyTranslation[glfw.KeyF5] = gui.EweyKeyF5
	keyTranslation[glfw.KeyF6] = gui.EweyKeyF6
	keyTranslation[glfw.KeyF7] = gui.EweyKeyF7
	keyTranslation[glfw.KeyF8] = gui.EweyKeyF8
	keyTranslation[glfw.KeyF9] = gui.EweyKeyF9
	keyTranslation[glfw.KeyF10] = gui.EweyKeyF10
	keyTranslation[glfw.KeyF11] = gui.EweyKeyF11
	keyTranslation[glfw.KeyF12] = gui.EweyKeyF12
	keyTranslation[glfw.KeyF13] = gui.EweyKeyF13
	keyTranslation[glfw.KeyF14] = gui.EweyKeyF14
	keyTranslation[glfw.KeyF15] = gui.EweyKeyF15
	keyTranslation[glfw.KeyF16] = gui.EweyKeyF16
	keyTranslation[glfw.KeyF17] = gui.EweyKeyF17
	keyTranslation[glfw.KeyF18] = gui.EweyKeyF18
	keyTranslation[glfw.KeyF19] = gui.EweyKeyF19
	keyTranslation[glfw.KeyF20] = gui.EweyKeyF20
	keyTranslation[glfw.KeyF21] = gui.EweyKeyF21
	keyTranslation[glfw.KeyF22] = gui.EweyKeyF22
	keyTranslation[glfw.KeyF23] = gui.EweyKeyF23
	keyTranslation[glfw.KeyF24] = gui.EweyKeyF24
	keyTranslation[glfw.KeyF25] = gui.EweyKeyF25
	keyTranslation[glfw.KeyLeftShift] = gui.EweyKeyLeftShift
	keyTranslation[glfw.KeyLeftAlt] = gui.EweyKeyLeftAlt
	keyTranslation[glfw.KeyLeftControl] = gui.EweyKeyLeftControl
	keyTranslation[glfw.KeyLeftSuper] = gui.EweyKeyLeftSuper
	keyTranslation[glfw.KeyRightShift] = gui.EweyKeyRightShift
	keyTranslation[glfw.KeyRightAlt] = gui.EweyKeyRightAlt
	keyTranslation[glfw.KeyRightControl] = gui.EweyKeyRightControl
	keyTranslation[glfw.KeyRightSuper] = gui.EweyKeyRightSuper

	//keyTranslation[glfw.Key] = gui.EweyKey

	// create our own handler for key input so that it can buffer the keys
	// and then consume them in an edit box or whatever widget has focus.
	var prevKeyCallback glfw.KeyCallback
	var prevCharModsCallback glfw.CharModsCallback
	prevKeyCallback = window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if action != glfw.Press && action != glfw.Repeat {
			return
		}

		// we have a new event, so init the structure
		var kpe gui.KeyPressEvent

		// try to look it up in the translation table; if it exists, then we log
		// the event; if it doesn't exist, then we assume it will be caught by
		// the CharMods callback.
		code, okay := keyTranslation[key]
		if okay == false {
			// there are some exceptions to this that will get implemented here.
			// when ctrl is held down, it doesn't appear that runes get sent
			// through the CharModsCallback function, so we must handle the
			// ones we want here.
			if (key == glfw.KeyV) && (mods&glfw.ModControl == glfw.ModControl) {
				kpe.Rune = 'V'
				kpe.IsRune = true
				kpe.CtrlDown = true
			} else {
				return
			}
		} else {
			kpe.KeyCode = code

			// set the modifier flags
			if mods&glfw.ModShift == glfw.ModShift {
				kpe.ShiftDown = true
			}
			if mods&glfw.ModAlt == glfw.ModAlt {
				kpe.AltDown = true
			}
			if mods&glfw.ModControl == glfw.ModControl {
				kpe.CtrlDown = true
			}
			if mods&glfw.ModSuper == glfw.ModSuper {
				kpe.SuperDown = true
			}
		}

		// add it to the keys that have been buffered
		keyBuffer = append(keyBuffer, kpe)

		// if there was a pre-existing callback, we'll chain it here
		if prevKeyCallback != nil {
			prevKeyCallback(w, key, scancode, action, mods)
		}
	})

	window.SetCharModsCallback(func(w *glfw.Window, char rune, mods glfw.ModifierKey) {
		var kpe gui.KeyPressEvent
		//fmt.Printf("SetCharModsCallback Rune: %v | mods:%v | ctrl: %v\n", char, mods, mods&glfw.ModControl)

		// set the character
		kpe.Rune = char
		kpe.IsRune = true

		// set the modifier flags
		if mods&glfw.ModShift == glfw.ModShift {
			kpe.ShiftDown = true
		}
		if mods&glfw.ModAlt == glfw.ModAlt {
			kpe.AltDown = true
		}
		if mods&glfw.ModControl == glfw.ModControl {
			kpe.CtrlDown = true
		}
		if mods&glfw.ModSuper == glfw.ModSuper {
			kpe.SuperDown = true
		}

		// add it to the keys that have been buffered
		keyBuffer = append(keyBuffer, kpe)

		// if there was a pre-existing callback, we'll chain it here
		if prevCharModsCallback != nil {
			prevCharModsCallback(w, char, mods)
		}
	})

	uiman.GetKeyEvents = func() []gui.KeyPressEvent {
		returnVal := keyBuffer
		keyBuffer = keyBuffer[:0]
		return returnVal
	}

	uiman.ClearKeyEvents = func() {
		keyBuffer = keyBuffer[:0]
	}

	uiman.GetClipboardString = func() (string, error) {
		return window.GetClipboardString()
	}

	uiman.SetClipboardString = func(clippy string) {
		window.SetClipboardString(clippy)
	}
}
