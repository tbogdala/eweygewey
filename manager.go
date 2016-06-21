// Copyright 2016, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package eweygewey

import (
	"fmt"
	"time"

	mgl "github.com/go-gl/mathgl/mgl32"
	graphics "github.com/tbogdala/fizzle/graphicsprovider"
)

// GetMousePositionFunc is the type of function to be called to get the mouse position.
type GetMousePositionFunc func() (float32, float32)

// GetMouseDownPositionFunc is the type of function to be called that takes a button number
// to query and should return the location of the mouse when that button transitioned
// from UP->DOWN.
type GetMouseDownPositionFunc func(buttonNumber int) (float32, float32)

// GetMouseButtonActionFunc is the type of function to be called that takes a button number
// to query and should return an enumeration value like MouseUp, MouseDown, etc...
type GetMouseButtonActionFunc func(buttonNumber int) int

// GetMousePositionDeltaFunc is the type of function to be called to get the
// last amount of change in the mouse position.
type GetMousePositionDeltaFunc func() (float32, float32)

// GetScrollWheelDeltaFunc is the type of function to be called to get the
// amount of change that has happened to the mouse scroll wheel since last check.
// The bool parameter indicate whether or not to return a cached value.
type GetScrollWheelDeltaFunc func(bool) float32

// FrameStartFunc is the type of function to be called when the manager is starting
// a new frame to construct and draw.
type FrameStartFunc func(startTime time.Time)

// Manager holds all of the widgets and knows how to draw the UI.
type Manager struct {
	// GetMousePosition should be a function that returns the current mouse
	// position for the application.
	GetMousePosition GetMousePositionFunc

	// GetMouseDownPosition should be a function that returns the mouse position
	// stored for when the button made the transition from UP->DOWN.
	GetMouseDownPosition GetMouseDownPositionFunc

	// GetMousePositionDelta should be a function that returns the amount
	// of change in the mouse position.
	GetMousePositionDelta GetMousePositionDeltaFunc

	// GetMouseButtonAction should be a function that returns the state
	// of a mouse button: MouseUp | MouseDown | MouseRepeat.
	GetMouseButtonAction GetMouseButtonActionFunc

	// GetScrollWheelDelta should be a function that returns the amount of
	// change to the scroll wheel position that has happened since last check.
	GetScrollWheelDelta GetScrollWheelDeltaFunc

	// FrameStart is the time the UI manager's Construct() was called.
	FrameStart time.Time

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

// NewFont loads the font from a file and 'registers' it with the UI manager.
func (ui *Manager) NewFont(name string, fontFilepath string, scaleInt int, glyphs string) (*Font, error) {
	f, err := newFont(ui, fontFilepath, scaleInt, glyphs)

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
// if the focus claim was successful
func (ui *Manager) SetActiveInputID(id string) bool {
	if ui.activeInputID == "" || ui.GetMouseButtonAction(0) != MouseDown {
		ui.activeInputID = id
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

// Construct loops through all of the Windows in the Manager and creates
// all of the widgets and their data. This function does not buffer the
// result to VBO or do the actual rendering -- call Draw() for that.
func (ui *Manager) Construct() {
	// reset the display data
	ui.comboBuffer = ui.comboBuffer[:0]
	ui.indexBuffer = ui.indexBuffer[:0]
	ui.faceCount = 0
	ui.FrameStart = time.Now()
	ui.textureStack = ui.textureStack[:0]

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

	style := DefaultStyle

	// for now, loop through all of the windows and copy all of the data into the manager's buffer
	// FIXME: this could be buffered straight from the cmdList
	startIndex := uint32(0)
	for _, w := range ui.windows {
		for _, cmd := range w.cmds {
			ui.comboBuffer = append(ui.comboBuffer, cmd.comboBuffer...)

			// reindex the index buffer to reference the correct vertex data
			for _, i := range cmd.indexBuffer {
				ui.indexBuffer = append(ui.indexBuffer, i+startIndex)
			}
			ui.faceCount += cmd.faceCount
			startIndex += cmd.faceCount * 2
		}
	}

	gfx.UseProgram(ui.shader)
	gfx.BindVertexArray(ui.vao)
	view := mgl.Ortho(0, float32(ui.width), 0, float32(ui.height), minZDepth, maxZDepth)

	// buffer the data
	gfx.BindBuffer(graphics.ARRAY_BUFFER, ui.comboVBO)
	gfx.BufferData(graphics.ARRAY_BUFFER, floatSize*len(ui.comboBuffer), gfx.Ptr(&ui.comboBuffer[0]), graphics.STREAM_DRAW)
	gfx.BindBuffer(graphics.ELEMENT_ARRAY_BUFFER, ui.indexVBO)
	gfx.BufferData(graphics.ELEMENT_ARRAY_BUFFER, uintSize*len(ui.indexBuffer), gfx.Ptr(&ui.indexBuffer[0]), graphics.STREAM_DRAW)

	// bind the attributes
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
			gfx.ActiveTexture(graphics.TEXTURE0 + graphics.Texture(stackIdx+1))
			gfx.BindTexture(graphics.TEXTURE_2D, texID)
			gfx.Uniform1i(texUniLoc, int32(stackIdx+1))
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

	// loop through the windows and each window's draw cmd list
	indexOffset := uint32(0)
	for _, w := range ui.windows {
		for _, cmd := range w.cmds {
			gfx.Scissor(int32(cmd.clipRect[0]), int32(cmd.clipRect[1]-cmd.clipRect[3]), int32(cmd.clipRect[2]), int32(cmd.clipRect[3]))
			gfx.DrawElements(graphics.TRIANGLES, int32(cmd.faceCount*3), graphics.UNSIGNED_INT, gfx.PtrOffset(int(indexOffset)*uintSize))
			indexOffset += cmd.faceCount * 3
		}
	}

	gfx.BindVertexArray(0)
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
