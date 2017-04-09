package pixel

import (
	"fmt"
	"image/color"
)

// Batch is a Target that allows for efficient drawing of many objects with the same Picture.
//
// To put an object into a Batch, just draw it onto it:
//   object.Draw(batch)
type Batch struct {
	cont Drawer

	mat Matrix
	col RGBA
}

var _ BasicTarget = (*Batch)(nil)

// NewBatch creates an empty Batch with the specified Picture and container.
//
// The container is where objects get accumulated. Batch will support precisely those Triangles
// properties, that the supplied container supports. If you retain access to the container and
// change it, call Dirty to notify Batch about the change.
//
// Note, that if the container does not support TrianglesColor, color masking will not work.
func NewBatch(container Triangles, pic Picture) *Batch {
	b := &Batch{cont: Drawer{Triangles: container, Picture: pic}}
	b.SetMatrix(IM)
	b.SetColorMask(RGBA{1, 1, 1, 1})
	return b
}

// Dirty notifies Batch about an external modification of it's container. If you retain access to
// the Batch's container and change it, call Dirty to notify Batch about the change.
//
//   container := &pixel.TrianglesData{}
//   batch := pixel.NewBatch(container, nil)
//   container.SetLen(10) // container changed from outside of Batch
//   batch.Dirty()        // notify Batch about the change
func (b *Batch) Dirty() {
	b.cont.Dirty()
}

// Clear removes all objects from the Batch.
func (b *Batch) Clear() {
	b.cont.Triangles.SetLen(0)
	b.cont.Dirty()
}

// Draw draws all objects that are currently in the Batch onto another Target.
func (b *Batch) Draw(t Target) {
	b.cont.Draw(t)
}

// SetMatrix sets a Matrix that every point will be projected by.
func (b *Batch) SetMatrix(m Matrix) {
	b.mat = m
}

// SetColorMask sets a mask color used in the following draws onto the Batch.
func (b *Batch) SetColorMask(c color.Color) {
	if c == nil {
		b.col = RGBA{1, 1, 1, 1}
		return
	}
	b.col = ToRGBA(c)
}

// MakeTriangles returns a specialized copy of the provided Triangles that draws onto this Batch.
func (b *Batch) MakeTriangles(t Triangles) TargetTriangles {
	bt := &batchTriangles{
		tri: t.Copy(),
		tmp: MakeTrianglesData(t.Len()),
		dst: b,
	}
	return bt
}

// MakePicture returns a specialized copy of the provided Picture that draws onto this Batch.
func (b *Batch) MakePicture(p Picture) TargetPicture {
	if p != b.cont.Picture {
		panic(fmt.Errorf("(%T).MakePicture: Picture is not the Batch's Picture", b))
	}
	bp := &batchPicture{
		pic: p,
		dst: b,
	}
	return bp
}

type batchTriangles struct {
	tri Triangles
	tmp *TrianglesData

	dst *Batch
}

func (bt *batchTriangles) Len() int {
	return bt.tri.Len()
}

func (bt *batchTriangles) SetLen(len int) {
	bt.tri.SetLen(len)
	bt.tmp.SetLen(len)
}

func (bt *batchTriangles) Slice(i, j int) Triangles {
	return &batchTriangles{
		tri: bt.tri.Slice(i, j),
		tmp: bt.tmp.Slice(i, j).(*TrianglesData),
		dst: bt.dst,
	}
}

func (bt *batchTriangles) Update(t Triangles) {
	bt.tri.Update(t)
}

func (bt *batchTriangles) Copy() Triangles {
	return &batchTriangles{
		tri: bt.tri.Copy(),
		tmp: bt.tmp.Copy().(*TrianglesData),
		dst: bt.dst,
	}
}

func (bt *batchTriangles) draw(bp *batchPicture) {
	bt.tmp.Update(bt.tri)

	for i := range *bt.tmp {
		(*bt.tmp)[i].Position = bt.dst.mat.Project((*bt.tmp)[i].Position)
		(*bt.tmp)[i].Color = bt.dst.col.Mul((*bt.tmp)[i].Color)
	}

	cont := bt.dst.cont.Triangles
	cont.SetLen(cont.Len() + bt.tri.Len())
	added := cont.Slice(cont.Len()-bt.tri.Len(), cont.Len())
	added.Update(bt.tri)
	added.Update(bt.tmp)
	bt.dst.cont.Dirty()
}

func (bt *batchTriangles) Draw() {
	bt.draw(nil)
}

type batchPicture struct {
	pic Picture
	dst *Batch
}

func (bp *batchPicture) Bounds() Rect {
	return bp.pic.Bounds()
}

func (bp *batchPicture) Draw(t TargetTriangles) {
	bt := t.(*batchTriangles)
	if bp.dst != bt.dst {
		panic(fmt.Errorf("(%T).Draw: TargetTriangles generated by different Batch", bp))
	}
	bt.draw(bp)
}
