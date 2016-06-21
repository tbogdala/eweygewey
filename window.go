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
	// ID is the widget id string for the window for claiming focus.
	ID string

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

	// ShowScrollBar indicates if the scroll bar should be attached to the side
	// of the window
	ShowScrollBar bool

	// ShowTitleBar indicates if the title bar should be drawn or not
	ShowTitleBar bool

	// IsMoveable indicates if the window should be moveable by LMB drags
	IsMoveable bool

	// IsScrollable indicates if the window should scroll the contents based
	// on mouse scroll wheel input.
	IsScrollable bool

	// AutoAdjustHeight indicates if the window's height should be automatically
	// adjusted to accommodate all of the widgets.
	AutoAdjustHeight bool

	// Title is the string to display in the title bar if it is visible
	Title string

	// OnBuild gets called by the UI Manager when the UI is getting built.
	// This should be a function that makes all of the calls necessary
	// to build the window's widgets.
	OnBuild BuildCallback

	// Owner is the owning UI Manager object.
	Owner *Manager

	// ScrollOffset is essentially the scrollbar *position* which tells the
	// window hot to offset the controlls to give the scrolling effect.
	ScrollOffset float32

	// widgetCursorDC is the current location to insert widgets and should
	// be updated after adding new widgets. This is specified in display
	// coordinates.
	widgetCursorDC mgl.Vec3

	// nextRowCursorOffset is the value the widgetCursorDC's y component
	// should change for the next widget that starts a new row in the window.
	nextRowCursorOffset float32

	// cmds is the slice of cmdLists used to to render the window
	cmds []*cmdList
}

// newWindow creates a new window with a top-left coordinate of (x,y) and
// dimensions of (w,h).
func newWindow(id string, x, y, w, h float32, constructor BuildCallback) *Window {
	wnd := new(Window)
	wnd.cmds = []*cmdList{}
	wnd.ID = id
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
	//wnd.IsScrollable = false
	return wnd
}

// construct builds the frame (if one is to be made) for the window and then
// calls the OnBuild function specified for the window to create the widgets.
func (wnd *Window) construct() {
	// empty out the cmd list and start a new command
	wnd.cmds = wnd.cmds[:0]

	mouseX, mouseY := wnd.Owner.GetMousePosition()
	mouseDeltaX, mouseDeltaY := wnd.Owner.GetMousePositionDelta()
	lmbDown := wnd.Owner.GetMouseButtonAction(0) == MouseDown

	// if the mouse is in the window, then let's scroll if the scroll input
	// was received.
	if wnd.IsScrollable && wnd.ContainsPosition(mouseX, mouseY) {
		wnd.ScrollOffset -= wnd.Owner.GetScrollWheelDelta(true)
		if wnd.ScrollOffset < 0.0 {
			wnd.ScrollOffset = 0.0
		}
	}

	// reset the cursor for the window
	wnd.widgetCursorDC = mgl.Vec3{0, wnd.ScrollOffset, 0}
	wnd.nextRowCursorOffset = 0

	// advance the cursor to account for the title bar
	_, _, _, frameHeight := wnd.GetFrameSize()
	_, _, _, displayHeight := wnd.GetDisplaySize()
	wnd.widgetCursorDC[1] = wnd.widgetCursorDC[1] - (frameHeight - displayHeight)


	// invoke the callback to build the widgets for the window
	if wnd.OnBuild != nil {
		wnd.OnBuild(wnd)
	}

	// calculate the height all of the controls would need to draw. this can be
	// used to automatically resize the window and will be used to draw a correctly
	// proportioned scroll bar cursor.
	totalControlHeightDC := -wnd.widgetCursorDC[1] + wnd.nextRowCursorOffset + wnd.ScrollOffset
	_, totalControlHeightS := wnd.Owner.DisplayToScreen(0.0, totalControlHeightDC)

	// are we going to fit the height of the window to the height of the controls?
	if wnd.AutoAdjustHeight {
		wnd.Height = totalControlHeightS
	}

	// do we need to roll back the scroll bar change? has it overextended the
	// bounds and need to be pulled back in?
	if wnd.IsScrollable && wnd.ScrollOffset > (totalControlHeightDC - displayHeight) {
		wnd.ScrollOffset = totalControlHeightDC - displayHeight
	}


	// build the frame background for the window including title bar and scroll bar.
	wnd.buildFrame(totalControlHeightDC)

	// next frame we potientially will have a different window location
	// do we need to move the window? (LMB down in a window and mouse dragged)
	if wnd.IsMoveable && lmbDown && wnd.ContainsPosition(mouseX, mouseY) {
		claimed := wnd.Owner.SetActiveInputID(wnd.ID)
		if claimed || wnd.Owner.GetActiveInputID() == wnd.ID {
			// mouse down in the window, lets move the thing before we make the vertices
			deltaXS, deltaYS := wnd.Owner.DisplayToScreen(mouseDeltaX, mouseDeltaY)
			wnd.Location[0] += deltaXS
			wnd.Location[1] += deltaYS
		}
	}
}

// GetDisplaySize returns four values: the x and y positions of the window
// on the screen in display-space and then the width and height of the window
// in display-space values. This does not include space for the scroll bars.
func (wnd *Window) GetDisplaySize() (float32, float32, float32, float32) {
	winxDC, winyDC := wnd.Owner.ScreenToDisplay(wnd.Location[0], wnd.Location[1])
	winwDC, winhDC := wnd.Owner.ScreenToDisplay(wnd.Width, wnd.Height)

	return winxDC, winyDC, winwDC, winhDC
}

// GetFrameSize returns the (x,y) top-left corner of the window in display-space
// coordinates and the width and height of the total window frame as well, including
// the space window decorations take up like titlebar and scrollbar.
func (wnd *Window) GetFrameSize() (float32, float32, float32, float32) {
	style := DefaultStyle
	winxDC, winyDC, winwDC, winhDC := wnd.GetDisplaySize()

	// add in the size of the scroll bar if we're going to show it
	if wnd.ShowScrollBar {
		winwDC += style.ScrollBarWidth
	}

	// add the size of the title bar if it's visible
	if wnd.ShowTitleBar {
		font := wnd.Owner.GetFont(DefaultStyle.FontName)
		if font != nil {
			_, dimY, _ := font.GetRenderSize(wnd.GetTitleString())
			// TODO: for now just add 1 pixel on each side of the string for padding
			winhDC += float32(dimY + 4)
		}
	}
	return winxDC, winyDC, winwDC, winhDC
}

// GetTitleString will return a string with one space in it or the Title property
// if the Title is not an empty string.
func (wnd *Window) GetTitleString() string {
	titleString := " "
	if len(wnd.Title) > 0 {
		titleString = wnd.Title
	}
	return titleString
}

func (wnd *Window) makeCmdList() *cmdList {
	// clip to the frame size which includes space for title bar and scroll bar
	wx, wy, ww, wh := wnd.GetFrameSize()
	cmdList := newCmdList()
	cmdList.clipRect[0] = wx
	cmdList.clipRect[1] = wy
	cmdList.clipRect[2] = ww
	cmdList.clipRect[3] = wh
	return cmdList
}

func (wnd *Window) getFirstCmd() *cmdList {
	if len(wnd.cmds) == 0 {
		// safety first!
		wnd.cmds = []*cmdList{wnd.makeCmdList()}
	}
	return wnd.cmds[0]
}

func (wnd *Window) getLastCmd() *cmdList {
	if len(wnd.cmds) == 0 {
		// safety first!
		wnd.cmds = []*cmdList{wnd.makeCmdList()}
	}
	return wnd.cmds[len(wnd.cmds)-1]
}

// buildFrame builds the background for the window
func (wnd *Window) buildFrame(totalControlHeightDC float32) {
	style := DefaultStyle
	var combos []float32
	var indexes []uint32
	var fc uint32

	// get the first cmdList and insert the frame data into it
	firstCmd := wnd.getFirstCmd()

	// get the dimensions for the window frame
	x, y, w, h := wnd.GetFrameSize()
	titleBarHeight := float32(0.0)

	// if we don't have a title bar, then simply render the background frame
	if wnd.ShowTitleBar {
		// how big should the title bar be?
		titleString := " "
		if len(wnd.Title) > 0 {
			titleString = wnd.Title
		}
		font := wnd.Owner.GetFont(style.FontName)
		_, dimY, _ := font.GetRenderSize(titleString)

		// TODO: for now just add 1 pixel on each side of the string for padding
		titleBarHeight = float32(dimY + 4)

		// render the title bar text
		if len(wnd.Title) > 0 {
			renderData := font.CreateText(mgl.Vec3{x, y, 0}, wnd.TitleBarTextColor, wnd.Title)
			firstCmd.PrefixFaces(renderData.ComboBuffer, renderData.IndexBuffer, renderData.Faces)
		}

		// render the title bar background
		combos, indexes, fc = firstCmd.DrawRectFilledDC(x, y, x+w, y-titleBarHeight, wnd.TitleBarBgColor, wnd.Owner.whitePixelUv)
		firstCmd.PrefixFaces(combos, indexes, fc)

		// render the rest of the window background
		combos, indexes, fc = firstCmd.DrawRectFilledDC(x, y-titleBarHeight, x+w, y-h, wnd.BgColor, wnd.Owner.whitePixelUv)
		firstCmd.PrefixFaces(combos, indexes, fc)
	} else {
		// build the background of the window
		combos, indexes, fc = firstCmd.DrawRectFilledDC(x, y, x+w, y-h, wnd.BgColor, wnd.Owner.whitePixelUv)
		firstCmd.PrefixFaces(combos, indexes, fc)
	}

	if wnd.ShowScrollBar {
		// now add in the scroll bar at the end to overlay everything
		sbX := x+w-style.ScrollBarWidth
		sbY := y-titleBarHeight
		combos, indexes, fc = firstCmd.DrawRectFilledDC(sbX, sbY, x+w, y-h, style.ScrollBarBgColor, wnd.Owner.whitePixelUv)
		firstCmd.AddFaces(combos, indexes, fc)

		// figure out the positioning
		sbCursorWidth := style.ScrollBarCursorWidth
		if sbCursorWidth > style.ScrollBarWidth {
			sbCursorWidth = style.ScrollBarWidth
		}
		sbCursorOffX := (style.ScrollBarWidth - sbCursorWidth) / 2.0

		// calculate the height required for the scrollbar
		sbUsableHeight := h-titleBarHeight
		sbRatio := sbUsableHeight / totalControlHeightDC
		sbCursorHeight := sbUsableHeight * sbRatio

		// move the scroll bar down based on the scroll position
		sbOffY := wnd.ScrollOffset * sbRatio

		// draw the scroll bar cursor
		combos, indexes, fc = firstCmd.DrawRectFilledDC(sbX + sbCursorOffX, sbY-sbOffY, x+w-sbCursorOffX, y-sbOffY-sbCursorHeight, style.ScrollBarCursorColor, wnd.Owner.whitePixelUv)
		firstCmd.AddFaces(combos, indexes, fc)

	}
}

// ContainsPosition returns true if the position passed in is contained within
// the window's space.
func (wnd *Window) ContainsPosition(x, y float32) bool {
	locXDC, locYDC, wndWDC, wndHDC := wnd.GetFrameSize()
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

/*
_    _  _____ ______  _____  _____  _____  _____
| |  | ||_   _||  _  \|  __ \|  ___||_   _|/  ___|
| |  | |  | |  | | | || |  \/| |__    | |  \ `--.
| |/\| |  | |  | | | || | __ |  __|   | |   `--. \
\  /\  / _| |_ | |/ / | |_\ \| |___   | |  /\__/ /
\/  \/  \___/ |___/   \____/\____/   \_/  \____/

*/

// Text renders a text widget
func (wnd *Window) Text(msg string) error {
	style := DefaultStyle
	cmd := wnd.getLastCmd()

	// get the font for the text
	font := wnd.Owner.GetFont(style.FontName)
	if font == nil {
		return fmt.Errorf("Couldn't access font %s from the Manager.", style.FontName)
	}

	// calculate the location for the widget
	pos := wnd.getCursorDC()

	// create the text widget itself
	renderData := font.CreateText(pos, style.TextColor, msg)
	cmd.AddFaces(renderData.ComboBuffer, renderData.IndexBuffer, renderData.Faces)

	// advance the cursor for the width of the text widget
	wnd.widgetCursorDC[0] = wnd.widgetCursorDC[0] + renderData.Width
	wnd.nextRowCursorOffset = renderData.Height

	return nil
}

// Button draws the button widget on screen with the given text.
func (wnd *Window) Button(id string, text string) (bool, error) {
	style := DefaultStyle
	cmd := wnd.getLastCmd()

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
	buttonW := dimX + style.ButtonPadding[0] + style.ButtonPadding[1]
	buttonH := dimY + style.ButtonPadding[2] + style.ButtonPadding[3]

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
			// mouse is down, but was it pressed inside the button?
			mdx, mdy := wnd.Owner.GetMouseDownPosition(0)
			if mdx > pos[0] && mdy > pos[1]-buttonH && mdx < pos[0]+buttonW && mdy < pos[1] {
				bgColor = style.ButtonActiveColor
				buttonPressed = true
				wnd.Owner.SetActiveInputID(id)
			}
		}
	}

	// render the button background
	combos, indexes, fc := cmd.DrawRectFilledDC(pos[0], pos[1], pos[0]+buttonW, pos[1]-buttonH, bgColor, wnd.Owner.whitePixelUv)
	cmd.AddFaces(combos, indexes, fc)

	// create the text for the button
	textPos := pos
	textPos[0] += style.ButtonPadding[0]
	textPos[1] -= style.ButtonPadding[2]
	renderData := font.CreateText(textPos, style.ButtonTextColor, text)
	cmd.AddFaces(renderData.ComboBuffer, renderData.IndexBuffer, renderData.Faces)

	// advance the cursor for the width of the text widget
	wnd.widgetCursorDC[0] = wnd.widgetCursorDC[0] + buttonW + style.ButtonMargin[0] + style.ButtonMargin[1]
	wnd.nextRowCursorOffset = buttonH + style.ButtonMargin[2] + style.ButtonMargin[3]

	return buttonPressed, nil
}

// SliderFloat creates a slider widget that alters a value based on the min/max
// values provided.
func (wnd *Window) SliderFloat(id string, value *float32, min, max float32) error {
	var valueString string
	style := DefaultStyle
	sliderPressed, sliderW, _ := wnd.sliderHitTest(id)

	// we have a mouse down in the widget, so check to see how much the mouse has
	// moved and slide the control cursor and edit the value accordingly.
	if sliderPressed {
		mouseDeltaX, _ := wnd.Owner.GetMousePositionDelta()
		moveRatio := mouseDeltaX / sliderW
		delta := moveRatio * max
		tmp := *value + delta
		if tmp > max {
			tmp = max
		} else if tmp < min {
			tmp = min
		}
		*value = tmp
	}

	// get the position / size for the slider
	cursorRel := *value
	cursorRel = (cursorRel - min) / (max - min)

	valueString = fmt.Sprintf(style.SliderFloatFormat, *value)
	return wnd.sliderBehavior(valueString, cursorRel, true)
}

// SliderInt creates a slider widget that alters a value based on the min/max
// values provided.
func (wnd *Window) SliderInt(id string, value *int, min, max int) error {
	var valueString string
	style := DefaultStyle
	sliderPressed, sliderW, _ := wnd.sliderHitTest(id)

	// we have a mouse down in the widget, so check to see how much the mouse has
	// moved and slide the control cursor and edit the value accordingly.
	if sliderPressed {
		mouseDeltaX, _ := wnd.Owner.GetMousePositionDelta()
		moveRatio := mouseDeltaX / sliderW
		delta := moveRatio * float32(max)
		tmp := int(float32(*value) + delta)
		if tmp > max {
			tmp = max
		} else if tmp < min {
			tmp = min
		}
		*value = tmp
	}

	// get the position / size for the slider
	cursorRel := float32(*value-min) / float32(max-min)

	valueString = fmt.Sprintf(style.SliderIntFormat, *value)
	return wnd.sliderBehavior(valueString, cursorRel, true)
}

// DragSliderInt creates a slider widget that alters a value based on mouse
// movement only.
func (wnd *Window) DragSliderInt(id string, speed float32, value *int) error {
	var valueString string
	style := DefaultStyle
	sliderPressed, _, _ := wnd.sliderHitTest(id)

	// we have a mouse down in the widget, so check to see how much the mouse has
	// moved and slide the control cursor and edit the value accordingly.
	if sliderPressed {
		mouseDeltaX, _ := wnd.Owner.GetMousePositionDelta()
		*value += int(mouseDeltaX * speed)
	}

	valueString = fmt.Sprintf(style.SliderIntFormat, *value)
	return wnd.sliderBehavior(valueString, 0.0, false)
}

// sliderHitTest calculates the size of the widget and then
// returns true if mouse is within the bounding box of this widget;
// as a convenience it also returns the width and height of the control
// as the second and third results respectively.
func (wnd *Window) sliderHitTest(id string) (bool, float32, float32) {
	style := DefaultStyle

	// get the font for the text
	font := wnd.Owner.GetFont(style.FontName)
	if font == nil {
		return false, 0, 0
	}

	// calculate the location for the widget
	pos := wnd.getCursorDC()
	pos[0] += style.SliderMargin[0]
	pos[1] -= style.SliderMargin[2]

	// calculate the size necessary for the widget
	_, _, wndWidth, _ := wnd.GetDisplaySize()
	dimY := float32(font.GlyphHeight) * font.GetCurrentScale()
	sliderW := wndWidth - style.WindowPadding[0] - style.WindowPadding[1] - style.SliderMargin[0] - style.SliderMargin[1]
	sliderH := dimY + style.SliderPadding[2] + style.SliderPadding[3]

	// calculate how much of the slider control is available to the cursor for
	// movement, which affects the scale of the value to edit.
	sliderW = sliderW - style.SliderCursorWidth - style.SliderPadding[0] - style.SliderPadding[1]

	// test to see if the mouse is inside the widget
	lmbStatus := wnd.Owner.GetMouseButtonAction(0)
	if lmbStatus != MouseUp {
		// are  we already the active widget?
		if wnd.Owner.GetActiveInputID() == id {
			return true, sliderW, sliderH
		}

		// try to claim focus
		mx, my := wnd.Owner.GetMouseDownPosition(0)
		if mx > pos[0] && my > pos[1]-sliderH && mx < pos[0]+sliderW && my < pos[1] {
			claimed := wnd.Owner.SetActiveInputID(id)
			if claimed {
				return true, sliderW, sliderH
			}
		}
	}

	return false, sliderW, sliderH
}

// sliderBehavior is the actual action of drawing the slider widget.
func (wnd *Window) sliderBehavior(valueString string, valueRatio float32, drawCursor bool) error {
	style := DefaultStyle
	cmd := wnd.getLastCmd()

	// get the font for the text
	font := wnd.Owner.GetFont(style.FontName)
	if font == nil {
		return fmt.Errorf("Couldn't access font %s from the Manager.", style.FontName)
	}

	// calculate the location for the widget
	pos := wnd.getCursorDC()
	pos[0] += style.SliderMargin[0]
	pos[1] -= style.SliderMargin[2]

	// calculate the size necessary for the widget
	_, _, wndWidth, _ := wnd.GetDisplaySize()
	dimX, dimY, _ := font.GetRenderSize(valueString)
	sliderW := wndWidth - style.WindowPadding[0] - style.WindowPadding[1] - style.SliderMargin[0] - style.SliderMargin[1]
	sliderH := dimY + style.SliderPadding[2] + style.SliderPadding[3]

	// set a default color for the background
	bgColor := style.SliderBgColor

	// render the widget background
	combos, indexes, fc := cmd.DrawRectFilledDC(pos[0], pos[1], pos[0]+sliderW, pos[1]-sliderH, bgColor, wnd.Owner.whitePixelUv)
	cmd.AddFaces(combos, indexes, fc)

	if drawCursor {
		// calculate how much of the slider control is available to the cursor for
		// movement, which affects the scale of the value to edit.
		sliderRangeW := sliderW - style.SliderCursorWidth - style.SliderPadding[0] - style.SliderPadding[1]
		cursorH := sliderH - style.SliderPadding[2] - style.SliderPadding[3]

		// get the position / size for the slider
		cursorPosX := valueRatio*sliderRangeW + style.SliderPadding[0]

		// render the slider cursor
		combos, indexes, fc = cmd.DrawRectFilledDC(pos[0]+cursorPosX, pos[1]-style.SliderPadding[2],
			pos[0]+cursorPosX+style.SliderCursorWidth, pos[1]-cursorH-style.SliderPadding[3], style.SliderCursorColor, wnd.Owner.whitePixelUv)
		cmd.AddFaces(combos, indexes, fc)
	}

	// create the text for the slider
	textPos := pos
	textPos[0] += style.SliderPadding[0] + (0.5 * sliderW) - (0.5 * dimX)
	textPos[1] -= style.SliderPadding[2]
	renderData := font.CreateText(textPos, style.SliderTextColor, valueString)
	cmd.AddFaces(renderData.ComboBuffer, renderData.IndexBuffer, renderData.Faces)

	// advance the cursor for the width of the text widget
	wnd.widgetCursorDC[0] = wnd.widgetCursorDC[0] + sliderW + style.SliderMargin[0] + style.SliderMargin[1]
	wnd.nextRowCursorOffset = sliderH + style.SliderMargin[2] + style.SliderMargin[3]

	return nil
}
