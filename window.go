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

	// requestedItemWidthMaxDC is set by the client code to adjust the width of the
	// next control to be at most a specific size.
	requestedItemWidthMaxDC float32

	// indentLevel is the number of indents that each row should start off with.
	// this means the new row should have a widgetCursorDC that is offset the
	// amount of (Style.IndentSpacing * indentLevel).
	indentLevel int

	// cmds is the slice of cmdLists used to to render the window
	cmds []*cmdList

	// intStorage is a map that allows an int to be stored by string -- typically
	// an ID from a widget as a key.
	intStorage map[string]int
}

// newWindow creates a new window with a top-left coordinate of (x,y) and
// dimensions of (w,h).
func newWindow(id string, x, y, w, h float32, constructor BuildCallback) *Window {
	wnd := new(Window)
	wnd.cmds = []*cmdList{}
	wnd.intStorage = make(map[string]int)
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
	// bounds and need to be pulled back in? make sure that the total control
	// height is actually greter than display height and requires scrolling first.
	controlHeightOverflow := totalControlHeightDC - displayHeight
	if wnd.IsScrollable && controlHeightOverflow > 0 && wnd.ScrollOffset > controlHeightOverflow {
		wnd.ScrollOffset = controlHeightOverflow
	} else if controlHeightOverflow < 0 {
		// more space then needed so reset the scroll bar
		wnd.ScrollOffset = 0
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

// GetAspectRatio returns the aspect ratio of the window (width / height)
func (wnd *Window) GetAspectRatio() float32 {
	return wnd.Width / wnd.Height
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
			winhDC += float32(dimY) + wnd.Style.TitleBarPadding[2] + wnd.Style.TitleBarPadding[3]
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

// getFirstCmd will return the first non-custom cmdList; if the first cmdList
// is custom, it makes a new one.
func (wnd *Window) getFirstCmd() *cmdList {
	// empty list
	if len(wnd.cmds) == 0 {
		wnd.cmds = []*cmdList{wnd.makeCmdList()}
	}

	// if the first cmd is custom, then insert a new one
	if wnd.cmds[0].isCustom {
		newCmd := wnd.makeCmdList()
		newSlice := []*cmdList{}
		newSlice = append(newSlice, newCmd)
		newSlice = append(newSlice, wnd.cmds...)
		wnd.cmds = newSlice
	}

	return wnd.cmds[0]
}

// getLastCmd will return the last non-custom cmdList
func (wnd *Window) getLastCmd() *cmdList {
	// empty list
	if len(wnd.cmds) == 0 {
		wnd.cmds = []*cmdList{wnd.makeCmdList()}
	}

	// we don't want to add to the custom draw command
	if wnd.cmds[len(wnd.cmds)-1].isCustom {
		return wnd.addNewCmd()
	}

	// just return the last cmdList
	return wnd.cmds[len(wnd.cmds)-1]
}

// addNewCmd creates a new cmdList and adds it to the window's slice of cmlLists.
func (wnd *Window) addNewCmd() *cmdList {
	if len(wnd.cmds) == 0 {
		return wnd.getFirstCmd()
	}
	newCmd := wnd.makeCmdList()
	wnd.cmds = append(wnd.cmds, newCmd)
	return newCmd
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

		titleBarHeight = float32(dimY) + wnd.Style.TitleBarPadding[2] + wnd.Style.TitleBarPadding[3]
		titleBarTextPos := mgl.Vec3{
			x + wnd.Style.TitleBarPadding[0],
			y - wnd.Style.TitleBarPadding[2],
			0}

		// render the title bar background
		combos, indexes, fc = firstCmd.DrawRectFilledDC(x, y, x+w, y-titleBarHeight, wnd.Style.TitleBarBgColor, defaultTextureSampler, wnd.Owner.whitePixelUv)
		firstCmd.AddFaces(combos, indexes, fc)

		// render the title bar text
		if len(wnd.Title) > 0 {
			renderData := font.CreateText(titleBarTextPos, wnd.Style.TitleBarTextColor, wnd.Title)
			firstCmd.AddFaces(renderData.ComboBuffer, renderData.IndexBuffer, renderData.Faces)
		}

		// render the rest of the window background
		combos, indexes, fc = firstCmd.DrawRectFilledDC(x, y-titleBarHeight, x+w, y-h-titleBarHeight, wnd.Style.WindowBgColor, defaultTextureSampler, wnd.Owner.whitePixelUv)
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
	wnd.widgetCursorDC[0] = wnd.Style.WindowPadding[0] + wnd.Style.IndentSpacing*float32(wnd.indentLevel)
	wnd.widgetCursorDC[1] = wnd.widgetCursorDC[1] - wnd.nextRowCursorOffsetDC

	// clear out the next row height offset
	wnd.nextRowCursorOffsetDC = 0.0
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
	unpaddedWndW := wndW - wnd.Style.WindowPadding[0] - wnd.Style.WindowPadding[1]

	// convert this to display space
	wnd.requestedItemWidthMinDC = reqMin * unpaddedWndW

	// clip the request to window size left
	if wnd.widgetCursorDC[0]+wnd.requestedItemWidthMinDC > unpaddedWndW {
		wnd.requestedItemWidthMinDC = unpaddedWndW - wnd.widgetCursorDC[0]
	}
}

// RequestItemWidthMax will request the window to draw the next widget with at most the
// specified window-normalized size (e.g. if Window's width is 500 px, then passing
// 0.25 here translates to 125 px).
func (wnd *Window) RequestItemWidthMax(nextMaxWS float32) {
	// clip the incoming value
	reqMax := ClipF32(0.0, 1.0, nextMaxWS)

	// calc the amount of window width we're requesting
	_, _, wndW, _ := wnd.GetDisplaySize()
	unpaddedWndW := wndW - wnd.Style.WindowPadding[0] - wnd.Style.WindowPadding[1]

	// convert this to display space
	wnd.requestedItemWidthMaxDC = reqMax * unpaddedWndW

	// clip the request to window size left
	if wnd.widgetCursorDC[0]+wnd.requestedItemWidthMaxDC > unpaddedWndW {
		wnd.requestedItemWidthMaxDC = unpaddedWndW - wnd.widgetCursorDC[0]
	}
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
	if wnd.requestedItemWidthMaxDC > 0.0 {
		if wnd.requestedItemWidthMaxDC < hWidth {
			hWidth = wnd.requestedItemWidthMaxDC
		}

		// reset the request to make it a one-off operation
		wnd.requestedItemWidthMaxDC = 0.0
	}

	wnd.widgetCursorDC[0] += hWidth
}

// setNextRowCursorOffset specifies how much to change the cursor position
// when a new row is started.
func (wnd *Window) setNextRowCursorOffset(offset float32) {
	// only set the next row offset if the one being passed in is greater
	// than the offset recorded by other widgets
	if offset > wnd.nextRowCursorOffsetDC {
		wnd.nextRowCursorOffsetDC = offset
	}
}

// clampWidgetWidthToReqW clamps the incoming width of a widget to the requested
// min and max values if they are set.
func (wnd *Window) clampWidgetWidthToReqW(widthDC float32) float32 {
	result := widthDC

	if wnd.requestedItemWidthMinDC > 0.0 && wnd.requestedItemWidthMinDC > widthDC {
		result = wnd.requestedItemWidthMinDC
	}
	if wnd.requestedItemWidthMaxDC > 0.0 && wnd.requestedItemWidthMaxDC < widthDC {
		result = wnd.requestedItemWidthMaxDC
	}

	return result
}

// Indent increases the indent level in the window, which also immediately changes
// the widgetCursorDC value.
func (wnd *Window) Indent() {
	wnd.indentLevel++
	wnd.widgetCursorDC[0] += wnd.Style.IndentSpacing
}

// Unindent decreases the indent level in the window, which also immediately changes
// the widgetCursorDC value.
func (wnd *Window) Unindent() {
	wnd.indentLevel--
	if wnd.indentLevel < 0 {
		wnd.indentLevel = 0
	}
	wnd.widgetCursorDC[0] -= wnd.Style.IndentSpacing
}

// getStoredInt will return an int and a bool indicating if key was present.
func (wnd *Window) getStoredInt(key string) (int, bool) {
	val, okay := wnd.intStorage[key]
	return val, okay
}

// setStoredInt stores an int value for a given key; returns the previous value
// and a bool indicating if a value was set previously.
func (wnd *Window) setStoredInt(key string, value int) (int, bool) {
	oldValue, present := wnd.intStorage[key]
	wnd.intStorage[key] = value
	return oldValue, present
}

/* *****************************************************************************************************************************************************
_    _  _____ ______  _____  _____  _____  _____
| |  | ||_   _||  _  \|  __ \|  ___||_   _|/  ___|
| |  | |  | |  | | | || |  \/| |__    | |  \ `--.
| |/\| |  | |  | | | || | __ |  __|   | |   `--. \
\  /\  / _| |_ | |/ / | |_\ \| |___   | |  /\__/ /
\/  \/  \___/ |___/   \____/\____/   \_/  \____/

***************************************************************************************************************************************************** */

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
	pos[0] += wnd.Style.TextMargin[0]
	pos[1] -= wnd.Style.TextMargin[2]

	// create the text widget itself
	renderData := font.CreateText(pos, wnd.Style.TextColor, msg)
	cmd.AddFaces(renderData.ComboBuffer, renderData.IndexBuffer, renderData.Faces)

	// advance the cursor for the width of the text widget
	wnd.addCursorHorizontalDelta(renderData.Width + wnd.Style.TextMargin[0] + wnd.Style.TextMargin[1])
	wnd.setNextRowCursorOffset(renderData.Height + wnd.Style.TextMargin[2] + wnd.Style.TextMargin[3])

	return nil
}

// Checkbox draws the checkbox widget on screen.
func (wnd *Window) Checkbox(id string, value *bool) (bool, error) {
	cmd := wnd.getLastCmd()

	// calculate the location for the widget
	pos := wnd.getCursorDC()
	pos[0] += wnd.Style.CheckboxMargin[0]
	pos[1] -= wnd.Style.CheckboxMargin[2]

	// calculate the size necessary for the widget
	//screenW, screenH := wnd.Owner.GetResolution()
	checkW := wnd.Style.CheckboxPadding[0] + wnd.Style.CheckboxPadding[1] + wnd.Style.CheckboxCursorWidth
	checkH := wnd.Style.CheckboxPadding[2] + wnd.Style.CheckboxPadding[3] + wnd.Style.CheckboxCursorWidth

	// clamp the width of the widget to respect any requests to size
	checkW = wnd.clampWidgetWidthToReqW(checkW)

	// set a default color for the button
	bgColor := wnd.Style.CheckboxColor
	pressed := false

	// test to see if the mouse is inside the widget
	buttonTest := wnd.buttonBehavior(id, pos[0], pos[1], checkW, checkH)
	if buttonTest == buttonPressed {
		pressed = true
		*value = !(*value)
	}

	// render the widget background
	combos, indexes, fc := cmd.DrawRectFilledDC(pos[0], pos[1], pos[0]+checkW, pos[1]-checkH, bgColor, defaultTextureSampler, wnd.Owner.whitePixelUv)
	cmd.AddFaces(combos, indexes, fc)

	// do we show the check in the checkbox
	if *value {
		// render the checkbox cursor
		combos, indexes, fc = cmd.DrawRectFilledDC(
			pos[0]+wnd.Style.CheckboxPadding[0],
			pos[1]-wnd.Style.CheckboxPadding[2],
			pos[0]+checkW-wnd.Style.CheckboxPadding[1], //-wnd.Style.CheckboxPadding[1],
			pos[1]-checkH+wnd.Style.CheckboxPadding[3], //+wnd.Style.CheckboxPadding[3],
			wnd.Style.CheckboxCheckColor, defaultTextureSampler, wnd.Owner.whitePixelUv)
		cmd.AddFaces(combos, indexes, fc)
	}

	// advance the cursor for the width of the text widget
	wnd.addCursorHorizontalDelta(checkW + wnd.Style.CheckboxMargin[0] + wnd.Style.CheckboxMargin[1])
	wnd.setNextRowCursorOffset(checkH + wnd.Style.CheckboxMargin[2] + wnd.Style.CheckboxMargin[3])

	// if we've captured the mouse click event and registered a button press, clear
	// the tracking data for the mouse button so that we don't get duplicate matches.
	if pressed {
		wnd.Owner.ClearMouseButtonAction(0)
	}

	return pressed, nil
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
	buttonW := dimX + wnd.Style.ButtonPadding[0] + wnd.Style.ButtonPadding[1] +
		wnd.Style.ButtonMargin[0] + wnd.Style.ButtonMargin[1]
	buttonH := dimY + wnd.Style.ButtonPadding[2] + wnd.Style.ButtonPadding[3]

	// clamp the width of the widget to respect any requests to size
	buttonW = wnd.clampWidgetWidthToReqW(buttonW + wnd.Style.ButtonMargin[0] + wnd.Style.ButtonMargin[1])
	buttonW = buttonW - wnd.Style.ButtonMargin[0] - wnd.Style.ButtonMargin[1]

	// set a default color for the button
	bgColor := wnd.Style.ButtonColor
	pressed := false

	// test to see if the mouse is inside the widget
	buttonTest := wnd.buttonBehavior(id, pos[0], pos[1], buttonW, buttonH)
	if buttonTest == buttonPressed {
		pressed = true
	} else if buttonTest == buttonHover {
		bgColor = wnd.Style.ButtonHoverColor
	}

	// render the button background
	combos, indexes, fc := cmd.DrawRectFilledDC(pos[0], pos[1], pos[0]+buttonW, pos[1]-buttonH, bgColor, defaultTextureSampler, wnd.Owner.whitePixelUv)
	cmd.AddFaces(combos, indexes, fc)

	// create the text for the button
	centerTextX := (buttonW - dimX) / 2.0
	textPos := pos
	textPos[0] = textPos[0] + centerTextX
	textPos[1] = textPos[1] - wnd.Style.ButtonPadding[2]
	renderData := font.CreateText(textPos, wnd.Style.ButtonTextColor, text)
	cmd.AddFaces(renderData.ComboBuffer, renderData.IndexBuffer, renderData.Faces)

	// advance the cursor for the width of the text widget
	wnd.addCursorHorizontalDelta(buttonW + wnd.Style.ButtonMargin[0] + wnd.Style.ButtonMargin[1])
	wnd.setNextRowCursorOffset(buttonH + wnd.Style.ButtonMargin[2] + wnd.Style.ButtonMargin[3])

	// if we've captured the mouse click event and registered a button press, clear
	// the tracking data for the mouse button so that we don't get duplicate matches.
	if pressed {
		wnd.Owner.ClearMouseButtonAction(0)
	}

	return pressed, nil
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

// DragSliderUInt creates a slider widget that alters a value based on mouse
// movement only.
func (wnd *Window) DragSliderUInt(id string, speed float32, value *uint) error {
	var valueString string
	sliderPressed, _, _ := wnd.sliderHitTest(id)

	// we have a mouse down in the widget, so check to see how much the mouse has
	// moved and slide the control cursor and edit the value accordingly.
	if sliderPressed {
		mouseDeltaX, _ := wnd.Owner.GetMousePositionDelta()
		if int(mouseDeltaX*speed)+int(*value) >= 0 {
			*value += uint(mouseDeltaX * speed)
		}
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

// DragSliderUFloat creates a slider widget that alters a value based on mouse
// movement only.
func (wnd *Window) DragSliderUFloat(id string, speed float32, value *float32) error {
	var valueString string
	sliderPressed, _, _ := wnd.sliderHitTest(id)

	// we have a mouse down in the widget, so check to see how much the mouse has
	// moved and slide the control cursor and edit the value accordingly.
	if sliderPressed {
		mouseDeltaX, _ := wnd.Owner.GetMousePositionDelta()
		*value += mouseDeltaX * speed
		if *value < 0.0 {
			*value = 0.0
		}
	}

	valueString = fmt.Sprintf(wnd.Style.SliderFloatFormat, *value)
	return wnd.sliderBehavior(valueString, 0.0, false)
}

// DragSliderFloat64 creates a slider widget that alters a value based on mouse
// movement only.
func (wnd *Window) DragSliderFloat64(id string, speed float64, value *float64) error {
	var valueString string
	sliderPressed, _, _ := wnd.sliderHitTest(id)

	// we have a mouse down in the widget, so check to see how much the mouse has
	// moved and slide the control cursor and edit the value accordingly.
	if sliderPressed {
		mouseDeltaX, _ := wnd.Owner.GetMousePositionDelta()
		*value += float64(mouseDeltaX) * speed
	}

	valueString = fmt.Sprintf(wnd.Style.SliderFloatFormat, *value)
	return wnd.sliderBehavior(valueString, 0.0, false)
}

// DragSliderUFloat64 creates a slider widget that alters a value based on mouse
// movement only.
func (wnd *Window) DragSliderUFloat64(id string, speed float64, value *float64) error {
	var valueString string
	sliderPressed, _, _ := wnd.sliderHitTest(id)

	// we have a mouse down in the widget, so check to see how much the mouse has
	// moved and slide the control cursor and edit the value accordingly.
	if sliderPressed {
		mouseDeltaX, _ := wnd.Owner.GetMousePositionDelta()
		*value += float64(mouseDeltaX) * speed
		if *value < 0.0 {
			*value = 0.0
		}
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
	_, dimY, _ := font.GetRenderSize("0.0")
	sliderW := wndWidth - wnd.Style.WindowPadding[0] - wnd.Style.WindowPadding[1] - wnd.Style.SliderMargin[0] - wnd.Style.SliderMargin[1]
	sliderH := dimY + wnd.Style.SliderPadding[2] + wnd.Style.SliderPadding[3]

	// calculate how much of the slider control is available to the cursor for
	// movement, which affects the scale of the value to edit.
	sliderW = sliderW - wnd.Style.SliderCursorWidth - wnd.Style.SliderPadding[0] - wnd.Style.SliderPadding[1]
	sliderW = sliderW - wnd.Style.SliderMargin[0] - wnd.Style.SliderMargin[1]

	// clamp the widget to the requested width
	sliderW = wnd.clampWidgetWidthToReqW(sliderW)

	// test to see if the mouse is inside the widget
	lmbStatus := wnd.Owner.GetMouseButtonAction(0)
	if lmbStatus != MouseUp {
		// are  we already the active widget?
		if wnd.Owner.GetActiveInputID() == id {
			return true, sliderW, sliderH
		}

		// try to claim focus -- wont work if something already claimed it this mouse press
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

	// clamp the widget to the requested width
	sliderW = wnd.clampWidgetWidthToReqW(sliderW)
	sliderW = sliderW - wnd.Style.SliderMargin[0] - wnd.Style.SliderMargin[1]

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
	wnd.setNextRowCursorOffset(sliderH + wnd.Style.SliderMargin[2] + wnd.Style.SliderMargin[3])

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

	// clamp the width to the requsted size
	widthDC = wnd.clampWidgetWidthToReqW(widthDC)

	// render the button background
	combos, indexes, fc := cmd.DrawRectFilledDC(pos[0], pos[1], pos[0]+widthDC, pos[1]-heightDC, color, textureIndex, uvPair)
	cmd.AddFaces(combos, indexes, fc)

	// advance the cursor for the width of the text widget
	wnd.addCursorHorizontalDelta(widthDC + wnd.Style.ImageMargin[0] + wnd.Style.ImageMargin[1])
	wnd.setNextRowCursorOffset(heightDC + wnd.Style.ImageMargin[2] + wnd.Style.ImageMargin[3])

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
	wnd.setNextRowCursorOffset(wnd.Style.SeparatorHeight + wnd.Style.SeparatorMargin[2] + wnd.Style.SeparatorMargin[3])
	wnd.StartRow()
}

// Space adds some horizontal space based on the relative width of the window.
// For example: a window width of 800, passing 0.1 adds a space of 80
func (wnd *Window) Space(spaceS float32) {
	_, _, widthDC, _ := wnd.GetDisplaySize()
	wnd.addCursorHorizontalDelta(widthDC * spaceS)
}

// Custom inserts a new cmdList and sets it up for custom rendering.
func (wnd *Window) Custom(widthS, heightS float32, margin mgl.Vec4, customDraw func()) {
	// get the location and size of this widget
	pos := wnd.getCursorDC()
	pos[0] += margin[0]
	pos[1] -= margin[2]
	widthDC, heightDC := wnd.Owner.ScreenToDisplay(widthS, heightS)

	// clamp the width to the requsted size
	widthDC = wnd.clampWidgetWidthToReqW(widthDC)

	// create a new command for this one
	cmd := wnd.addNewCmd()
	cmd.isCustom = true
	cmd.onCustomDraw = customDraw
	cmd.clipRect[0] = pos[0] + 1.0
	cmd.clipRect[1] = pos[1]
	cmd.clipRect[2] = widthDC - 1.0
	cmd.clipRect[3] = heightDC

	// advance the cursor
	wnd.addCursorHorizontalDelta(widthDC + margin[0] + margin[1])
	wnd.setNextRowCursorOffset(heightDC + margin[2] + margin[3])
}

// Editbox creates an editbox control that changes the value string.
func (wnd *Window) Editbox(id string, value *string) (bool, error) {
	cmd := wnd.getLastCmd()

	// get the font for the text
	font := wnd.Owner.GetFont(wnd.Style.FontName)
	if font == nil {
		return false, fmt.Errorf("Couldn't access font %s from the Manager.", wnd.Style.FontName)
	}

	// calculate the location for the widget
	pos := wnd.getCursorDC()
	pos[0] += wnd.Style.EditboxMargin[0]
	pos[1] -= wnd.Style.EditboxMargin[2]

	// calculate the size necessary for the widget; if the text is empty use
	// a const string to calculate the height.
	_, _, wndWidth, _ := wnd.GetDisplaySize()
	textToSize := *value
	if len(textToSize) == 0 {
		textToSize = "FIXEDSIZE"
	}
	_, dimY, _ := font.GetRenderSize(textToSize)
	editboxW := wndWidth - wnd.widgetCursorDC[0] - wnd.Style.WindowPadding[1] - wnd.Style.EditboxMargin[1]
	editboxH := dimY + wnd.Style.EditboxPadding[2] + wnd.Style.EditboxPadding[3]

	// clamp the width to the requsted size
	editboxW = wnd.clampWidgetWidthToReqW(editboxW)
	editboxW = editboxW - wnd.Style.EditboxMargin[0] - wnd.Style.EditboxMargin[1]

	// set a default color for the button
	bgColor := wnd.Style.EditboxBgColor

	// test to see if the mouse is inside the widget
	lmbStatus := wnd.Owner.GetMouseButtonAction(0)
	if lmbStatus != MouseUp {
		// are  we already the active widget?
		if wnd.Owner.GetActiveInputID() != id {
			// try to claim focus -- wont work if something already claimed it this mouse press
			mx, my := wnd.Owner.GetMouseDownPosition(0)
			if mx > pos[0] && my > pos[1]-editboxH && mx < pos[0]+editboxW && my < pos[1] {
				wnd.Owner.SetActiveInputID(id)
				wnd.Owner.setActiveTextEditor(id, 0)
			}
		}
	}

	// see if we're the active editor. if so, then we can consume the key events;
	// otherwise we leave them be.
	editorState := wnd.Owner.getActiveTextEditor()
	if editorState != nil && editorState.ID == id {
		// we're the active ditor so set the background color accordingly
		bgColor = wnd.Style.EditboxActiveColor

		// grab the key events
		keyEvents := wnd.Owner.GetKeyEvents()
		for _, event := range keyEvents {
			if event.IsRune == false {
				// all of these keys reset the timer if it doesn't lose focus, so
				// just reset it here for convenience
				editorState.CursorTimer = 0.0

				// handle the key events specially in their own way
				switch event.KeyCode {
				case EweyKeyRight:
					if editorState.CursorOffset < len(*value) {
						editorState.CursorOffset++
					}
				case EweyKeyLeft:
					if editorState.CursorOffset > 0 {
						editorState.CursorOffset--
					}
				case EweyKeyBackspace:
					// erase the rune previous to the cursor
					if editorState.CursorOffset > 0 {
						newString := (*value)[:editorState.CursorOffset-1] + (*value)[editorState.CursorOffset:]
						*value = newString
						editorState.CursorOffset--
					}
				case EweyKeyDelete:
					// erase the rune just after the cursor
					if editorState.CursorOffset < len(*value) {
						newString := (*value)[:editorState.CursorOffset] + (*value)[editorState.CursorOffset+1:]
						*value = newString
					}
				case EweyKeyEnter, EweyKeyEscape:
					// give up the focus voluntarily here
					wnd.Owner.clearActiveTextEditor()
					wnd.Owner.ClearActiveInputID()
					editorState = nil
				case EweyKeyEnd:
					editorState.CursorOffset = len(*value)
				case EweyKeyHome:
					editorState.CursorOffset = 0
				case EweyKeyInsert:
					if event.ShiftDown {
						clippy, _ := wnd.Owner.GetClipboardString()
						newString := (*value)[:editorState.CursorOffset] + clippy + (*value)[editorState.CursorOffset:]
						*value = newString
					}
				}
			} else {
				// do some special testing for clipboard commands
				if event.Rune == 'V' && event.CtrlDown {
					clippy, _ := wnd.Owner.GetClipboardString()
					newString := (*value)[:editorState.CursorOffset] + clippy + (*value)[editorState.CursorOffset:]
					*value = newString
				} else {
					// insert the rune into the value string
					newString := (*value)[:editorState.CursorOffset] + string(event.Rune) + (*value)[editorState.CursorOffset:]
					*value = newString
					editorState.CursorOffset++
				}
			}
		}

	}

	// render the button background
	combos, indexes, fc := cmd.DrawRectFilledDC(pos[0], pos[1], pos[0]+editboxW, pos[1]-editboxH, bgColor, defaultTextureSampler, wnd.Owner.whitePixelUv)
	cmd.AddFaces(combos, indexes, fc)

	// create the text for the button if the string is not empty
	if len(*value) > 0 {
		textPos := pos
		textPos[0] += wnd.Style.EditboxPadding[0]
		textPos[1] -= wnd.Style.EditboxPadding[2]
		renderData := font.CreateText(textPos, wnd.Style.EditboxTextColor, *value)
		cmd.AddFaces(renderData.ComboBuffer, renderData.IndexBuffer, renderData.Faces)
	}

	// if we're the active editor, deal with drawing the cursor here
	if editorState != nil && editorState.ID == id {
		// add the current delta to the timer
		editorState.CursorTimer += float32(wnd.Owner.FrameDelta)

		// did we overflow the blink interval? if so, reset the timer
		if editorState.CursorTimer > wnd.Style.EditboxBlinkInterval {
			editorState.CursorTimer -= wnd.Style.EditboxBlinkInterval
		}

		// draw the cursor if we're within the blink duration
		if editorState.CursorTimer < wnd.Style.EditboxBlinkDuration {
			cursorOffsetDC := font.OffsetForIndex(*value, editorState.CursorOffset)
			cursorOffsetDC += wnd.Style.EditboxPadding[0]

			// render the editbox cursor
			combos, indexes, fc := cmd.DrawRectFilledDC(pos[0]+cursorOffsetDC, pos[1], pos[0]+cursorOffsetDC+wnd.Style.EditboxCursorWidth, pos[1]-editboxH,
				wnd.Style.EditboxCursorColor, defaultTextureSampler, wnd.Owner.whitePixelUv)
			cmd.AddFaces(combos, indexes, fc)
		}
	}

	// advance the cursor for the width of the text widget
	wnd.addCursorHorizontalDelta(editboxW + wnd.Style.EditboxMargin[0] + wnd.Style.EditboxMargin[1])
	wnd.setNextRowCursorOffset(editboxH + wnd.Style.EditboxMargin[2] + wnd.Style.EditboxMargin[3])

	return true, nil
}

// TreeNode draws the tree node widget on screen with the given text. Returns a
// bool indicating if the tree node is considered to be 'open'.
func (wnd *Window) TreeNode(id string, text string) (bool, error) {
	cmd := wnd.getLastCmd()

	// get the font for the text
	font := wnd.Owner.GetFont(wnd.Style.FontName)
	if font == nil {
		return false, fmt.Errorf("Couldn't access font %s from the Manager.", wnd.Style.FontName)
	}

	// calculate the location for the widget
	pos := wnd.getCursorDC()
	pos[0] += wnd.Style.TreeNodeMargin[0]
	pos[1] -= wnd.Style.TreeNodeMargin[2]

	// calculate the size necessary for the widget
	dimX, dimY, _ := font.GetRenderSize(text)
	nodeW := dimX + wnd.Style.TreeNodePadding[0] + wnd.Style.TreeNodePadding[1]
	nodeH := dimY + wnd.Style.TreeNodePadding[2] + wnd.Style.TreeNodePadding[3]

	// clamp the width of the widget to respect any requests to size
	nodeW = wnd.clampWidgetWidthToReqW(nodeW)
	nodeW = nodeW - wnd.Style.TreeNodeMargin[0] - wnd.Style.TreeNodeMargin[1]

	// check to see if the window has a stored value for this node's ID and
	// whether or not that value indicates if the node is considered open.
	var openState bool
	storedOpenState, statePresent := wnd.getStoredInt(id)
	if statePresent && storedOpenState > 0 {
		openState = true
	}

	// test to see if the mouse is inside the widget
	pressed := false
	buttonTest := wnd.buttonBehavior(id, pos[0], pos[1], nodeW, nodeH)
	if buttonTest == buttonPressed {
		pressed = true
	}

	// if it's pressed, we invert the state and store the updated value
	if pressed {
		openState = !openState
		if openState == false {
			wnd.setStoredInt(id, 0)
		} else {
			wnd.setStoredInt(id, 1)
		}
	}

	// render the node icons in a square
	iconX1 := pos[0]
	iconY1 := pos[1] - nodeH*0.375
	iconX2 := pos[0] + nodeH*0.25
	iconY2 := pos[1] - nodeH*0.625
	combos, indexes, fc := cmd.drawTreeNodeIcon(openState, iconX1, iconY1, iconX2, iconY2, wnd.Style.TreeNodeTextColor, defaultTextureSampler, wnd.Owner.whitePixelUv)
	cmd.AddFaces(combos, indexes, fc)
	iconXOffset := nodeH*0.25 + 4
	pos[0] += iconXOffset // adjust the position to acocund for this

	// create the text for the button
	internalW := nodeW - wnd.Style.TreeNodePadding[0] - wnd.Style.TreeNodePadding[0]
	centerTextX := (internalW / 2.0) - (dimX / 2.0)
	textPos := pos
	textPos[0] += centerTextX
	textPos[1] -= wnd.Style.TreeNodePadding[2]
	renderData := font.CreateText(textPos, wnd.Style.TreeNodeTextColor, text)
	cmd.AddFaces(renderData.ComboBuffer, renderData.IndexBuffer, renderData.Faces)

	// advance the cursor for the width of the text + iconXOffset
	wnd.addCursorHorizontalDelta(nodeW + iconXOffset + wnd.Style.TreeNodeMargin[0] + wnd.Style.TreeNodeMargin[1])
	wnd.setNextRowCursorOffset(nodeH + wnd.Style.TreeNodeMargin[2] + wnd.Style.TreeNodeMargin[3])

	// if we've captured the mouse click event and registered a button press, clear
	// the tracking data for the mouse button so that we don't get duplicate matches.
	if pressed {
		wnd.Owner.ClearMouseButtonAction(0)
	}

	return openState, nil
}

const (
	buttonNoAction = 0
	buttonPressed  = 1
	buttonHover    = 2
)

// buttonBehavior returns a enumerated value of the consts above indicating a
// buttonNoAction, buttonPressed or buttonHover for the mouse interaction with
// the 'button'.
// the function also will set the active input id if mouse is down and was
// originally pressed inside the 'button' space.
func (wnd *Window) buttonBehavior(id string, minX, minY, width, height float32) int {
	result := buttonNoAction

	// test to see if the mouse is inside the widget
	mx, my := wnd.Owner.GetMousePosition()
	if mx > minX && my > minY-height && mx < minX+width && my < minY {
		lmbStatus := wnd.Owner.GetMouseButtonAction(0)

		if lmbStatus == MouseClick {
			result = buttonPressed
		} else if lmbStatus == MouseUp {
			result = buttonHover
		} else {
			// mouse is down, but was it pressed inside the button?
			mdx, mdy := wnd.Owner.GetMouseDownPosition(0)
			if mdx > minX && mdy > minY-height && mdx < minX+width && mdy < minY {
				result = buttonHover
				wnd.Owner.SetActiveInputID(id)
			}
		}
	}

	return result
}
