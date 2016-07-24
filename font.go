// Copyright 2015, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package eweygewey

/*
Based primarily on gltext found at https://github.com/go-gl/gltext
But also based on examples from the freetype-go project:

	https://github.com/golang/freetype

This implementation differs in the way the images are rendered and then
copied into an OpenGL texture. In addition to that, this module can
create a renderable 'string' node which is a bunch of polygons with uv's
mapped to the appropriate glyphs.
*/

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"
	"os"

	mgl "github.com/go-gl/mathgl/mgl32"
	ft "github.com/golang/freetype"
	graphics "github.com/tbogdala/fizzle/graphicsprovider"
	"golang.org/x/image/math/fixed"
)

// runeData stores information pulled from the freetype parsing of glyphs.
type runeData struct {
	imgX, imgY                    int // offset into the image texture for the top left position of rune
	advanceWidth, leftSideBearing int // HMetric data from glyph
	advanceHeight, topSideBearing int // VMetric data from glyph
	uvMinX, uvMinY                float32
	uvMaxX, uvMaxY                float32
}

// Font contains data regarding a font and the texture that was created
// with the specified set of glyphs. It can then be used to create
// renderable string objects.
type Font struct {
	Texture     graphics.Texture
	TextureSize int
	Glyphs      string
	GlyphHeight int
	GlyphWidth  int
	Owner       *Manager
	locations   map[rune]runeData
}

// newFont takes a fontFilepath and uses the Go freetype library to parse it
// and render the specified glyphs to a texture that is then buffered into OpenGL.
func newFont(owner *Manager, fontFilepath string, scaleInt int, glyphs string) (f *Font, e error) {
	f = new(Font)
	scale := fixed.I(scaleInt)

	// allocate the location map
	f.locations = make(map[rune]runeData)

	// Load the font used for UI interaction
	fontFile, err := os.Open(fontFilepath)
	if err != nil {
		return f, fmt.Errorf("Failed to open the font file.\n%v", err)
	}
	defer fontFile.Close()

	// load in the font
	fontBytes, err := ioutil.ReadAll(fontFile)
	if err != nil {
		return f, fmt.Errorf("Failed to load font data from stream.\n%v", err)
	}

	// parse the truetype font data
	ttfData, err := ft.ParseFont(fontBytes)
	if err != nil {
		return f, fmt.Errorf("Failed to prase the truetype font data.\n%v", err)
	}

	// this may have negative components, but get the bounds for the font
	glyphBounds := ttfData.Bounds(scale)

	// width and height are getting +2 here since the glyph will be buffered by a
	// pixel in the texture
	glyphDimensions := glyphBounds.Max.Sub(glyphBounds.Min)
	glyphWidth := int(glyphDimensions.X>>6) + 4
	glyphHeight := int(glyphDimensions.Y>>6) + 4

	// create the buffer image used to draw the glyphs
	glyphRect := image.Rect(0, 0, glyphWidth, glyphHeight)
	glyphImg := image.NewRGBA(glyphRect)

	// calculate the area needed for the font texture
	var fontTexSize = 2
	minAreaNeeded := (glyphWidth) * (glyphHeight) * len(glyphs)
	for (fontTexSize * fontTexSize) < minAreaNeeded {
		fontTexSize *= 2
		if fontTexSize > 2048 {
			return f, fmt.Errorf("Font texture was going to exceed 2048x2048 and that's currently not supported.")
		}
	}

	// create the font image
	fontImgRect := image.Rect(0, 0, fontTexSize, fontTexSize)
	fontImg := image.NewRGBA(fontImgRect)

	// the number of glyphs
	fontRowSize := fontTexSize / (glyphWidth)

	// create the freetype context
	c := ft.NewContext()
	c.SetDPI(72)
	c.SetFont(ttfData)
	c.SetFontSize(float64(scaleInt))
	c.SetClip(glyphImg.Bounds())
	c.SetDst(glyphImg)
	c.SetSrc(image.White)

	// NOTE: always disabled for now since it causes a stack overflow error
	//c.SetHinting(font.HintingFull)

	var fx, fy int
	for _, ch := range glyphs {
		index := ttfData.Index(ch)
		metricH := ttfData.HMetric(scale, index)
		metricV := ttfData.VMetric(scale, index)

		fxGW := fx * glyphWidth
		fyGH := fy * glyphHeight

		f.locations[ch] = runeData{
			fxGW, fyGH,
			int(metricH.AdvanceWidth) >> 6, int(metricH.LeftSideBearing) >> 6,
			int(metricV.AdvanceHeight) >> 6, int(metricV.TopSideBearing) >> 6,
			float32(fxGW) / float32(fontTexSize), float32(fyGH+glyphHeight) / float32(fontTexSize),
			float32(fxGW+glyphWidth) / float32(fontTexSize), float32(fyGH) / float32(fontTexSize),
		}

		pt := ft.Pt(1, 1+int(c.PointToFixed(float64(scaleInt))>>6))
		_, err := c.DrawString(string(ch), pt)
		if err != nil {
			return f, fmt.Errorf("Freetype returned an error while drawing a glyph: %v.", err)
		}

		// copy the glyph image into the font image
		for subY := 0; subY < glyphHeight; subY++ {
			for subX := 0; subX < glyphWidth; subX++ {
				glyphRGBA := glyphImg.RGBAAt(subX, subY)
				fontImg.SetRGBA((fxGW)+subX, (fyGH)+subY, glyphRGBA)
			}
		}

		// erase the glyph image buffer
		draw.Draw(glyphImg, glyphImg.Bounds(), image.Transparent, image.ZP, draw.Src)

		// adjust the pointers into the font image
		fx++
		if fx > fontRowSize {
			fx = 0
			fy++
		}

	}

	// set the white point
	fontImg.SetRGBA(fontTexSize-1, fontTexSize-1, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	// buffer the font image into an OpenGL texture
	f.Glyphs = glyphs
	f.TextureSize = fontTexSize
	f.GlyphWidth = glyphWidth
	f.GlyphHeight = glyphHeight
	f.Owner = owner
	f.Texture = f.loadRGBAToTexture(fontImg.Pix, int32(fontImg.Rect.Max.X))

	return
}

// Destroy releases the OpenGL texture for the font.
func (f *Font) Destroy() {
	f.Owner.gfx.DeleteTexture(f.Texture)
}

// GetCurrentScale returns the scale value for the font based on the current
// Manager's resolution vs the resolution the UI was designed for.
func (f *Font) GetCurrentScale() float32 {
	_, uiHeight := f.Owner.GetResolution()
	designHeight := f.Owner.GetDesignHeight()
	return float32(uiHeight) / float32(designHeight)
}

// GetRenderSize returns the width and height necessary in pixels for the
// font to display a string. The third return value is the advance height the string.
func (f *Font) GetRenderSize(msg string) (float32, float32, float32) {
	var w, h float32

	// see how much to scale the size based on current resolution vs desgin resolution
	fontScale := f.GetCurrentScale()

	for i, ch := range msg {
		// get the rune data
		chData := f.locations[ch]

		// the last character must use the full glyph width and not just
		// the advance.
		if i != len(msg)-1 {
			w += float32(chData.advanceWidth) * fontScale
		} else {
			w += float32(f.GlyphWidth) * fontScale
		}
		if float32(chData.advanceHeight) > h {
			h = float32(chData.advanceHeight)
		}
	}

	h = h * fontScale

	return w, float32(f.GlyphHeight) * fontScale, h
}

// OffsetFloor returns the maximum width offset that will fit between characters that
// is still smaller than the offset passed in.
func (f *Font) OffsetFloor(msg string, offset float32) float32 {
	var w float32

	// see how much to scale the size based on current resolution vs desgin resolution
	fontScale := f.GetCurrentScale()

	for _, ch := range msg {
		// get the rune data
		chData := f.locations[ch]

		// break if we go over the distance
		if w+(float32(chData.advanceWidth)*fontScale) > offset {
			break
		}
		w += float32(chData.advanceWidth) * fontScale
	}

	return w
}

// OffsetForIndex returns the width offset that will fit just before the `stopIndex`
// number character in the msg.
func (f *Font) OffsetForIndex(msg string, stopIndex int) float32 {
	var w float32

	// see how much to scale the size based on current resolution vs desgin resolution
	fontScale := f.GetCurrentScale()

	for i, ch := range msg {
		// calculate up to the stopIndex but do not include it
		if i >= stopIndex {
			break
		}

		// get the rune data
		chData := f.locations[ch]

		// break if we go over the distance
		w += float32(chData.advanceWidth) * fontScale
	}

	return w
}

// TextRenderData is a structure containing the raw OpenGL VBO data needed
// to render a text string for a given texture.
type TextRenderData struct {
	ComboBuffer   []float32 // the combo VBO data (vert/uv/color)
	IndexBuffer   []uint32  // the element index VBO data
	Faces         uint32    // the number of faces in the text string
	Width         float32   // the width in pixels of the text string
	Height        float32   // the height in pixels of the text string
	AdvanceHeight float32   // the amount of pixels to move the pen in the verticle direction
}

// CreateText makes a new renderable object from the supplied string
// using the data in the font. The data is returned as a TextRenderData object.
func (f *Font) CreateText(pos mgl.Vec3, color mgl.Vec4, msg string) TextRenderData {
	// this is the texture ID of the font to use in the shader; by default
	// the library always binds the font to the first texture sampler.
	const floatTexturePosition = 0.0

	// get the length of our message
	msgLength := len(msg)

	// create the arrays to hold the data to buffer to OpenGL
	comboBuffer := make([]float32, 0, msgLength*(2+2+4)*4) // pos, uv, color4
	indexBuffer := make([]uint32, 0, msgLength*6)          // two faces * three indexes

	// do a preliminary test to see how much room the message will take up
	dimX, dimY, advH := f.GetRenderSize(msg)

	// see how much to scale the size based on current resolution vs desgin resolution
	fontScale := f.GetCurrentScale()

	// loop through the message
	var penX = pos[0]
	var penY = pos[1] - float32(advH)
	for chi, ch := range msg {
		// get the rune data
		chData := f.locations[ch]

		// setup the coordinates for ther vetexes
		x0 := penX
		y0 := penY - float32(f.GlyphHeight-chData.advanceHeight)*fontScale
		x1 := x0 + float32(f.GlyphWidth)*fontScale
		y1 := y0 + float32(f.GlyphHeight)*fontScale
		s0 := chData.uvMinX
		t0 := chData.uvMinY
		s1 := chData.uvMaxX
		t1 := chData.uvMaxY

		// set the vertex data
		comboBuffer = append(comboBuffer, x1)
		comboBuffer = append(comboBuffer, y0)
		comboBuffer = append(comboBuffer, s1)
		comboBuffer = append(comboBuffer, t0)
		comboBuffer = append(comboBuffer, floatTexturePosition)
		comboBuffer = append(comboBuffer, color[:]...)

		comboBuffer = append(comboBuffer, x1)
		comboBuffer = append(comboBuffer, y1)
		comboBuffer = append(comboBuffer, s1)
		comboBuffer = append(comboBuffer, t1)
		comboBuffer = append(comboBuffer, floatTexturePosition)
		comboBuffer = append(comboBuffer, color[:]...)

		comboBuffer = append(comboBuffer, x0)
		comboBuffer = append(comboBuffer, y1)
		comboBuffer = append(comboBuffer, s0)
		comboBuffer = append(comboBuffer, t1)
		comboBuffer = append(comboBuffer, floatTexturePosition)
		comboBuffer = append(comboBuffer, color[:]...)

		comboBuffer = append(comboBuffer, x0)
		comboBuffer = append(comboBuffer, y0)
		comboBuffer = append(comboBuffer, s0)
		comboBuffer = append(comboBuffer, t0)
		comboBuffer = append(comboBuffer, floatTexturePosition)
		comboBuffer = append(comboBuffer, color[:]...)

		startIndex := uint32(chi) * 4
		indexBuffer = append(indexBuffer, startIndex)
		indexBuffer = append(indexBuffer, startIndex+1)
		indexBuffer = append(indexBuffer, startIndex+2)

		indexBuffer = append(indexBuffer, startIndex+2)
		indexBuffer = append(indexBuffer, startIndex+3)
		indexBuffer = append(indexBuffer, startIndex)

		// advance the pen
		penX += float32(chData.advanceWidth) * fontScale
	}

	return TextRenderData{
		ComboBuffer:   comboBuffer,
		IndexBuffer:   indexBuffer,
		Faces:         uint32(msgLength * 2),
		Width:         float32(dimX),
		Height:        float32(dimY),
		AdvanceHeight: float32(advH),
	}
}

// loadRGBAToTexture takes a byte slice and throws it into an OpenGL texture.
func (f *Font) loadRGBAToTexture(rgba []byte, imageSize int32) graphics.Texture {
	return f.loadRGBAToTextureExt(rgba, imageSize, graphics.LINEAR, graphics.LINEAR, graphics.CLAMP_TO_EDGE, graphics.CLAMP_TO_EDGE)
}

// loadRGBAToTextureExt takes a byte slice and throws it into an OpenGL texture.
func (f *Font) loadRGBAToTextureExt(rgba []byte, imageSize, magFilter, minFilter, wrapS, wrapT int32) graphics.Texture {
	tex := f.Owner.gfx.GenTexture()
	f.Owner.gfx.ActiveTexture(graphics.TEXTURE0)
	f.Owner.gfx.BindTexture(graphics.TEXTURE_2D, tex)
	f.Owner.gfx.TexParameteri(graphics.TEXTURE_2D, graphics.TEXTURE_MAG_FILTER, magFilter)
	f.Owner.gfx.TexParameteri(graphics.TEXTURE_2D, graphics.TEXTURE_MIN_FILTER, minFilter)
	f.Owner.gfx.TexParameteri(graphics.TEXTURE_2D, graphics.TEXTURE_WRAP_S, wrapS)
	f.Owner.gfx.TexParameteri(graphics.TEXTURE_2D, graphics.TEXTURE_WRAP_T, wrapT)
	f.Owner.gfx.TexImage2D(graphics.TEXTURE_2D, 0, graphics.RGBA, imageSize, imageSize, 0, graphics.RGBA, graphics.UNSIGNED_BYTE, f.Owner.gfx.Ptr(rgba), len(rgba))
	return tex
}
