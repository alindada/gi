// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glos

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/goki/gi/mat32"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver/internal/drawer"
	"github.com/goki/gi/oswin/gpu"
)

// note: use a different interface for different formats of "textures" such as a
// a depth buffer, and have ways of converting between.  Texture2D is always
// RGBA picture texture

// textureImpl manages a texture, including loading from an image file
// and activating on GPU
type textureImpl struct {
	init   bool
	handle uint32
	name   string
	size   image.Point
	img    *image.RGBA // when loaded
	fbuff  gpu.Framebuffer
	// magFilter uint32 // magnification filter
	// minFilter uint32 // minification filter
	// wrapS     uint32 // wrap mode for s coordinate
	// wrapT     uint32 // wrap mode for t coordinate
}

// Name returns the name of the texture (filename without extension
// by default)
func (tx *textureImpl) Name() string {
	return tx.name
}

// SetName sets the name of the texture
func (tx *textureImpl) SetName(name string) {
	tx.name = name
}

// Open loads texture image from file.
// format inferred from filename -- JPEG and PNG
// supported by default.
func (tx *textureImpl) Open(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	im, _, err := image.Decode(file)
	if err != nil {
		return err
	}
	return tx.SetImage(im)
}

// Image returns the image -- typically as an *image.RGBA
// If this Texture has been Activate'd then this retrieves
// the current contents of the Texture, e.g., if it has been
// used as a rendering target.
// If Activate()'d, then must be called with a valid gpu context
// and on proper thread for that context.
func (tx *textureImpl) Image() image.Image {
	if !tx.init {
		if tx.img == nil {
			return nil
		}
		return tx.img
	}
	// todo: get image from buffer
	return tx.img
}

func rgbaImage(img image.Image) (*image.RGBA, error) {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba, nil
	} else {
		// Converts image to RGBA format
		rgba := image.NewRGBA(img.Bounds())
		if rgba.Stride != rgba.Rect.Size().X*4 {
			return nil, fmt.Errorf("glos Texture2D: unsupported stride")
		}
		draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)
		return rgba, nil
	}
}

// SetImage sets entire contents of the Texture from given image
// (including setting the size of the texture from that of the img).
// This is most efficiently done using an image.RGBA, but other
// formats will be converted as necessary.
// Can be called prior to doing Activate(), in which case the image
// pixels will then initialize the GPU version of the texture.
// If called after Activate then the image is copied up to the GPU
// and texture is left in an Activate state.
// If Activate()'d, then must be called with a valid gpu context
// and on proper thread for that context.
func (tx *textureImpl) SetImage(img image.Image) error {
	rgba, err := rgbaImage(img)
	if err != nil {
		return err
	}
	tx.img = rgba
	tx.size = rgba.Rect.Size()
	if tx.init {
		tx.Delete()
		tx.Activate(0)
	}
	return nil
}

// SetSubImage uploads the sub-Image defined by src and sr to the texture.
// such that sr.Min in src-space aligns with dp in dst-space.
// The textures's contents are overwritten; the draw operator
// is implicitly draw.Src. Texture must be Activate'd to the GPU for this
// to proceed -- if Activate() has not yet been called, it will be (on texture 0).
// Must be called with a valid gpu context and on proper thread for that context.
func (tx *textureImpl) SetSubImage(dp image.Point, src image.Image, sr image.Rectangle) error {
	rgba, err := rgbaImage(src)
	if err != nil {
		return err
	}

	// todo: if needed for windows, do this here:
	// buf := src.(*imageImpl)
	// buf.preUpload()

	// src2dst is added to convert from the src coordinate space to the dst
	// coordinate space. It is subtracted to convert the other way.
	src2dst := dp.Sub(sr.Min)

	// Clip to the source.
	sr = sr.Intersect(rgba.Bounds())

	// Clip to the destination.
	dr := sr.Add(src2dst)
	dr = dr.Intersect(tx.Bounds())
	if dr.Empty() {
		return nil
	}

	// Bring dr.Min in dst-space back to src-space to get the pixel image offset.
	pix := rgba.Pix[rgba.PixOffset(dr.Min.X-src2dst.X, dr.Min.Y-src2dst.Y):]

	tx.Activate(0)

	width := dr.Dx()
	if width*4 == rgba.Stride {
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, int32(dr.Min.X), int32(dr.Min.Y), int32(width), int32(dr.Dy()), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pix))
		gpu.TheGPU.ErrCheck("tex subimg")
		fmt.Printf("uploaded tex: dr: %+v\n", dr)
		return nil
	}
	// TODO: can we use GL_UNPACK_ROW_LENGTH with glPixelStorei for stride in
	// ES 3.0, instead of uploading the pixels row-by-row?
	for y, p := dr.Min.Y, 0; y < dr.Max.Y; y++ {
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, int32(dr.Min.X), int32(y), int32(width), 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pix[p:]))
		p += rgba.Stride
	}
	fmt.Printf("uploaded tex: dr: %+v\n", dr)
	return nil
}

// Size returns the size of the image
func (tx *textureImpl) Size() image.Point {
	return tx.size
}

func (tx *textureImpl) Bounds() image.Rectangle {
	return image.Rectangle{Max: tx.size}
}

// SetSize sets the size of the texture.
// If texture has been Activate'd, then this resizes the GPU side as well.
// If Activate()'d, then must be called with a valid gpu context
// and on proper thread for that context.
func (tx *textureImpl) SetSize(size image.Point) {
	if tx.size == size {
		return
	}
	wasInit := tx.init
	if tx.init {
		tx.Delete()
	}
	tx.size = size
	tx.img = nil
	if wasInit {
		tx.Activate(0)
	}
}

// Activate establishes the GPU resources and handle for the
// texture, using the given texture number (0-31 range).
// If an image has already been set for this texture, then it is
// copied up to the GPU at this point -- otherwise the texture
// is nil initialized.
// Must be called with a valid gpu context and on proper thread for that context.
func (tx *textureImpl) Activate(texNo int) {
	if !tx.init {
		gl.GenTextures(1, &tx.handle)
		gl.ActiveTexture(gl.TEXTURE0 + uint32(texNo))
		gl.BindTexture(gl.TEXTURE_2D, tx.handle)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		szx := int32(tx.size.X)
		szy := int32(tx.size.Y)
		if tx.img != nil {
			gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, szx, szy, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(tx.img.Pix))
		} else {
			gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, szx, szy, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(nil))
		}
		tx.init = true
	} else {
		gl.ActiveTexture(gl.TEXTURE0 + uint32(texNo))
		gl.BindTexture(gl.TEXTURE_2D, tx.handle)
	}
}

// Handle returns the GPU handle for the texture -- only
// valid after Activate
func (tx *textureImpl) Handle() uint32 {
	return tx.handle
}

// Delete deletes the GPU resources associated with this texture
// (requires Activate to re-establish a new one).
// Should be called prior to Go object being deleted
// (ref counting can be done externally).
// Must be called with a valid gpu context and on proper thread for that context.
func (tx *textureImpl) Delete() {
	if !tx.init {
		return
	}
	gl.DeleteTextures(1, &tx.handle)
	tx.init = false
}

// ActivateFramebuffer creates a gpu.Framebuffer for rendering onto
// this texture (if not already created) and activates it for
// rendering.  The gpu.Texture2D interface can provide direct access
// to the created framebuffer.
// Call gpu.TheGPU.RenderToWindow() or DeActivateFramebuffer
// to return to window rendering.
// Must be called with a valid gpu context and on proper thread for that context.
func (tx *textureImpl) ActivateFramebuffer() {
	tx.Activate(0)
	if tx.fbuff == nil {
		tx.fbuff = theGPU.NewFramebuffer("", tx.size, 0)
		tx.fbuff.SetTexture(tx)
	}
	tx.fbuff.Activate()
}

func (tx *textureImpl) Framebuffer() gpu.Framebuffer {
	return tx.fbuff
}

// DeActivateFramebuffer de-activates this texture's framebuffer
// for rendering (just calls gpu.TheGPU.RenderToWindow())
// Must be called with a valid gpu context and on proper thread for that context.
func (tx *textureImpl) DeActivateFramebuffer() {
	theGPU.RenderToWindow()
}

// DeleteFramebuffer deletes this Texture's framebuffer
// created during ActivateFramebuffer.
// Must be called with a valid gpu context and on proper thread for that context.
func (tx *textureImpl) DeleteFramebuffer() {
	if tx.fbuff != nil {
		tx.fbuff.Delete()
		tx.fbuff = nil
	}
}

////////////////////////////////////////////////
//   Drawer wrappers

func (tx *textureImpl) Draw(src2dst mat32.Matrix3, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	sz := tx.Size()
	tx.ActivateFramebuffer()
	theApp.draw(sz, src2dst, src, sr, op, opts)
	tx.DeActivateFramebuffer()
}

func (tx *textureImpl) DrawUniform(src2dst mat32.Matrix3, src color.Color, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	sz := tx.Size()
	tx.ActivateFramebuffer()
	theApp.drawUniform(sz, src2dst, src, sr, op, opts)
	tx.DeActivateFramebuffer()
}

func (tx *textureImpl) Copy(dp image.Point, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	drawer.Copy(tx, dp, src, sr, op, opts)
}

func (tx *textureImpl) Scale(dr image.Rectangle, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	drawer.Scale(tx, dr, src, sr, op, opts)
}

func (tx *textureImpl) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
	sz := tx.Size()
	tx.ActivateFramebuffer()
	theApp.fillRect(sz, dr, src, op)
	tx.DeActivateFramebuffer()
}