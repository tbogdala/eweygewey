// Copyright 2016, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package eweygewey

import (
	"fmt"
	"time"

	mgl "github.com/go-gl/mathgl/mgl32"
	graphics "github.com/tbogdala/fizzle/graphicsprovider"
)

// FrameStartFunc is the type of function to be called when the manager is starting
// a new frame to construct and draw.
type FrameStartFunc func(startTime time.Time)

// textEditState containst state information for the last widget that edits text
// and can be used to store cursor position and other useful info.
type textEditState struct {
	// ID is the ID of the text widget claiming text edit state
	ID string

	// The number of runes in the buffer string to place the cursor after
	CursorOffset int

	// CursorTimer tracks the amount of time since the start of the last blink
	// episode of the cursor. The cursor should be shown during the interval
	// [0 .. Style.EditboxBlinkDuration].
	CursorTimer float32

	// CharacterShift is the amount of characters to shift the dispayed text.
	CharacterShift int
}

// Manager holds all of the widgets and knows how to draw the UI.
type Manager struct {
	// GetMousePosition should be a function that returns the current mouse
	// position for the application.
	GetMousePosition func() (float32, float32)

	// GetMouseDownPosition should be a function that returns the mouse position
	// stored for when the button made the transition from UP->DOWN.
	GetMouseDownPosition func(buttonNumber int) (float32, float32)

	// GetMousePositionDelta should be a function that returns the amount
	// of change in the mouse position.
	GetMousePositionDelta func() (float32, float32)

	// GetMouseButtonAction should be a function that returns the state
	// of a mouse button: MouseUp | MouseDown | MouseRepeat.
	GetMouseButtonAction func(buttonNumber int) int

	// ClearMouseButtonAction should be a function that clears any tracked
	// action data for a mouse button
	ClearMouseButtonAction func(buttonNumber int)

	// GetScrollWheelDelta should be a function that returns the amount of
	// change to the scroll wheel position that has happened since last check.
	GetScrollWheelDelta func(bool) float32

	// GetKeyEvents is the function to be called to get the slice of
	// currently buffered key press events
	GetKeyEvents func() []KeyPressEvent

	// ClearKeyEvents is the function to be called to clear out the key press event buffer
	ClearKeyEvents func()

	// GetClipboardString returns a possible string from the clipboarnd and
	// possibly an error.
	GetClipboardString func() (string, error)

	// SetClipboardString sets a string in the system clipboard.
	SetClipboardString func(string)

	// FrameStart is the time the UI manager's Construct() was called.
	FrameStart time.Time

	// FrameDelta is the time between frames as given to Construct().
	FrameDelta float64

	// ScrollSpeed is how much each move of the scroll wheel should be magnified
	ScrollSpeed float32

	// width is used to construct the ortho projection matrix and is probably
	// best set to the width of the window.
	width int32

	// height is used to construct the ortho projection matrix and is probably
	// best set to the height of the window.
	height int32

	// designHeight is the height the UI was designed at. Practically, this
	// means that text should scale to adjust for resolution changes so that it
	// has the same height relative to the different resolutions.
	// E.g. 800x600 and the font glyphs have a height of 30, then adjusting
	// to 1600x1200 will instruct the package to create text with a height of 60.
	designHeight int32

	// windows is the slice of known windows to render.
	windows []*Window

	// activeInputID is the ID string of the widget that claimed input on mouse down.
	activeInputID string

	// activeTextEdit is the active text editing widget state; if set til nil
	// then there are no text editing widgets with active input focus.
	activeTextEdit *textEditState

	// gfx is the underlying graphics implementation to be used for rendering.
	gfx graphics.GraphicsProvider

	// shader is the shader program used to draw the user interface.
	shader graphics.Program

	// fonts is a map of loaded fonts keyed by a client specified name.
	fonts map[string]*Font

	// whitePixelUv is a vec4 of the UV coordinate to use for the white pixel
	// with (s1,t1,s2,t2) where (s1,t1) is bottom-left and (s2,t2) is top-right.
	whitePixelUv mgl.Vec4

	// frameStartCallbacks is a slice of functions that should be called when
	// the manager is constructing a new frame to draw.
	frameStartCallbacks []FrameStartFunc

	comboBuffer  []float32
	indexBuffer  []uint32
	comboVBO     graphics.Buffer
	indexVBO     graphics.Buffer
	vao          uint32
	faceCount    uint32
	textureStack []graphics.Texture // cleared each frame
}

// NewManager is the constructor for the Manager type that will create
// a new object and sets sane defaults.
func NewManager(gfx graphics.GraphicsProvider) *Manager {
	m := new(Manager)
	m.windows = make([]*Window, 0)
	m.fonts = make(map[string]*Font)
	m.gfx = gfx
	m.whitePixelUv = mgl.Vec4{1.0, 1.0, 1.0, 1.0}
	m.FrameStart = time.Now()
	m.ScrollSpeed = 10.0

	m.vao = gfx.GenVertexArray()

	m.GetMousePosition = func() (float32, float32) { return 0, 0 }
	m.GetMousePositionDelta = func() (float32, float32) { return 0, 0 }
	m.GetMouseButtonAction = func(buttonNumber int) int { return MouseUp }
	m.frameStartCallbacks = []FrameStartFunc{}
	m.textureStack = []graphics.Texture{}

	return m
}

// Initialize does the setup required for the user interface to draw. This
// includes heavier operations like compiling shaders.
func (ui *Manager) Initialize(vertShader, fragShader string, w, h, designH int32) error {
	// compile the shader program from the source provided
	var err error
	ui.shader, err = ui.compileShader(vertShader, fragShader)
	if err != nil {
		return err
	}

	// generate the VBOs
	ui.comboVBO = ui.gfx.GenBuffer()
	ui.indexVBO = ui.gfx.GenBuffer()

	// set the resolution for the user interface
	ui.AdviseResolution(w, h)
	ui.designHeight = designH

	return nil
}

// AddTextureToStack adds a texture ID to the stack of textures the manager maintains
// and returns it's index in the stack +1. In other words, this is a one-based
// number scheme because 0 is reserved for the font.
func (ui *Manager) AddTextureToStack(texID graphics.Texture) uint32 {
	ui.textureStack = append(ui.textureStack, texID)
	return uint32(len(ui.textureStack))
}

// AdviseResolution will change the resolution the Manager uses to draw widgets.
func (ui *Manager) AdviseResolution(w int32, h int32) {
	ui.width = w
	ui.height = h
}

// GetDesignHeight returns the normalized height for the UI.
func (ui *Manager) GetDesignHeight() int32 {
	return ui.designHeight
}

// GetResolution returns the width and height of the user interface.
func (ui *Manager) GetResolution() (int32, int32) {
	return ui.width, ui.height
}

// NewWindow creates a new window and adds it to the collection of windows to draw.
func (ui *Manager) NewWindow(id string, x, y, w, h float32, constructor BuildCallback) *Window {
	wnd := newWindow(id, x, y, w, h, constructor)
	wnd.Owner = ui
	ui.windows = append(ui.windows, wnd)
	return wnd
}

// GetWindow returns a window based on the id string passed in
func (ui *Manager) GetWindow(id string) *Window {
	for _, wnd := range ui.windows {
		if wnd.ID == id {
			return wnd
		}
	}

	return nil
}

// GetWindowsByFilter returns a slice of *Window which is populated by
// filtering the internal window list with the function provided.
// If the function returns true the window will get included in the results.
func (ui *Manager) GetWindowsByFilter(filter func(w *Window) bool) []*Window {
	results := []*Window{}
	for _, wnd := range ui.windows {
		if filter(wnd) {
			results = append(results, wnd)
		}
	}

	return results
}

// RemoveWindow will remove the window from the user interface.
func (ui *Manager) RemoveWindow(wndToRemove *Window) {
	filtered := ui.windows[:0]
	for _, wnd := range ui.windows {
		if wnd.ID != wndToRemove.ID {
			filtered = append(filtered, wnd)
		}
	}
	ui.windows = filtered
}

// NewFont loads the font from a file and 'registers' it with the UI manager.
func (ui *Manager) NewFont(name string, fontFilepath string, scaleInt int, glyphs string) (*Font, error) {
	f, err := newFont(ui, fontFilepath, scaleInt, glyphs)

	// if we succeeded, store the font with the name specified
	if err == nil {
		ui.fonts[name] = f
	}

	return f, err
}

// NewFontBytes loads the font from a byte slice and 'registers' it with the UI manager.
func (ui *Manager) NewFontBytes(name string, fontBytes []byte, scaleInt int, glyphs string) (*Font, error) {
	f, err := newFontBytes(ui, fontBytes, scaleInt, glyphs)

	// if we succeeded, store the font with the name specified
	if err == nil {
		ui.fonts[name] = f
	}

	return f, err
}

// GetFont attempts to get the font by name from the Manager's collection.
// It returns the font on success or nil on failure.
func (ui *Manager) GetFont(name string) *Font {
	return ui.fonts[name]
}

// AddConstructionStartCallback adds a new callback to the slice of callbacks that
// will be called when the manager is starting construction of a new frame to draw.
func (ui *Manager) AddConstructionStartCallback(cb FrameStartFunc) {
	ui.frameStartCallbacks = append(ui.frameStartCallbacks, cb)
}

// SetActiveInputID sets the active input id which tells the user interface
// which widget is currently claiming 'focus' for input. Returns a bool indicating
// if the focus claim was successful because the input can be claimed only once
// per UP->DOWN mouse transition.
func (ui *Manager) SetActiveInputID(id string) bool {
	if ui.activeInputID == "" || ui.GetMouseButtonAction(0) != MouseDown {
		ui.activeInputID = id

		// clear out the editor state if we select a different widget
		if ui.activeTextEdit != nil && ui.activeInputID != ui.activeTextEdit.ID {
			ui.activeTextEdit = nil
		}

		return true
	}

	return false
}

// GetActiveInputID returns the active input id which claimed input focus.
func (ui *Manager) GetActiveInputID() string {
	return ui.activeInputID
}

// ClearActiveInputID clears any focus claims.
func (ui *Manager) ClearActiveInputID() {
	ui.activeInputID = ""
}

// setActiveTextEditor sets the active widget id which gets text editing input.
// Returns a bool indicating if the claim for active text editor successed.
func (ui *Manager) setActiveTextEditor(id string, cursorPos int) bool {
	// already claimed, so sorry
	if ui.activeTextEdit != nil {
		return false
	}

	// claim the fresh focus
	var ate textEditState
	ate.ID = id
	ate.CursorOffset = cursorPos
	ui.activeTextEdit = &ate

	// clear out the old key events
	ui.ClearKeyEvents()

	return true
}

// getActiveTextEditor returns the active text editor state if one is set.
func (ui *Manager) getActiveTextEditor() *textEditState {
	return ui.activeTextEdit
}

// clearActiveTextEditor will remove the active text editor from tracking.
func (ui *Manager) clearActiveTextEditor() {
	ui.activeTextEdit = nil
}

// Construct loops through all of the Windows in the Manager and creates
// all of the widgets and their data. This function does not buffer the
// result to VBO or do the actual rendering -- call Draw() for that.
func (ui *Manager) Construct(frameDelta float64) {
	// reset the display data
	ui.comboBuffer = ui.comboBuffer[:0]
	ui.indexBuffer = ui.indexBuffer[:0]
	ui.faceCount = 0
	ui.FrameStart = time.Now()
	ui.textureStack = ui.textureStack[:0]
	ui.FrameDelta = frameDelta

	// call all of the frame start callbacks
	for _, frameStartCB := range ui.frameStartCallbacks {
		frameStartCB(ui.FrameStart)
	}

	// trigger a mouse position check each frame
	ui.GetMousePosition()
	ui.GetScrollWheelDelta(false)

	// see if we need to clear the active widget id
	if ui.GetMouseButtonAction(0) != MouseDown {
		ui.ClearActiveInputID()
	}

	// loop through all of the windows and tell them to self-construct.
	for _, w := range ui.windows {
		w.construct()
	}
}

// bindOpenGLData sets the program, VAO, uniforms and attributes required for the
// controls to be drawn from the command buffers
func (ui *Manager) bindOpenGLData(style *Style, view mgl.Mat4) {
	const floatSize = 4
	const uintSize = 4
	const posOffset = 0
	const uvOffset = floatSize * 2
	const texIdxOffset = floatSize * 4
	const colorOffset = floatSize * 5
	const VBOStride = floatSize * (2 + 2 + 1 + 4) // vert / uv / texIndex / color

	gfx := ui.gfx

	gfx.UseProgram(ui.shader)
	gfx.BindVertexArray(ui.vao)

	// bind the uniforms and attributes
	shaderViewMatrix := gfx.GetUniformLocation(ui.shader, "VIEW")
	gfx.UniformMatrix4fv(shaderViewMatrix, 1, false, view)

	font := ui.GetFont(style.FontName)
	shaderTex0 := gfx.GetUniformLocation(ui.shader, "TEX[0]")
	if shaderTex0 >= 0 {
		if font != nil {
			gfx.ActiveTexture(graphics.TEXTURE0)
			gfx.BindTexture(graphics.TEXTURE_2D, font.Texture)
			gfx.Uniform1i(shaderTex0, 0)
		}
	}
	if len(ui.textureStack) > 0 {
		for stackIdx, texID := range ui.textureStack {
			uniStr := fmt.Sprintf("TEX[%d]", stackIdx+1)
			texUniLoc := gfx.GetUniformLocation(ui.shader, uniStr)
			if texUniLoc >= 0 {
				gfx.ActiveTexture(graphics.TEXTURE0 + graphics.Texture(stackIdx+1))
				gfx.BindTexture(graphics.TEXTURE_2D, texID)
				gfx.Uniform1i(texUniLoc, int32(stackIdx+1))
			}
		}
	}

	shaderPosition := gfx.GetAttribLocation(ui.shader, "VERTEX_POSITION")
	gfx.BindBuffer(graphics.ARRAY_BUFFER, ui.comboVBO)
	gfx.EnableVertexAttribArray(uint32(shaderPosition))
	gfx.VertexAttribPointer(uint32(shaderPosition), 2, graphics.FLOAT, false, VBOStride, gfx.PtrOffset(posOffset))

	uvPosition := gfx.GetAttribLocation(ui.shader, "VERTEX_UV")
	gfx.EnableVertexAttribArray(uint32(uvPosition))
	gfx.VertexAttribPointer(uint32(uvPosition), 2, graphics.FLOAT, false, VBOStride, gfx.PtrOffset(uvOffset))

	colorPosition := gfx.GetAttribLocation(ui.shader, "VERTEX_COLOR")
	gfx.EnableVertexAttribArray(uint32(colorPosition))
	gfx.VertexAttribPointer(uint32(colorPosition), 4, graphics.FLOAT, false, VBOStride, gfx.PtrOffset(colorOffset))

	texIdxPosition := gfx.GetAttribLocation(ui.shader, "VERTEX_TEXTURE_INDEX")
	gfx.EnableVertexAttribArray(uint32(texIdxPosition))
	gfx.VertexAttribPointer(uint32(texIdxPosition), 1, graphics.FLOAT, false, VBOStride, gfx.PtrOffset(texIdxOffset))

	gfx.BindBuffer(graphics.ELEMENT_ARRAY_BUFFER, ui.indexVBO)
}

// Draw buffers the UI vertex data into the rendering pipeline and does
// the actual draw call.
func (ui *Manager) Draw() {
	const floatSize = 4
	const uintSize = 4
	const posOffset = 0
	const uvOffset = floatSize * 2
	const texIdxOffset = floatSize * 4
	const colorOffset = floatSize * 5
	const VBOStride = floatSize * (2 + 2 + 1 + 4) // vert / uv / texIndex / color
	gfx := ui.gfx

	// FIXME: move the zdepth definitions elsewhere
	const minZDepth = -100.0
	const maxZDepth = 100.0

	gfx.Disable(graphics.DEPTH_TEST)
	gfx.Enable(graphics.SCISSOR_TEST)

	// for now, loop through all of the windows and copy all of the data into the manager's buffer
	// FIXME: this could be buffered straight from the cmdList
	var startIndex uint32
	for _, w := range ui.windows {
		for _, cmd := range w.cmds {
			if cmd.isCustom {
				continue
			}

			ui.comboBuffer = append(ui.comboBuffer, cmd.comboBuffer...)

			// reindex the index buffer to reference the correct vertex data
			highestIndex := uint32(0)
			for _, i := range cmd.indexBuffer {
				if i > highestIndex {
					highestIndex = i
				}
				ui.indexBuffer = append(ui.indexBuffer, i+startIndex)
			}
			ui.faceCount += cmd.faceCount
			startIndex += highestIndex + 1
		}
	}

	// make sure that we're going to draw something
	if startIndex == 0 {
		return
	}

	gfx.BindVertexArray(ui.vao)
	view := mgl.Ortho(0.5, float32(ui.width)+0.5, 0.5, float32(ui.height)+0.5, minZDepth, maxZDepth)

	// buffer the data
	gfx.BindBuffer(graphics.ARRAY_BUFFER, ui.comboVBO)
	gfx.BufferData(graphics.ARRAY_BUFFER, floatSize*len(ui.comboBuffer), gfx.Ptr(&ui.comboBuffer[0]), graphics.STREAM_DRAW)
	gfx.BindBuffer(graphics.ELEMENT_ARRAY_BUFFER, ui.indexVBO)
	gfx.BufferData(graphics.ELEMENT_ARRAY_BUFFER, uintSize*len(ui.indexBuffer), gfx.Ptr(&ui.indexBuffer[0]), graphics.STREAM_DRAW)

	// this should be set to true when the uniforms and attributes, etc... need to be rebound
	needRebinding := true

	// loop through the windows and each window's draw cmd list
	indexOffset := uint32(0)
	for _, w := range ui.windows {
		for _, cmd := range w.cmds {
			gfx.Scissor(int32(cmd.clipRect[0]), int32(cmd.clipRect[1]-cmd.clipRect[3]), int32(cmd.clipRect[2]), int32(cmd.clipRect[3]))

			// for most widgets, isCustom will be false, so we just draw things how we have them bound and then
			// update the index offset into the master combo and index buffers stored in Manager.
			if cmd.isCustom == false {
				if needRebinding {
					// bind all of the uniforms and attributes
					ui.bindOpenGLData(&DefaultStyle, view)
					gfx.Viewport(0, 0, ui.width, ui.height)
					needRebinding = false
				}
				gfx.DrawElements(graphics.TRIANGLES, int32(cmd.faceCount*3), graphics.UNSIGNED_INT, gfx.PtrOffset(int(indexOffset)*uintSize))
				indexOffset += cmd.faceCount * 3
			} else {
				gfx.Viewport(int32(cmd.clipRect[0]), int32(cmd.clipRect[1]-cmd.clipRect[3]), int32(cmd.clipRect[2]), int32(cmd.clipRect[3]))
				cmd.onCustomDraw()
				needRebinding = true
			}
		}
	}

	gfx.BindVertexArray(0)
	gfx.Disable(graphics.SCISSOR_TEST)
	gfx.Enable(graphics.DEPTH_TEST)
}

func (ui *Manager) compileShader(vertShader, fragShader string) (graphics.Program, error) {
	gfx := ui.gfx

	// create the program
	prog := gfx.CreateProgram()

	// create the vertex shader
	var status int32
	vs := gfx.CreateShader(graphics.VERTEX_SHADER)
	gfx.ShaderSource(vs, vertShader)
	gfx.CompileShader(vs)
	gfx.GetShaderiv(vs, graphics.COMPILE_STATUS, &status)
	if status == graphics.FALSE {
		log := gfx.GetShaderInfoLog(vs)
		return 0, fmt.Errorf("Failed to compile the vertex shader:\n%s", log)
	}
	defer gfx.DeleteShader(vs)

	// create the fragment shader
	fs := gfx.CreateShader(graphics.FRAGMENT_SHADER)
	gfx.ShaderSource(fs, fragShader)
	gfx.CompileShader(fs)
	gfx.GetShaderiv(fs, graphics.COMPILE_STATUS, &status)
	if status == graphics.FALSE {
		log := gfx.GetShaderInfoLog(fs)
		return 0, fmt.Errorf("Failed to compile the fragment shader:\n%s", log)
	}
	defer gfx.DeleteShader(fs)

	// attach the shaders to the program and link
	gfx.AttachShader(prog, vs)
	gfx.AttachShader(prog, fs)
	gfx.LinkProgram(prog)
	gfx.GetProgramiv(prog, graphics.LINK_STATUS, &status)
	if status == graphics.FALSE {
		log := gfx.GetProgramInfoLog(prog)
		return 0, fmt.Errorf("Failed to link the program!\n%s", log)
	}

	return prog, nil
}

// ScreenToDisplay converts screen-normalized point to resolution-specific
// coordinates with the origin in the lower left corner.
// E.g. if the UI is 800x600, calling with (0.5, 0.5) returns (400, 300)
func (ui *Manager) ScreenToDisplay(xS, yS float32) (float32, float32) {
	return xS * float32(ui.width), yS * float32(ui.height)
}

// DisplayToScreen converts a resolution-specific coordinate to screen-normalized
// space with the origin in the lower left corner.
// E.g. if the UI is 800x600, coalling with (400,300) returns (0.5, 0.5)
func (ui *Manager) DisplayToScreen(xD, yD float32) (float32, float32) {
	return xD / float32(ui.width), yD / float32(ui.height)
}

// DrawRectFilled draws a rectangle in the user interface using a solid background.
// Coordinate parameters should be passed in screen-normalized space. This gets
// appended to the command list passed in.
func (ui *Manager) DrawRectFilled(cmd *cmdList, xS, yS, wS, hS float32, color mgl.Vec4, textureIndex uint32) {
	x, y := ui.ScreenToDisplay(xS, yS)
	w, h := ui.ScreenToDisplay(wS, hS)
	combos, indexes, fc := cmd.DrawRectFilledDC(x, y, x+w, y-h, color, textureIndex, ui.whitePixelUv)
	cmd.AddFaces(combos, indexes, fc)
}
