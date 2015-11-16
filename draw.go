package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"

	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
)

type img struct {
	filename         string
	w, h             int
	centerX, centerY float64
	levelWidth       int
	rgba             *image.RGBA
	ctx              *draw2dimg.GraphicContext
}

func newImg(filename string, w, h, levels int) img {
	var levelWidth float64
	if w > h {
		levelWidth = float64(h) / 2 / float64(levels)
	} else {
		levelWidth = float64(w) / 2 / float64(levels)
	}

	if levelWidth < 1 {
		log.Panicf("level width is too small! %f", levelWidth)
	}

	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	ctx := draw2dimg.NewGraphicContext(rgba)

	return img{
		filename:   filename,
		w:          w,
		h:          h,
		centerX:    float64(w) / 2,
		centerY:    float64(h) / 2,
		levelWidth: int(levelWidth),
		rgba:       rgba,
		ctx:        ctx,
	}
}

type curve struct {
	level      int
	color      int
	start, end float64
}

//func (c *curve) ColorModel() color.Model {
//	return color.AlphaModel
//}
//
//func (c *curve) Bounds() image.Rectangle {
//	r := (c.level + 1) * c.levelWidth
//	return image.Rect(c.x-r, c.y-r, c.x+r, c.y+r)
//}
//
//func (c *curve) At(x, y int) color.Color {
//	rIn := float64(c.level * c.levelWidth)
//	rOut := float64((c.level + 1) * c.levelWidth)
//	xx, yy := float64(x-c.x)+0.5, float64(y-c.y)+0.5
//	if xxyy := xx*xx + yy*yy; xxyy > rOut*rOut || xxyy < rIn*rIn {
//		return color.Alpha{0}
//	}
//
//	angle := math.Atan2(yy, xx)
//	if angle < 0 {
//		angle += 2 * math.Pi
//	}
//	angle /= 2 * math.Pi
//
//	if c.end == 0 {
//		c.end = 1
//	}
//	if angle < c.start || angle > c.end {
//		return color.Alpha{0}
//	}
//
//	return color.Alpha{255}
//}

func (i img) drawCurve(c curve) {
	red := byte((c.color >> 16))
	green := byte((c.color >> 8))
	blue := byte(c.color)

	startAngle := c.start * 2 * math.Pi
	angle := (c.end - c.start) * 2 * math.Pi
	radius := float64(c.level * i.levelWidth)
	i.ctx.SetStrokeColor(color.RGBA{red, green, blue, 0xFF})
	i.ctx.SetLineWidth(float64(i.levelWidth))
	i.ctx.SetLineCap(draw2d.SquareCap)
	i.ctx.MoveTo(
		i.centerX+math.Cos(startAngle)*radius,
		i.centerY+math.Sin(startAngle)*radius,
	)
	i.ctx.ArcTo(i.centerX, i.centerY, radius, radius, startAngle, angle)
	i.ctx.Stroke()
}

func (i img) cat(i2 img) {
	draw.Draw(i.rgba, i.rgba.Bounds(), i2.rgba, image.ZP, draw.Over)
}

func (i img) copyBlank() img {
	i2 := i
	i2.rgba = image.NewRGBA(image.Rect(0, 0, i.w, i.h))
	i2.ctx = draw2dimg.NewGraphicContext(i2.rgba)
	return i2
}

func (i img) save() error {
	back := image.NewRGBA(image.Rect(0, 0, i.w, i.h))
	draw.Draw(back, back.Bounds(), &image.Uniform{color.White}, image.ZP, draw.Src)
	draw.Draw(back, back.Bounds(), i.rgba, image.ZP, draw.Over)

	f, err := os.Create(i.filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, back)
}
