// Copyright 2016, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package eweygewey

import (
	"fmt"

	mgl "github.com/go-gl/mathgl/mgl32"
)

const (
	defaultTextureSampler = uint32(0)
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

	// Style is the set of visual parameters to use when drawing this window.
	Style

	// widgetCursorDC is the current location to insert widgets and should
	// be updated after adding new widgets. This is specified in display
	// coordinates.
	widgetCursorDC mgl.Vec3

	// nextRowCursorOffsetDC is the value the widgetCursorDC's y component
	// should change for the next widget that starts a new row in the window.
	nextRowCursorOffsetDC float32

	// requestedItemWidthMinDC is set by the client code to adjust the width of the
	// next control to be at least a specific size.
	requestedItemWidthMinDC float32

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
	wnd.OnBuild = constructor
	wnd.ShowTitleBar = true
	wnd.IsMoveable = true
	//wnd.IsScrollable = false
	wnd.Style = DefaultStyle
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
	wnd.widgetCursorDC = mgl.Vec3{wnd.Style.WindowPadding[0], wnd.ScrollOffset, 0}
	wnd.nextRowCursorOffsetDC = 0

	// advance the cursor to account for the title bar
	_, _, _, frameHeight := wnd.GetFrameSize()
	_, _, _, displayHeight := wnd.GetDisplaySize()
	wnd.widgetCursorDC[1] = wnd.widgetCursorDC[1] - (frameHeight - displayHeight) - wnd.WindowPadding[2]

	// invoke the callback to build the widgets for the window
	if wnd.OnBuild != nil {
		wnd.OnBuild(wnd)
	}

	// calculate the height all of the controls would need to draw. this can be
	// used to automatically resize the window and will be used to draw a correctly
	// proportioned scroll bar cursor.
	totalControlHeightDC := -wnd.widgetCursorDC[1] + wnd.nextRowCursorOffsetDC + wnd.ScrollOffset + wnd.WindowPadding[3]
	_, totalControlHeightS := wnd.Owner.DisplayToScreen(0.0, totalControlHeightDC)

	// are we going to fit the height of the window to the height of the controls?
	if wnd.AutoAdjustHeight {
		wnd.Height = totalControlHeightS
	}

	// do we need to roll back the scroll bar change? has it overextended the
	// bounds and need to be pulled back in?
	if wnd.IsScrollable && wnd.ScrollOffset > (totalControlHeightDC-displayHeight) {
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
	winxDC, winyDC, winwDC, winhDC := wnd.GetDisplaySize()

	// add in the size of the scroll bar if we're going to show it
	if wnd.ShowScrollBar {
		winwDC += wnd.Style.ScrollBarWidth
	}

	// add the size of the title bar if it's visible
	if wnd.ShowTitleBar {
		font := wnd.Owner.GetFont(wnd.Style.FontName)
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
		font := wnd.Owner.GetFont(wnd.Style.FontName)
		_, dimY, _ := font.GetRenderSize(titleString)

		// TODO: for now just add 1 pixel on each side of the string for padding
		titleBarHeight = float32(dimY + 4)

		// render the title bar background
		combos, indexes, fc = firstCmd.DrawRectFilledDC(x, y, x+w, y-titleBarHeight, wnd.Style.TitleBarBgColor, defaultTextureSampler, wnd.Owner.whitePixelUv)
		firstCmd.AddFaces(combos, indexes, fc)

		// render the title bar text
		if len(wnd.Title) > 0 {
			renderData := font.CreateText(mgl.Vec3{x + wnd.Style.WindowPadding[0], y, 0}, wnd.Style.TitleBarTextColor, wnd.Title)
			firstCmd.AddFaces(renderData.ComboBuffer, renderData.IndexBuffer, renderData.Faces)
		}

		// render the rest of the window background
		combos, indexes, fc = firstCmd.DrawRectFilledDC(x, y-titleBarHeight, x+w, y-h, wnd.Style.WindowBgColor, defaultTextureSampler, wnd.Owner.whitePixelUv)
		firstCmd.PrefixFaces(combos, indexes, fc)
	} else {
		// build the background of the window
		combos, indexes, fc = firstCmd.DrawRectFilledDC(x, y, x+w, y-h, wnd.Style.WindowBgColor, defaultTextureSampler, wnd.Owner.whitePixelUv)
		firstCmd.PrefixFaces(combos, indexes, fc)
	}

	if wnd.ShowScrollBar {
		// now add in the scroll bar at the end to overlay everything
		sbX := x + w - wnd.Style.ScrollBarWidth
		sbY := y - titleBarHeight
		combos, indexes, fc = firstCmd.DrawRectFilledDC(sbX, sbY, x+w, y-h, wnd.Style.ScrollBarBgColor, defaultTextureSampler, wnd.Owner.whitePixelUv)
		firstCmd.AddFaces(combos, indexes, fc)

		// figure out the positioning
		sbCursorWidth := wnd.Style.ScrollBarCursorWidth
		if sbCursorWidth > wnd.Style.ScrollBarWidth {
			sbCursorWidth = wnd.Style.ScrollBarWidth
		}
		sbCursorOffX := (wnd.Style.ScrollBarWidth - sbCursorWidth) / 2.0

		// calculate the height required for the scrollbar
		sbUsableHeight := h - titleBarHeight
		sbRatio := sbUsableHeight / totalControlHeightDC

		// if we have more usable height than controls, just make the scrollbar
		// take up the whole space.
		if sbRatio >= 1.0 {
			sbRatio = 1.0
		}

		sbCursorHeight := sbUsableHeight * sbRatio

		// move the scroll bar down based on the scroll position
		sbOffY := wnd.ScrollOffset * sbRatio

		// draw the scroll bar cursor
		combos, indexes, fc = firstCmd.DrawRectFilledDC(sbX+sbCursorOffX, sbY-sbOffY, x+w-sbCursorOffX, sbY-sbOffY-sbCursorHeight, wnd.Style.ScrollBarCursorColor,
			defaultTextureSampler, wnd.Owner.whitePixelUv)
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
	wnd.widgetCursorDC[0] = wnd.Style.WindowPadding[0]
	wnd.widgetCursorDC[1] = wnd.widgetCursorDC[1] - wnd.nextRowCursorOffsetDC
}

// getCursorDC returns the current cursor offset as an absolute location
// in the user interface.
func (wnd *Window) getCursorDC() mgl.Vec3 {
	// start with the widget DC offet
	pos := wnd.widgetCursorDC

	// add in the position of the window in pixels
	windowDx, windowDy := wnd.Owner.ScreenToDisplay(wnd.Location[0], wnd.Location[1])
	pos[0] += windowDx
	pos[1] += windowDy

	return pos
}

// RequestItemWidthMin will request the window to draw the next widget with the
// specified window-normalized size (e.g. if Window's width is 500 px, then passing
// 0.25 here translates to 125 px).
func (wnd *Window) RequestItemWidthMin(nextMinWS float32) {
	// clip the incoming value
	reqMin := ClipF32(0.0, 1.0, nextMinWS)

	// calc the amount of window width we're requesting
	_, _, wndW, _ := wnd.GetDisplaySize()

	// convert this to display space
	wnd.requestedItemWidthMinDC = reqMin * wndW
}

// addCursorHorizontalDelta sets the amount the cursor will to change
// laterally based on wether or not the client code requested a minimum size.
func (wnd *Window) addCursorHorizontalDelta(hWidth float32) {
	// we have request, so expand the width if necessary
	if wnd.requestedItemWidthMinDC > 0.0 {
		if wnd.requestedItemWidthMinDC > hWidth {
			hWidth = wnd.requestedItemWidthMinDC
		}

		// reset the request to make it a one-off operation
		wnd.requestedItemWidthMinDC = 0.0
	}

	wnd.widgetCursorDC[0] += hWidth
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
	cmd := wnd.getLastCmd()

	// get the font for the text
	font := wnd.Owner.GetFont(wnd.Style.FontName)
	if font == nil {
		return fmt.Errorf("Couldn't access font %s from the Manager.", wnd.Style.FontName)
	}

	// calculate the location for the widget
	pos := wnd.getCursorDC()

	// create the text widget itself
	renderData := font.CreateText(pos, wnd.Style.TextColor, msg)
	cmd.AddFaces(renderData.ComboBuffer, renderData.IndexBuffer, renderData.Faces)

	// advance the cursor for the width of the text widget
	wnd.addCursorHorizontalDelta(renderData.Width + wnd.Style.TextMargin[0] + wnd.Style.TextMargin[1])
	wnd.nextRowCursorOffsetDC = renderData.Height + wnd.Style.TextMargin[2] + wnd.Style.TextMargin[3]

	return nil
}

// Button draws the button widget on screen with the given text.
func (wnd *Window) Button(id string, text string) (bool, error) {
	cmd := wnd.getLastCmd()

	// get the font for the text
	font := wnd.Owner.GetFont(wnd.Style.FontName)
	if font == nil {
		return false, fmt.Errorf("Couldn't access font %s from the Manager.", wnd.Style.FontName)
	}

	// calculate the location for the widget
	pos := wnd.getCursorDC()
	pos[0] += wnd.Style.ButtonMargin[0]
	pos[1] -= wnd.Style.ButtonMargin[2]

	// calculate the size necessary for the widget
	dimX, dimY, _ := font.GetRenderSize(text)
	buttonW := dimX + wnd.Style.ButtonPadding[0] + wnd.Style.ButtonPadding[1]
	buttonH := dimY + wnd.Style.ButtonPadding[2] + wnd.Style.ButtonPadding[3]

	// set a default color for the button
	bgColor := wnd.Style.ButtonColor
	buttonPressed := false

	// test to see if the mouse is inside the widget
	mx, my := wnd.Owner.GetMousePosition()
	if mx > pos[0] && my > pos[1]-buttonH && mx < pos[0]+buttonW && my < pos[1] {
		lmbStatus := wnd.Owner.GetMouseButtonAction(0)
		if lmbStatus == MouseUp {
			bgColor = wnd.Style.ButtonHoverColor
		} else {
			// mouse is down, but was it pressed inside the button?
			mdx, mdy := wnd.Owner.GetMouseDownPosition(0)
			if mdx > pos[0] && mdy > pos[1]-buttonH && mdx < pos[0]+buttonW && mdy < pos[1] {
				bgColor = wnd.Style.ButtonActiveColor
				buttonPressed = true
				wnd.Owner.SetActiveInputID(id)
			}
		}
	}

	// render the button background
	combos, indexes, fc := cmd.DrawRectFilledDC(pos[0], pos[1], pos[0]+buttonW, pos[1]-buttonH, bgColor, defaultTextureSampler, wnd.Owner.whitePixelUv)
	cmd.AddFaces(combos, indexes, fc)

	// create the text for the button
	textPos := pos
	textPos[0] += wnd.Style.ButtonPadding[0]
	textPos[1] -= wnd.Style.ButtonPadding[2]
	renderData := font.CreateText(textPos, wnd.Style.ButtonTextColor, text)
	cmd.AddFaces(renderData.ComboBuffer, renderData.IndexBuffer, renderData.Faces)

	// advance the cursor for the width of the text widget
	wnd.addCursorHorizontalDelta(buttonW + wnd.Style.ButtonMargin[0] + wnd.Style.ButtonMargin[1])
	wnd.nextRowCursorOffsetDC = buttonH + wnd.Style.ButtonMargin[2] + wnd.Style.ButtonMargin[3]

	return buttonPressed, nil
}

// SliderFloat creates a slider widget that alters a value based on the min/max
// values provided.
func (wnd *Window) SliderFloat(id string, value *float32, min, max float32) error {
	var valueString string
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

	valueString = fmt.Sprintf(wnd.Style.SliderFloatFormat, *value)
	return wnd.sliderBehavior(valueString, cursorRel, true)
}

// SliderInt creates a slider widget that alters a value based on the min/max
// values provided.
func (wnd *Window) SliderInt(id string, value *int, min, max int) error {
	var valueString string
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

	valueString = fmt.Sprintf(wnd.Style.SliderIntFormat, *value)
	return wnd.sliderBehavior(valueString, cursorRel, true)
}

// DragSliderInt creates a slider widget that alters a value based on mouse
// movement only.
func (wnd *Window) DragSliderInt(id string, speed float32, value *int) error {
	var valueString string
	sliderPressed, _, _ := wnd.sliderHitTest(id)

	// we have a mouse down in the widget, so check to see how much the mouse has
	// moved and slide the control cursor and edit the value accordingly.
	if sliderPressed {
		mouseDeltaX, _ := wnd.Owner.GetMousePositionDelta()
		*value += int(mouseDeltaX * speed)
	}

	valueString = fmt.Sprintf(wnd.Style.SliderIntFormat, *value)
	return wnd.sliderBehavior(valueString, 0.0, false)
}

// DragSliderFloat creates a slider widget that alters a value based on mouse
// movement only.
func (wnd *Window) DragSliderFloat(id string, speed float32, value *float32) error {
	var valueString string
	sliderPressed, _, _ := wnd.sliderHitTest(id)

	// we have a mouse down in the widget, so check to see how much the mouse has
	// moved and slide the control cursor and edit the value accordingly.
	if sliderPressed {
		mouseDeltaX, _ := wnd.Owner.GetMousePositionDelta()
		*value += mouseDeltaX * speed
	}

	valueString = fmt.Sprintf(wnd.Style.SliderFloatFormat, *value)
	return wnd.sliderBehavior(valueString, 0.0, false)
}

// sliderHitTest calculates the size of the widget and then
// returns true if mouse is within the bounding box of this widget;
// as a convenience it also returns the width and height of the control
// as the second and third results respectively.
func (wnd *Window) sliderHitTest(id string) (bool, float32, float32) {
	// get the font for the text
	font := wnd.Owner.GetFont(wnd.Style.FontName)
	if font == nil {
		return false, 0, 0
	}

	// calculate the location for the widget
	pos := wnd.getCursorDC()
	pos[0] += wnd.Style.SliderMargin[0]
	pos[1] -= wnd.Style.SliderMargin[2]

	// calculate the size necessary for the widget
	_, _, wndWidth, _ := wnd.GetDisplaySize()
	dimY := float32(font.GlyphHeight) * font.GetCurrentScale()
	sliderW := wndWidth - wnd.Style.WindowPadding[0] - wnd.Style.WindowPadding[1] - wnd.Style.SliderMargin[0] - wnd.Style.SliderMargin[1]
	sliderH := dimY + wnd.Style.SliderPadding[2] + wnd.Style.SliderPadding[3]

	// calculate how much of the slider control is available to the cursor for
	// movement, which affects the scale of the value to edit.
	sliderW = sliderW - wnd.Style.SliderCursorWidth - wnd.Style.SliderPadding[0] - wnd.Style.SliderPadding[1]

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
	cmd := wnd.getLastCmd()

	// get the font for the text
	font := wnd.Owner.GetFont(wnd.Style.FontName)
	if font == nil {
		return fmt.Errorf("Couldn't access font %s from the Manager.", wnd.Style.FontName)
	}

	// calculate the location for the widget
	pos := wnd.getCursorDC()
	pos[0] += wnd.Style.SliderMargin[0]
	pos[1] -= wnd.Style.SliderMargin[2]

	// calculate the size necessary for the widget
	_, _, wndWidth, _ := wnd.GetDisplaySize()
	dimX, dimY, _ := font.GetRenderSize(valueString)
	sliderW := wndWidth - wnd.widgetCursorDC[0] - wnd.Style.WindowPadding[1] - wnd.Style.SliderMargin[1]
	sliderH := dimY + wnd.Style.SliderPadding[2] + wnd.Style.SliderPadding[3]

	// set a default color for the background
	bgColor := wnd.Style.SliderBgColor

	// render the widget background
	combos, indexes, fc := cmd.DrawRectFilledDC(pos[0], pos[1], pos[0]+sliderW, pos[1]-sliderH, bgColor, defaultTextureSampler, wnd.Owner.whitePixelUv)
	cmd.AddFaces(combos, indexes, fc)

	if drawCursor {
		// calculate how much of the slider control is available to the cursor for
		// movement, which affects the scale of the value to edit.
		sliderRangeW := sliderW - wnd.Style.SliderCursorWidth - wnd.Style.SliderPadding[0] - wnd.Style.SliderPadding[1]
		cursorH := sliderH - wnd.Style.SliderPadding[2] - wnd.Style.SliderPadding[3]

		// get the position / size for the slider
		cursorPosX := valueRatio*sliderRangeW + wnd.Style.SliderPadding[0]

		// render the slider cursor
		combos, indexes, fc = cmd.DrawRectFilledDC(pos[0]+cursorPosX, pos[1]-wnd.Style.SliderPadding[2],
			pos[0]+cursorPosX+wnd.Style.SliderCursorWidth, pos[1]-cursorH-wnd.Style.SliderPadding[3],
			wnd.Style.SliderCursorColor, defaultTextureSampler, wnd.Owner.whitePixelUv)
		cmd.AddFaces(combos, indexes, fc)
	}

	// create the text for the slider
	textPos := pos
	textPos[0] += wnd.Style.SliderPadding[0] + (0.5 * sliderW) - (0.5 * dimX)
	textPos[1] -= wnd.Style.SliderPadding[2]
	renderData := font.CreateText(textPos, wnd.Style.SliderTextColor, valueString)
	cmd.AddFaces(renderData.ComboBuffer, renderData.IndexBuffer, renderData.Faces)

	// advance the cursor for the width of the text widget
	wnd.addCursorHorizontalDelta(sliderW + wnd.Style.SliderMargin[0] + wnd.Style.SliderMargin[1])
	wnd.nextRowCursorOffsetDC = sliderH + wnd.Style.SliderMargin[2] + wnd.Style.SliderMargin[3]

	return nil
}

// Image draws the image widget on screen.
func (wnd *Window) Image(id string, widthS, heightS float32, color mgl.Vec4, textureIndex uint32, uvPair mgl.Vec4) error {
	cmd := wnd.getLastCmd()

	// get the font for the text
	font := wnd.Owner.GetFont(wnd.Style.FontName)
	if font == nil {
		return fmt.Errorf("Couldn't access font %s from the Manager.", wnd.Style.FontName)
	}

	// calculate the location for the widget
	pos := wnd.getCursorDC()
	pos[0] += wnd.Style.ImageMargin[0]
	pos[1] += wnd.Style.ImageMargin[2]
	widthDC, heightDC := wnd.Owner.ScreenToDisplay(widthS, heightS)

	// render the button background
	combos, indexes, fc := cmd.DrawRectFilledDC(pos[0], pos[1], pos[0]+widthDC, pos[1]-heightDC, color, textureIndex, uvPair)
	cmd.AddFaces(combos, indexes, fc)

	// advance the cursor for the width of the text widget
	wnd.addCursorHorizontalDelta(widthDC + wnd.Style.ImageMargin[0] + wnd.Style.ImageMargin[1])
	wnd.nextRowCursorOffsetDC = heightDC + wnd.Style.ImageMargin[2] + wnd.Style.ImageMargin[3]

	return nil
}

// Separator draws a separator rectangle and advances the cursor to a new row automatically.
func (wnd *Window) Separator() {
	wnd.StartRow()
	cmd := wnd.getLastCmd()

	// calculate the location for the widget
	pos := wnd.getCursorDC()
	pos[0] += wnd.Style.SeparatorMargin[0]
	pos[1] -= wnd.Style.SeparatorMargin[2]

	_, _, widthDC, _ := wnd.GetDisplaySize()
	widthDC += -wnd.Style.SeparatorMargin[0] - wnd.Style.SeparatorMargin[1] - wnd.Style.WindowPadding[0] - wnd.Style.WindowPadding[1]

	// draw the separator
	combos, indexes, fc := cmd.DrawRectFilledDC(pos[0], pos[1], pos[0]+widthDC, pos[1]-wnd.Style.SeparatorHeight, wnd.Style.SeparatorColor, defaultTextureSampler, wnd.Owner.whitePixelUv)
	cmd.AddFaces(combos, indexes, fc)

	// start a new row
	wnd.nextRowCursorOffsetDC = wnd.Style.SeparatorHeight + wnd.Style.SeparatorMargin[2] + wnd.Style.SeparatorMargin[3]
	wnd.StartRow()
}
