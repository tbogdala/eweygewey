// Copyright 2016, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package eweygewey

import (
	"fmt"

	mgl "github.com/go-gl/mathgl/mgl32"
)

// BuildCallback is a type for the function that builds the widgets for the window.
type BuildCallback func(window *Window)

// Window represents a collection of widgets in the user interface.
type Window struct {
	// Location is the location of the upper left hand corner of the window.
	// The X and Y axis should be specified screen-normalized coordinates.
	Location mgl.Vec3

	// Width is how wide the window is in screen-normalized space.
	Width float32

	// Height is how tall the window is in screen-normalized space.
	Height float32

	// BgColor is the background color of the window
	BgColor mgl.Vec4

	// TitleBarBgColor is the background color of the window title bar
	TitleBarBgColor mgl.Vec4

	// TitleBarTextColor is the background color of the window title bar text
	TitleBarTextColor mgl.Vec4

	// ShowTitleBar indicates if the title bar should be drawn or not
	ShowTitleBar bool

	// IsMoveable indicates if the window should be moveable by LMB drags
	IsMoveable bool

	// Title is the string to display in the title bar if it is visible
	Title string

	// OnBuild gets called by the UI Manager when the UI is getting built.
	// This should be a function that makes all of the calls necessary
	// to build the window's widgets.
	OnBuild BuildCallback

	// Owner is the owning UI Manager object.
	Owner *Manager

	// widgetCursorDC is the current location to insert widgets and should
	// be updated after adding new widgets. This is specified in display
	// coordinates.
	widgetCursorDC mgl.Vec3

	// nextRowCursorOffset is the value the widgetCursorDC's y component
	// should change for the next widget that starts a new row in the window.
	nextRowCursorOffset float32
}

// newWindow creates a new window with a top-left coordinate of (x,y) and
// dimensions of (w,h).
func newWindow(x, y, w, h float32, constructor BuildCallback) *Window {
	wnd := new(Window)
	wnd.Location[0] = x
	wnd.Location[1] = y
	wnd.Width = w
	wnd.Height = h
	wnd.TitleBarTextColor = DefaultStyle.TitleBarTextColor
	wnd.TitleBarBgColor = DefaultStyle.TitleBarBgColor
	wnd.BgColor = DefaultStyle.WindowBgColor
	wnd.OnBuild = constructor
	wnd.ShowTitleBar = true
	wnd.IsMoveable = true
	return wnd
}

// construct builds the frame (if one is to be made) for the window and then
// calls the OnBuild function specified for the window to create the widgets.
func (wnd *Window) construct() {
	mouseX, mouseY := wnd.Owner.GetMousePosition()
	mouseDeltaX, mouseDeltaY := wnd.Owner.GetMousePositionDelta()
	lmbDown := wnd.Owner.GetMouseButtonAction(0) == MouseDown

	// do we need to move the window? (LMB down in a window and mouse dragged)
	if wnd.IsMoveable && lmbDown && wnd.ContainsPosition(mouseX, mouseY) {
		// mouse down in the window, lets move the thing before we make the vertices
		deltaXS, deltaYS := wnd.Owner.DisplayToScreen(mouseDeltaX, mouseDeltaY)
		wnd.Location[0] += deltaXS
		wnd.Location[1] += deltaYS
	}

	// build the frame background for the window
	wnd.buildFrame()

	// invoke the callback to build the widgets for the window
	if wnd.OnBuild != nil {
		wnd.OnBuild(wnd)
	}
}

// buildFrame builds the background for the window
func (wnd *Window) buildFrame() {
	// reset the cursor for the window
	wnd.widgetCursorDC = mgl.Vec3{0, 0, 0}
	wnd.nextRowCursorOffset = 0

	// if we don't have a title bar, then simply render the background frame
	if wnd.ShowTitleBar == false {
		// build the background of the window
		wnd.Owner.DrawRectFilled(wnd.Location[0], wnd.Location[1], wnd.Width, wnd.Height, wnd.BgColor)
		return
	}

	winxDC, winyDC := wnd.Owner.ScreenToDisplay(wnd.Location[0], wnd.Location[1])
	winwDC, winhDC := wnd.Owner.ScreenToDisplay(wnd.Width, wnd.Height)
	_ = winhDC

	// how big should the title bar be?
	titleString := " "
	if len(wnd.Title) > 0 {
		titleString = wnd.Title
	}
	font := wnd.Owner.GetFont(DefaultStyle.FontName)
	_, dimY, _ := font.GetRenderSize(titleString)

	// TODO: for now just add 1 pixel on each side of the string for padding
	titleBarHeight := float32(dimY + 4)

	// render the title bar background
	wnd.Owner.DrawRectFilledDC(winxDC, winyDC, winxDC+winwDC, winyDC-titleBarHeight, wnd.TitleBarBgColor)

	// render the rest of the window background
	wnd.Owner.DrawRectFilledDC(winxDC, winyDC-titleBarHeight, winxDC+winwDC, winyDC-winhDC, wnd.BgColor)

	// render the title bar text
	if len(wnd.Title) > 0 {
		renderData := font.CreateText(mgl.Vec3{winxDC, winyDC, 0}, wnd.TitleBarTextColor, wnd.Title)
		wnd.Owner.AddFaces(renderData.ComboBuffer, renderData.IndexBuffer, renderData.Faces)
	}

	// advance the cursor to account for the title bar
	wnd.widgetCursorDC[1] = wnd.widgetCursorDC[1] - titleBarHeight
}

// ContainsPosition returns true if the position passed in is contained within
// the window's space.
func (wnd *Window) ContainsPosition(x, y float32) bool {
	locXDC, locYDC := wnd.Owner.ScreenToDisplay(wnd.Location[0], wnd.Location[1])
	wndWDC, wndHDC := wnd.Owner.ScreenToDisplay(wnd.Width, wnd.Height)

	if x > locXDC && x < locXDC+wndWDC && y < locYDC && y > locYDC-wndHDC {
		return true
	}
	return false
}

// StartRow starts a new row of widgets in the window.
func (wnd *Window) StartRow() {
	// adjust the widgetCursor if necessary to start a new row.
	wnd.widgetCursorDC[0] = 0.0
	wnd.widgetCursorDC[1] = wnd.widgetCursorDC[1] - wnd.nextRowCursorOffset
}

func (wnd *Window) getCursorDC() mgl.Vec3 {
	// start with the widget DC offet
	pos := wnd.widgetCursorDC

	// add in the position of the window in pixels
	windowDx, windowDy := wnd.Owner.ScreenToDisplay(wnd.Location[0], wnd.Location[1])
	pos[0] += windowDx
	pos[1] += windowDy

	// add in any padding
	style := DefaultStyle
	pos[0] += style.WindowPadding[0]
	pos[1] += style.WindowPadding[2]

	return pos
}

// Text renders a text widget
func (wnd *Window) Text(msg string) error {
	style := DefaultStyle

	// get the font for the text
	font := wnd.Owner.GetFont(style.FontName)
	if font == nil {
		return fmt.Errorf("Couldn't access font %s from the Manager.", style.FontName)
	}

	// calculate the location for the widget
	pos := wnd.getCursorDC()

	// create the text widget itself
	renderData := font.CreateText(pos, style.TextColor, msg)
	wnd.Owner.AddFaces(renderData.ComboBuffer, renderData.IndexBuffer, renderData.Faces)

	// advance the cursor for the width of the text widget
	wnd.widgetCursorDC[0] = wnd.widgetCursorDC[0] + renderData.Width
	wnd.nextRowCursorOffset = renderData.Height

	return nil
}

func (wnd *Window) Button(text string) (bool, error) {
	style := DefaultStyle

	// get the font for the text
	font := wnd.Owner.GetFont(style.FontName)
	if font == nil {
		return false, fmt.Errorf("Couldn't access font %s from the Manager.", style.FontName)
	}

	// calculate the location for the widget
	pos := wnd.getCursorDC()
	pos[0] += style.ButtonMargin[0]
	pos[1] -= style.ButtonMargin[2]

	// calculate the size necessary for the widget
	dimX, dimY, _ := font.GetRenderSize(text)
	buttonW := float32(dimX) + style.ButtonPadding[0] + style.ButtonPadding[1]
	buttonH := float32(dimY) + style.ButtonPadding[2] + style.ButtonPadding[3]

	// set a default color for the button
	bgColor := style.ButtonColor
	buttonPressed := false

	// test to see if the mouse is inside the widget
	mx, my := wnd.Owner.GetMousePosition()
	if mx > pos[0] && my > pos[1]-buttonH && mx < pos[0]+buttonW && my < pos[1] {
		lmbStatus := wnd.Owner.GetMouseButtonAction(0)
		if lmbStatus == MouseUp {
			bgColor = style.ButtonHoverColor
		} else {
			bgColor = style.ButtonActiveColor
			buttonPressed = true
		}
	}

	// render the button background
	wnd.Owner.DrawRectFilledDC(pos[0], pos[1], pos[0]+buttonW, pos[1]-buttonH, bgColor)

	// create the text for the button
	textPos := pos
	textPos[0] += style.ButtonPadding[0]
	textPos[1] -= style.ButtonPadding[2]
	renderData := font.CreateText(textPos, style.ButtonTextColor, text)
	wnd.Owner.AddFaces(renderData.ComboBuffer, renderData.IndexBuffer, renderData.Faces)

	// advance the cursor for the width of the text widget
	wnd.widgetCursorDC[0] = wnd.widgetCursorDC[0] + buttonW + style.ButtonMargin[0] + style.ButtonMargin[1]
	wnd.nextRowCursorOffset = buttonH + style.ButtonMargin[2] + style.ButtonMargin[3]

	return buttonPressed, nil
}
