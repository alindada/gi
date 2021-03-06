// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gi3d"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/mat32"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

// AnimateTicker is the time.Ticker for animating the scene
var AnimateTicker *time.Ticker

var DoAnimation = false

var TheScene *gi3d.Scene

func Animate() {
	torAng := float32(0)
	var torPosOrig, gophPosOrig mat32.Vec3
	for {
		if AnimateTicker == nil || TheScene == nil {
			return
		}
		<-AnimateTicker.C // wait for tick
		if !DoAnimation || TheScene == nil {
			continue
		}
		torusi := TheScene.ChildByName("torus", 0)
		if torusi == nil {
			continue
		}
		torus := torusi.(*gi3d.Solid)
		if torPosOrig.IsNil() {
			torPosOrig = torus.Pose.Pos
		}
		ggp := TheScene.ChildByName("go-group", 0)
		if ggp == nil {
			continue
		}
		gophi := ggp.Child(1)
		if gophi == nil {
			continue
		}
		goph := gophi.(*gi3d.Group)
		if gophPosOrig.IsNil() {
			gophPosOrig = goph.Pose.Pos
		}
		updt := TheScene.UpdateStart()
		radius := float32(0.3)
		tdx := radius * mat32.Cos(torAng)
		tdz := radius * mat32.Sin(torAng)

		gdx := 0.1 * radius * mat32.Cos(torAng+math.Pi)
		gdz := 0.1 * radius * mat32.Sin(torAng+math.Pi)

		torAng += .1
		tp := torPosOrig
		tp.X += tdx
		tp.Z += tdz
		torus.Pose.Pos = tp

		gp := gophPosOrig
		gp.X += gdx
		gp.Z += gdz
		goph.Pose.Pos = gp

		TheScene.UpdateEnd(updt) // triggers re-render -- don't need a full Update() which updates meshes
	}
}

func mainrun() {
	width := 1024
	height := 768

	// turn these on to see a traces of various stages of processing..
	// ki.SignalTrace = true
	// gi.WinEventTrace = true
	// gi3d.Update3DTrace = true
	// gi.Update2DTrace = true

	rec := ki.Node{}          // receiver for events
	rec.InitName(&rec, "rec") // this is essential for root objects not owned by other Ki tree nodes

	gi.SetAppName("gi3d")
	gi.SetAppAbout(`This is a demo of the 3D graphics aspect of the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>.
<p>The <a href="https://github.com/goki/gi/blob/master/examples/gi3d/README.md">README</a> page for this example app has further info.</p>`)

	win := gi.NewMainWindow("gogi-gi3d-demo", "GoGi 3D Demo", width, height)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.SetProp("spacing", units.NewEx(1))

	trow := gi.AddNewLayout(mfr, "trow", gi.LayoutHoriz)
	trow.SetStretchMaxWidth()

	title := gi.AddNewLabel(trow, "title", `This is a demonstration of the
<a href="https://github.com/goki/gi/gi">GoGi</a> <i>3D</i> Framework<br>
See <a href="https://github.com/goki/gi/blob/master/examples/gi3d/README.md">README</a> for detailed info and things to try.`)
	title.SetProp("white-space", gi.WhiteSpaceNormal) // wrap
	title.SetProp("text-align", gi.AlignCenter)       // note: this also sets horizontal-align, which controls the "box" that the text is rendered in..
	title.SetProp("vertical-align", gi.AlignCenter)
	title.SetProp("font-size", "x-large")
	title.SetProp("line-height", 1.5)
	title.SetStretchMax()

	//////////////////////////////////////////
	//    Scene

	gi.AddNewSpace(mfr, "scspc")
	scrow := gi.AddNewLayout(mfr, "scrow", gi.LayoutHoriz)
	scrow.SetStretchMax()

	// gi.AddNewLabel(scrow, "tmp", "This is test text")

	scvw := gi3d.AddNewSceneView(scrow, "sceneview")
	scvw.SetStretchMax()
	scvw.Config()
	sc := scvw.Scene()
	TheScene = sc

	// first, add lights, set camera
	sc.BgColor.SetUInt8(230, 230, 255, 255) // sky blue-ish
	gi3d.AddNewAmbientLight(sc, "ambient", 0.3, gi3d.DirectSun)

	dir := gi3d.AddNewDirLight(sc, "dir", 1, gi3d.DirectSun)
	dir.Pos.Set(0, 2, 1) // default: 0,1,1 = above and behind us (we are at 0,0,X)

	// point := gi3d.AddNewPointLight(sc, "point", 1, gi3d.DirectSun)
	// point.Pos.Set(0, 5, 5)

	// spot := gi3d.AddNewSpotLight(sc, "spot", 1, gi3d.DirectSun)
	// spot.Pose.Pos.Set(0, 5, 5)

	cbm := gi3d.AddNewBox(sc, "cube1", 1, 1, 1)
	// cbm.Segs.Set(10, 10, 10) // not clear if any diff really..

	rbgp := gi3d.AddNewGroup(sc, sc, "r-b-group")

	// style sheet
	var css = ki.Props{
		".cube": ki.Props{
			"shiny": 20,
		},
	}
	sc.CSS = css

	rcb := gi3d.AddNewSolid(sc, rbgp, "red-cube", cbm.Name())
	rcb.Class = "cube"
	rcb.Pose.Pos.Set(-1, 0, 0)
	rcb.SetProp("color", "red")
	rcb.SetProp("shiny", 500) // note: this will be overridden by the css sheet
	rcb.Mat.Color.SetName("red")

	bcb := gi3d.AddNewSolid(sc, rbgp, "blue-cube", cbm.Name())
	bcb.Class = "cube"
	bcb.Pose.Pos.Set(1, 1, 0)
	bcb.Pose.Scale.X = 2
	bcb.Mat.Color.SetName("blue")
	bcb.Mat.Shiny = 10 // note: this will be overridden by the css sheet

	bcb.Mat.Specular.SetName("blue") // how you get rid of specular highlights

	gcb := gi3d.AddNewSolid(sc, sc, "green-trans-cube", cbm.Name())
	gcb.Pose.Pos.Set(0, 0, 1)
	gcb.Mat.Color.SetUInt8(0, 255, 0, 128) // alpha = .5
	gcb.Class = "cube"

	lnsm := gi3d.AddNewLines(sc, "Lines", []mat32.Vec3{mat32.Vec3{-3, -1, 0}, mat32.Vec3{-2, 1, 0}, mat32.Vec3{2, 1, 0}, mat32.Vec3{3, -1, 0}}, mat32.Vec2{.2, .1}, gi3d.CloseLines)
	lns := gi3d.AddNewSolid(sc, sc, "hi-line", lnsm.Name())
	lns.Pose.Pos.Set(0, 0, 1)
	lns.Mat.Color.SetUInt8(255, 255, 0, 128) // alpha = .5
	// sc.Wireframe = true                      // debugging

	// this line should go from lower left front of red cube to upper vertex of above hi-line
	cyan := gi.Color{}
	cyan.SetUInt8(0, 255, 255, 255)
	gi3d.AddNewLine(sc, sc, "UnitLineW.05", "one-line", mat32.Vec3{-1.5, -.5, .5}, mat32.Vec3{2, 1, 1}, .05, cyan)

	// bbclr := gi.Color{}
	// bbclr.SetUInt8(255, 255, 0, 255)
	// gi3d.AddNewLineBox(sc, sc, "bbox", "bbox", mat32.Box3{Min: mat32.Vec3{-2, -2, -1}, Max: mat32.Vec3{-1, -1, .5}}, .01, bbclr, gi3d.Active)

	cylm := gi3d.AddNewCylinder(sc, "cylinder", 1.5, .5, 32, 1, true, true)
	cyl := gi3d.AddNewSolid(sc, sc, "cylinder", cylm.Name())
	cyl.Pose.Pos.Set(-2.25, 0, 0)

	capm := gi3d.AddNewCapsule(sc, "capsule", 1.5, .5, 32, 1)
	caps := gi3d.AddNewSolid(sc, sc, "capsule", capm.Name())
	caps.Pose.Pos.Set(3.25, 0, 0)
	caps.Mat.Color.SetName("tan")

	sphm := gi3d.AddNewSphere(sc, "sphere", .75, 32)
	sph := gi3d.AddNewSolid(sc, sc, "sphere", sphm.Name())
	sph.Pose.Pos.Set(0, -2, 0)
	sph.Mat.Color.SetName("orange")
	sph.Mat.Color.A = 200

	// Good strategy for objects if used in multiple places is to load
	// into library, then add from there.
	lgo, err := sc.OpenToLibrary("gopher.obj", "")
	if err != nil {
		log.Println(err)
	}
	lgo.Pose.SetAxisRotation(0, 1, 0, -90) // for all cases

	gogp := gi3d.AddNewGroup(sc, sc, "go-group")

	bgo, _ := sc.AddFmLibrary("gopher", gogp)
	bgo.Pose.Scale.Set(.5, .5, .5)
	bgo.Pose.Pos.Set(1.4, -2.5, 0)
	bgo.Pose.SetAxisRotation(0, 1, 0, -160)

	sgo, _ := sc.AddFmLibrary("gopher", gogp)
	sgo.Pose.Pos.Set(-1.5, -2, 0)
	sgo.Pose.Scale.Set(.2, .2, .2)

	trsm := gi3d.AddNewTorus(sc, "torus", .75, .1, 32)
	trs := gi3d.AddNewSolid(sc, sc, "torus", trsm.Name())
	trs.Pose.Pos.Set(-1.6, -1.6, -.2)
	trs.Pose.SetAxisRotation(1, 0, 0, 90)
	trs.Mat.Color.SetName("white")
	trs.Mat.Color.A = 200

	grtx := gi3d.AddNewTextureFile(sc, "ground", "ground.png")
	// wdtx := gi3d.AddNewTextureFile(sc, "wood", "wood.png")

	floorp := gi3d.AddNewPlane(sc, "floor-plane", 100, 100)
	floor := gi3d.AddNewSolid(sc, sc, "floor", floorp.Name())
	floor.Pose.Pos.Set(0, -5, 0)
	floor.Mat.Color.SetName("tan")
	// // floor.Mat.Emissive.SetName("brown")
	// floor.Mat.Bright = 2 // .5 for wood / brown
	floor.Mat.SetTexture(sc, grtx)
	floor.Mat.Tiling.Repeat.Set(40, 40)
	floor.SetInactive() // not selectable

	txt := gi3d.AddNewText2D(sc, sc, "text", "Text2D can put <b>HTML</b> formatted<br>Text anywhere you might <i>want</i>")
	// 	txt.SetProp("background-color", gi.Color{0, 0, 0, 0}) // transparent -- default
	// txt.SetProp("background-color", "white")
	// txt.SetProp("color", "black") // default
	// txt.SetProp("margin", units.NewPt(4)) // default is 2 px
	// txt.Mat.Bright = 5 // no dim text -- key if using a background and want it to be bright..
	txt.Pose.Scale.SetScalar(0.2)
	txt.Pose.Pos.Set(0, 2.2, 0)

	tcg := gi3d.AddNewGroup(sc, sc, gi3d.TrackCameraName) // automatically tracks camera -- FPS effect
	fpgun := gi3d.AddNewSolid(sc, tcg, "first-person-gun", cbm.Name())
	fpgun.Pose.Scale.Set(.1, .1, 1)
	fpgun.Pose.Pos.Set(.5, -.5, -2.5)          // in front of camera
	fpgun.Mat.Color.SetUInt8(255, 0, 255, 128) // alpha = .5

	sc.Camera.Pose.Pos.Set(0, 0, 10)              // default position
	sc.Camera.LookAt(mat32.Vec3Zero, mat32.Vec3Y) // defaults to looking at origin

	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "Window"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu.AddAppMenu(win)
	win.MainMenuUpdated()

	abut := gi.AddNewCheckBox(mfr, "anim-but")
	abut.SetText("Animate")
	abut.ButtonSig.Connect(rec.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			DoAnimation = !abut.IsChecked() // note: not yet updated so status is opposite!
		}
	})

	emb := gi3d.AddNewEmbed2D(sc, sc, "embed-but", 100, 50)
	emb.Pose.Pos.Set(-2, 2, 0)
	// eabut := gi.AddNewCheckBox(emb.Viewport, "anim-but")
	eabut := gi.AddNewButton(emb.Viewport, "anim-but")
	eabut.SetText("Animate")
	eabut.ButtonSig.Connect(rec.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			fmt.Printf("embedded button!")
			// DoAnimation = !eabut.IsChecked() // note: not yet updated so status is opposite!
		}
	})

	AnimateTicker = time.NewTicker(10 * time.Millisecond)
	go Animate()

	vp.UpdateEndNoSig(updt)
	win.StartEventLoop()
}
