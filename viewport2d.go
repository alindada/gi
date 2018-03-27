// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "fmt"
	"github.com/rcoreilly/goki/ki"
	// "golang.org/x/image/font"
	"image"
	"image/draw"
	"image/png"
	"io"
	// "log"
	"os"
	"reflect"
)

// A Viewport ALWAYS presents its children with a 0,0 - (Size.X, Size.Y)
// rendering area even if it is itself a child of another Viewport.  This is
// necessary for rendering onto the image that it provides.  This creates
// challenges for managing the different geometries in a coherent way, e.g.,
// events come through the Window in terms of the root VP coords.  Thus, nodes
// require a  WinBBox for events and a VpBBox for their parent Viewport.

// Viewport2D provides an image and a stack of Paint contexts for drawing onto the image
// with a convenience forwarding of the Paint methods operating on the current Paint
type Viewport2D struct {
	Node2DBase
	Fill    bool        `desc:"fill the viewport with background-color from style"`
	ViewBox ViewBox2D   `xml:"viewBox" desc:"viewbox within any parent Viewport2D"`
	Render  RenderState `json:"-" desc:"render state for rendering"`
	Pixels  *image.RGBA `json:"-" desc:"pixels that we render into"`
	Backing *image.RGBA `json:"-" desc:"if non-nil, this is what goes behind our image -- copied from our region in parent image -- allows us to re-render cleanly into parent, even with transparency"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Viewport2D = ki.Types.AddType(&Viewport2D{}, nil)

// NewViewport2D creates a new image.RGBA with the specified width and height
// and prepares a context for rendering onto that image.
func NewViewport2D(width, height int) *Viewport2D {
	return NewViewport2DForRGBA(image.NewRGBA(image.Rect(0, 0, width, height)))
}

// NewViewport2DForImage copies the specified image into a new image.RGBA
// and prepares a context for rendering onto that image.
func NewViewport2DForImage(im image.Image) *Viewport2D {
	return NewViewport2DForRGBA(imageToRGBA(im))
}

// NewViewport2DForRGBA prepares a context for rendering onto the specified image.
// No copy is made.
func NewViewport2DForRGBA(im *image.RGBA) *Viewport2D {
	vp := &Viewport2D{
		ViewBox: ViewBox2D{Size: image.Point{im.Bounds().Size().X, im.Bounds().Size().Y}},
		Pixels:  im,
	}
	vp.Render.Image = vp.Pixels
	return vp
}

// resize viewport, creating a new image (no point in trying to resize the image -- need to re-render) -- updates ViewBox Size too -- triggers update -- wrap in other UpdateStart/End calls as appropriate
func (vp *Viewport2D) Resize(width, height int) {
	if vp.Pixels.Bounds().Size().X == width && vp.Pixels.Bounds().Size().Y == height {
		return // already good
	}
	vp.UpdateStart()
	vp.Pixels = image.NewRGBA(image.Rect(0, 0, width, height))
	vp.Render.Image = vp.Pixels
	vp.ViewBox.Size = image.Point{width, height}
	vp.UpdateEnd()
	vp.FullRender2DTree()
}

////////////////////////////////////////////////////////////////////////////////////////
//  Main Rendering code

// draw our image into parents -- called at right place in Render
func (vp *Viewport2D) DrawIntoParent(parVp *Viewport2D) {
	r := vp.ViewBox.Bounds()
	if vp.Backing != nil {
		draw.Draw(parVp.Pixels, r, vp.Backing, image.ZP, draw.Src)
	}
	draw.Draw(parVp.Pixels, r, vp.Pixels, image.ZP, draw.Src)
}

// copy our backing image from parent -- called at right place in Render
func (vp *Viewport2D) CopyBacking(parVp *Viewport2D) {
	r := vp.ViewBox.Bounds()
	if vp.Backing == nil {
		vp.Backing = image.NewRGBA(vp.ViewBox.SizeRect())
	}
	draw.Draw(vp.Backing, r, parVp.Pixels, image.ZP, draw.Src)
}

func (vp *Viewport2D) DrawIntoWindow() {
	wini := vp.FindParentByType(reflect.TypeOf(Window{}))
	if wini != nil {
		win := (wini).(*Window)
		// width, height := win.Win.Size() // todo: update size of our window
		s := win.Win.Screen()
		s.CopyRGBA(vp.Pixels, vp.Pixels.Bounds())
		win.Win.FlushImage()
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// Node2D interface

func (vp *Viewport2D) AsNode2D() *Node2DBase {
	return &vp.Node2DBase
}

func (vp *Viewport2D) AsViewport2D() *Viewport2D {
	return vp
}

func (g *Viewport2D) AsLayout2D() *Layout {
	return nil
}

func (vp *Viewport2D) Init2D() {
	vp.Init2DBase()
	// we update oursleves whenever any node update event happens
	vp.NodeSig.Connect(vp.This, func(recvp, sendvp ki.Ki, sig int64, data interface{}) {
		rvpi, _ := KiToNode2D(recvp)
		rvp := rvpi.AsViewport2D()
		// fmt.Printf("viewport: %v rendering due to signal: %v from node: %v\n", rvp.PathUnique(), ki.NodeSignals(sig), sendvp.PathUnique())
		// todo: don't re-render if deleting!
		rvp.FullRender2DTree()
	})
}

func (vp *Viewport2D) Style2D() {
	vp.Style2DWidget()
}

func (vp *Viewport2D) Size2D() {
	vp.InitLayout2D()
	vp.LayData.AllocSize.SetFromPoint(vp.ViewBox.Size)
}

func (vp *Viewport2D) Layout2D(parBBox image.Rectangle) {
	// viewport ignores any parent parent bbox info!
	psize := vp.AddParentPos()
	vp.VpBBox = vp.Pixels.Bounds()
	vp.SetWinBBox()                    // still add offsets
	vp.Style.SetUnitContext(vp, psize) // update units with final layout
	vp.Paint.SetUnitContext(vp, psize) // always update paint
	vp.Layout2DChildren()
}

func (vp *Viewport2D) BBox2D() image.Rectangle {
	return vp.Pixels.Bounds() // not sure about: ViewBox.Bounds()
}

func (g *Viewport2D) ChildrenBBox2D() image.Rectangle {
	return g.VpBBox
}

func (vp *Viewport2D) RenderViewport2D() {
	if vp.Viewport != nil {
		vp.CopyBacking(vp.Viewport) // full re-render is when we copy the backing
		vp.DrawIntoParent(vp.Viewport)
	} else { // top-level, try drawing into window
		vp.DrawIntoWindow()
	}
}

// we use our own render for these -- Viewport member is our parent!
func (vp *Viewport2D) PushBounds() bool {
	if vp.VpBBox.Empty() {
		return false
	}
	rs := &vp.Render
	rs.PushBounds(vp.VpBBox)
	return true
}

func (vp *Viewport2D) PopBounds() {
	rs := &vp.Render
	rs.PopBounds()
}

func (vp *Viewport2D) Render2D() {
	if vp.PushBounds() {
		if vp.Fill {
			pc := &vp.Paint
			pc.FillStyle.SetColor(&vp.Style.Background.Color)
			pc.StrokeStyle.SetColor(nil)
			pc.DrawRectangle(&vp.Render, 0.0, 0.0, float64(vp.ViewBox.Size.X),
				float64(vp.ViewBox.Size.Y))
		}
		vp.Render2DChildren() // we must do children first, then us!
		vp.RenderViewport2D() // update our parent image
		vp.PopBounds()
	}
}

func (vp *Viewport2D) CanReRender2D() bool {
	return true // always true for viewports
}

func (g *Viewport2D) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &Viewport2D{}

////////////////////////////////////////////////////////////////////////////////////////
//  Signal Handling

// each node calls this signal method to notify its parent viewport whenever it changes, causing a re-render
func SignalViewport2D(vpki, node ki.Ki, sig int64, data interface{}) {
	vpgi, ok := vpki.(Node2D)
	if !ok {
		return
	}
	vp := vpgi.AsViewport2D()
	if vp == nil { // should not happen -- should only be called on viewports
		return
	}
	gii, gi := KiToNode2D(node)
	if gii == nil { // should not happen
		return
	}
	// fmt.Printf("viewport: %v rendering due to signal: %v from node: %v\n", vp.PathUnique(), ki.NodeSignals(sig), node.PathUnique())

	// todo: probably need better ways of telling how much re-rendering is needed
	if ki.NodeSignalAnyMod(sig) {
		vp.FullRender2DTree()
	} else if ki.NodeSignalAnyUpdate(sig) {
		if gii.CanReRender2D() {
			vp.ReRender2DNode(gii)
		} else {
			gi.Style2DTree()    // restyle only from affected node downward
			vp.ReRender2DTree() // need to re-render entirely from us
		}
	}
	// don't do anything on deleting or destroying, and
}

////////////////////////////////////////////////////////////////////////////////////////
// Root-level Viewport API -- does all the recursive calls

// re-render a specific node that has said it can re-render
func (vp *Viewport2D) ReRender2DNode(gni Node2D) {
	gn := gni.AsNode2D()
	gn.Render2DTree()
	vp.RenderViewport2D()
}

// SavePNG encodes the image as a PNG and writes it to disk.
func (vp *Viewport2D) SavePNG(path string) error {
	return SavePNG(path, vp.Pixels)
}

// EncodePNG encodes the image as a PNG and writes it to the provided io.Writer.
func (vp *Viewport2D) EncodePNG(w io.Writer) error {
	return png.Encode(w, vp.Pixels)
}

// todo:

// DrawPoint is like DrawCircle but ensures that a circle of the specified
// size is drawn regardless of the current transformation matrix. The position
// is still transformed, but not the shape of the point.
// func (vp *Viewport2D) DrawPoint(x, y, r float64) {
// 	pc := vp.PushNewPaint()
// 	p := pc.TransformPoint(x, y)
// 	pc.Identity()
// 	pc.DrawCircle(p.X, p.Y, r)
// 	vp.PopPaint()
// }

//////////////////////////////////////////////////////////////////////////////////
//  Image utilities

func LoadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	im, _, err := image.Decode(file)
	return im, err
}

func LoadPNG(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return png.Decode(file)
}

func SavePNG(path string, im image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, im)
}

func imageToRGBA(src image.Image) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Rect, src, image.ZP, draw.Src)
	return dst
}
