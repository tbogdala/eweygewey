// Copyright 2016, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package glfw

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
			mbData.lastAction = action
			mouseButtonTracker[button] = mbData
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
		mouseButtonTracker[button] = mbData
		return action
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
}
