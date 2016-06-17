// Copyright 2016, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package eweygewey

import (
	mgl "github.com/go-gl/mathgl/mgl32"
)

// constants used for polling the state of a mouse button
const (
	MouseDown        = 0
	MouseUp          = 1
	MouseClick       = 2
	MouseDoubleClick = 4
)

// Style defines parameters to the drawing functions that control the way
// the widgets are organized and drawn.
type Style struct {
	ButtonColor       mgl.Vec4 // button background color
	ButtonHoverColor  mgl.Vec4 // button background color with mouse hovering
	ButtonActiveColor mgl.Vec4 // button background color when clicked
	ButtonTextColor   mgl.Vec4 // button text color
	ButtonMargin      mgl.Vec4 // [left,right,top,bottom] margin values for buttons
	ButtonPadding     mgl.Vec4 // [left,right,top,bottom] padding values for buttons
	FontName          string   // font name to use by default
	SliderBgColor     mgl.Vec4 // slider background color
	SliderCursorColor mgl.Vec4 // slider cursor color
	SliderFloatFormat string   // formatting string for the float value in a slider
	SliderIntFormat   string   // formatting string for the int value in a slider
	SliderMargin      mgl.Vec4 // margin for the slider text strings
	SliderPadding     mgl.Vec4 // padding for the slider text strings
	SliderTextColor   mgl.Vec4 // slider text color
	SliderCursorWidth float32  // slider cursor width
	TextColor         mgl.Vec4 // text color
	TitleBarTextColor mgl.Vec4 // text color
	TitleBarBgColor   mgl.Vec4 // window background color
	WindowBgColor     mgl.Vec4 // window background color
	WindowPadding     mgl.Vec4 // [left,right,top,bottom] padding values for windows
}

var (
	// VertShader330 is the GLSL vertex shader program for the user interface.
	VertShader330 = `#version 330
  uniform mat4 VIEW;
  in vec3 VERTEX_POSITION;
  in vec2 VERTEX_UV;
  in vec4 VERTEX_COLOR;
  out vec3 vs_pos;
  out vec2 vs_uv;
  out vec4 vs_color;
  void main()
  {
    vs_pos = VERTEX_POSITION;
    vs_uv = VERTEX_UV;
    vs_color = VERTEX_COLOR;
    gl_Position = VIEW * vec4(VERTEX_POSITION, 1.0);
  }`

	// FragShader330 is the GLSL fragment shader program for the user interface.
	FragShader330 = `#version 330
  uniform sampler2D TEX_0;
  in vec2 vs_uv;
  in vec4 vs_color;
  out vec4 frag_color;
  void main()
  {
	frag_color = vs_color * texture(TEX_0, vs_uv).rgba;
  }`

	// DefaultStyle is the default style to use for drawing widgets
	DefaultStyle = Style{
		ButtonColor:       ColorIToV(171, 102, 102, 153),
		ButtonActiveColor: ColorIToV(204, 128, 120, 255),
		ButtonHoverColor:  ColorIToV(171, 102, 102, 255),
		ButtonTextColor:   ColorIToV(230, 230, 230, 255),
		ButtonMargin:      mgl.Vec4{2, 2, 4, 4},
		ButtonPadding:     mgl.Vec4{4, 4, 2, 2},
		FontName:          "Default",
		SliderBgColor:     ColorIToV(128, 128, 128, 179),
		SliderCursorColor: ColorIToV(179, 179, 179, 179),
		SliderFloatFormat: "%0.3f",
		SliderIntFormat:   "%d",
		SliderMargin:      mgl.Vec4{2, 2, 4, 4},
		SliderPadding:     mgl.Vec4{2, 2, 2, 2},
		SliderTextColor:   ColorIToV(230, 230, 230, 255),
		SliderCursorWidth: 15,
		TextColor:         ColorIToV(230, 230, 230, 255),
		TitleBarTextColor: ColorIToV(230, 230, 230, 255),
		TitleBarBgColor:   ColorIToV(69, 69, 138, 212),
		WindowBgColor:     ColorIToV(0, 0, 0, 179),
		WindowPadding:     mgl.Vec4{4, 4, 4, 4},
	}
)

// ColorIToV takes the color parameters as integers and returns them
// as a float vector.
func ColorIToV(r, g, b, a int) mgl.Vec4 {
	return mgl.Vec4{float32(r) / 255.0, float32(g) / 255.0, float32(b) / 255.0, float32(a) / 255.0}
}
