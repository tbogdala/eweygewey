// Copyright 2016, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
	"fmt"
	"runtime"
	"time"

	glfw "github.com/go-gl/glfw/v3.1/glfw"

	gui "github.com/tbogdala/eweygewey"
	glfwinput "github.com/tbogdala/eweygewey/glfwinput"
	fizzle "github.com/tbogdala/fizzle"
	graphics "github.com/tbogdala/fizzle/graphicsprovider"
	gl "github.com/tbogdala/fizzle/graphicsprovider/opengl"
)

const (
	fontScale    = 18
	fontFilepath = "../assets/Oswald-Heavy.ttf"
	fontGlyphs   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890., :[]{}\\|<>;\"'~`?/-+_=()*&^%$#@!"
)

var (
	glfwWindow *glfw.Window
	gfx        graphics.GraphicsProvider
	uiman      *gui.Manager

	thisFrame        time.Time
	lastFrame        time.Time
	frameCounterTime time.Time
	frameCounter     int
	lastCalcFPS      int
	frameDelta       float64
)

// GLFW event handling must run on the main OS thread
func init() {
	runtime.LockOSThread()
}

func keyCallback(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if key == glfw.KeyEscape && action == glfw.Press {
		w.SetShouldClose(true)
	}
}

func renderFrame(frameDelta float64) {
	// calculate the frame timing and FPS
	if thisFrame.Sub(frameCounterTime).Seconds() > 1.0 {
		lastCalcFPS = frameCounter
		frameCounterTime = thisFrame
		frameCounter = 0
	}
	frameCounter++
	lastFrame = thisFrame

	// clear the screen
	width, height := uiman.GetResolution()
	clearColor := gui.ColorIToV(114, 144, 154, 255)
	gfx.Viewport(0, 0, width, height)
	gfx.ClearColor(clearColor[0], clearColor[1], clearColor[2], clearColor[3])
	gfx.Clear(graphics.COLOR_BUFFER_BIT | graphics.DEPTH_BUFFER_BIT)

	// draw the user interface
	gfx.Disable(graphics.DEPTH_TEST)
	uiman.Construct()
	uiman.Draw()
	gfx.Enable(graphics.DEPTH_TEST)
}

func main() {
	const w = 1280
	const h = 720
	glfwWindow, gfx = initGraphics("gui basic", w, h)
	glfwWindow.SetKeyCallback(keyCallback)
	lastFrame = time.Now()
	frameCounterTime = lastFrame
	lastCalcFPS = -1

	// setup the OpenGL graphics provider
	var err error
	gfx, err = gl.InitOpenGL()
	if err != nil {
		panic("Failed to initialize OpenGL! " + err.Error())
	}

	// create and initialize the gui Manager
	uiman = gui.NewManager(gfx)
	err = uiman.Initialize(gui.VertShader330, gui.FragShader330, w, h, h)
	if err != nil {
		panic("Failed to initialize the user interface! " + err.Error())
	}
	glfwinput.SetInputHandlers(uiman, glfwWindow)

	// load a font
	_, err = uiman.NewFont("Default", fontFilepath, fontScale, fontGlyphs)
	if err != nil {
		panic("Failed to load the font file! " + err.Error())
	}

	// delcare the windows so that we can use them in the closures below
	var testInt int
	var mouseTestWindow, mainWindow *gui.Window

	// create a small overlay window in the corner
	mouseTestWindow = uiman.NewWindow(0.05, 0.95, 0.2, 0.35, func(wnd *gui.Window) {
		// display the mouse coordinate
		mouseX, mouseY := uiman.GetMousePosition()
		wnd.Text(fmt.Sprintf("Mouse position = %.2f,%.2f", mouseX, mouseY))

		// display the LMB button status
		wnd.StartRow()
		lmbAction := uiman.GetMouseButtonAction(0)
		if lmbAction == gui.MouseUp {
			wnd.Text("LMB = UP")
		} else if lmbAction == gui.MouseDown {
			wnd.Text("LMB = DOWN")
		}

		// display the RMB button status
		wnd.StartRow()
		rmbAction := uiman.GetMouseButtonAction(1)
		if rmbAction == gui.MouseUp {
			wnd.Text("RMB = UP")
		} else if rmbAction == gui.MouseDown {
			wnd.Text("RMB = DOWN")
		}

		// throw a few test buttons into the mix
		wnd.StartRow()
		wnd.Button("Show Cursor Pos")
		wnd.Button("Test 1")

		wnd.StartRow()
		wnd.SliderFloat(&mainWindow.Width, 0.0, 1.0)
		wnd.StartRow()
		wnd.SliderInt(&testInt, 0, 255)
	})
	mouseTestWindow.ShowTitleBar = false
	mouseTestWindow.IsMoveable = false

	// create the test window for widgets
	mainWindow = uiman.NewWindow(0.3, 0.6, 0.5, 0.5, func(wnd *gui.Window) {
		wnd.Text(fmt.Sprintf("Current FPS = %d ; frame delta = %0.06g ms", lastCalcFPS, frameDelta/1000.0))
	})
	mainWindow.Title = "Widget Test"

	// set some additional OpenGL flags
	gfx.BlendEquation(graphics.FUNC_ADD)
	gfx.BlendFunc(graphics.SRC_ALPHA, graphics.ONE_MINUS_SRC_ALPHA)
	gfx.Enable(graphics.BLEND)

	// enter the renderloop
	thisFrame = time.Now()
	for !glfwWindow.ShouldClose() {
		// draw the sample
		thisFrame = time.Now()
		frameDelta = thisFrame.Sub(lastFrame).Seconds()
		renderFrame(frameDelta)

		// draw the screen and get any input
		glfwWindow.SwapBuffers()
		glfw.PollEvents()

		// update the last render time
		lastFrame = thisFrame
	}
}

// onWindowResize is called when the window changes size
func onWindowResize(w *glfw.Window, width int, height int) {
	uiman.AdviseResolution(int32(width), int32(height))
}

// initGraphics creates an OpenGL window and initializes the required graphics libraries.
// It will either succeed or panic.
func initGraphics(title string, w int, h int) (*glfw.Window, graphics.GraphicsProvider) {
	// GLFW must be initialized before it's called
	err := glfw.Init()
	if err != nil {
		panic("Can't init glfw! " + err.Error())
	}

	// request a OpenGL 3.3 core context
	glfw.WindowHint(glfw.Samples, 0)
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)

	// do the actual window creation
	mainWindow, err := glfw.CreateWindow(w, h, title, nil, nil)
	if err != nil {
		panic("Failed to create the main window! " + err.Error())
	}
	mainWindow.SetSizeCallback(onWindowResize)
	mainWindow.MakeContextCurrent()

	// disable v-sync for max draw rate
	glfw.SwapInterval(0)

	// initialize OpenGL
	gfx, err := gl.InitOpenGL()
	if err != nil {
		panic("Failed to initialize OpenGL! " + err.Error())
	}
	fizzle.SetGraphics(gfx)

	return mainWindow, gfx
}
