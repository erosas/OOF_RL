// tools/genicon generates icon.ico in the repo root.
// Run: go run ./tools/genicon
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

// Rocket League palette
var (
	colBG     = color.RGBA{10, 18, 30, 255}   // deep navy
	colBlue   = color.RGBA{58, 142, 255, 255}  // RL blue
	colOrange = color.RGBA{255, 130, 30, 255}  // RL orange
	colWhite  = color.RGBA{230, 240, 255, 255} // slightly blue-white
)

// 5×9 pixel art glyphs.  1 = lit, 0 = off.
var glyphs = map[rune][9][5]uint8{
	'O': {
		{0, 1, 1, 1, 0},
		{1, 1, 0, 1, 1},
		{1, 1, 0, 1, 1},
		{1, 1, 0, 1, 1},
		{1, 1, 0, 1, 1},
		{1, 1, 0, 1, 1},
		{1, 1, 0, 1, 1},
		{1, 1, 0, 1, 1},
		{0, 1, 1, 1, 0},
	},
	'F': {
		{1, 1, 1, 1, 1},
		{1, 1, 0, 0, 0},
		{1, 1, 0, 0, 0},
		{1, 1, 0, 0, 0},
		{1, 1, 1, 1, 0},
		{1, 1, 0, 0, 0},
		{1, 1, 0, 0, 0},
		{1, 1, 0, 0, 0},
		{1, 1, 0, 0, 0},
	},
}

func main() {
	sizes := []int{256, 48, 32, 16}
	imgs := make([]*image.NRGBA, len(sizes))
	for i, sz := range sizes {
		imgs[i] = drawIcon(sz)
	}

	ico := encodeICO(imgs)
	if err := os.WriteFile("icon.ico", ico, 0644); err != nil {
		log.Fatalf("write icon.ico: %v", err)
	}
	log.Println("icon.ico written")
}

func drawIcon(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	cx := float64(size) / 2
	cy := float64(size) / 2
	radius := cx - 1

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			px := float64(x) + 0.5
			py := float64(y) + 0.5
			dx := px - cx
			dy := py - cy
			dist := math.Sqrt(dx*dx + dy*dy)

			if dist > radius {
				// outside circle → transparent
				img.SetNRGBA(x, y, color.NRGBA{0, 0, 0, 0})
				continue
			}

			// radial background gradient: dark center → slightly lighter edge
			t := dist / radius
			r := lerpU8(colBG.R, 22, t)
			g := lerpU8(colBG.G, 35, t)
			b := lerpU8(colBG.B, 58, t)
			img.SetNRGBA(x, y, color.NRGBA{r, g, b, 255})

			// subtle inner glow ring (RL blue at ~65% radius)
			glow := math.Abs(t - 0.65)
			if glow < 0.15 {
				strength := uint8((1 - glow/0.15) * 35)
				c := img.NRGBAAt(x, y)
				img.SetNRGBA(x, y, addGlow(c, colBlue, strength))
			}

			// outer rim — thin blue ring near edge
			if dist > radius*0.92 {
				rimT := (dist - radius*0.92) / (radius * 0.08)
				strength := uint8(rimT * 180)
				c := img.NRGBAAt(x, y)
				img.SetNRGBA(x, y, blendOver(c, color.NRGBA{colBlue.R, colBlue.G, colBlue.B, strength}))
			}
		}
	}

	// draw "OOF" glyph letters
	text := "OOF"
	glyphW := 5
	glyphH := 9
	gap := 1
	scale := max(1, size/36)
	totalW := len(text)*glyphW*scale + (len(text)-1)*gap*scale
	startX := (size - totalW) / 2
	// vertically centred, shifted up a little to leave room for the bar
	startY := size/2 - (glyphH*scale)/2 - size/16

	curX := startX
	for i, ch := range text {
		glyph, ok := glyphs[ch]
		if !ok {
			curX += (glyphW + gap) * scale
			continue
		}
		// colour scheme: first two letters blue, 'F' orange
		var lit color.NRGBA
		if i < 2 {
			lit = color.NRGBA{colBlue.R, colBlue.G, colBlue.B, 255}
		} else {
			lit = color.NRGBA{colOrange.R, colOrange.G, colOrange.B, 255}
		}
		for row := 0; row < glyphH; row++ {
			for col := 0; col < glyphW; col++ {
				if glyph[row][col] == 0 {
					continue
				}
				for sy := 0; sy < scale; sy++ {
					for sx := 0; sx < scale; sx++ {
						px := curX + col*scale + sx
						py := startY + row*scale + sy
						if px >= 0 && px < size && py >= 0 && py < size {
							// only draw if inside circle
							ddx := float64(px)+0.5 - cx
							ddy := float64(py)+0.5 - cy
							if math.Sqrt(ddx*ddx+ddy*ddy) <= radius {
								img.SetNRGBA(px, py, lit)
							}
						}
					}
				}
			}
		}
		curX += (glyphW + gap) * scale
	}

	// orange accent bar below letters
	barY := startY + glyphH*scale + scale*2
	barH := max(1, scale)
	barMargin := size / 5
	for y := barY; y < barY+barH && y < size; y++ {
		for x := barMargin; x < size-barMargin; x++ {
			ddx := float64(x)+0.5 - cx
			ddy := float64(y)+0.5 - cy
			if math.Sqrt(ddx*ddx+ddy*ddy) <= radius {
				img.SetNRGBA(x, y, color.NRGBA{colOrange.R, colOrange.G, colOrange.B, 220})
			}
		}
	}

	return img
}

func lerpU8(a, b uint8, t float64) uint8 {
	return uint8(float64(a) + (float64(b)-float64(a))*t)
}

func addGlow(base color.NRGBA, glow color.RGBA, strength uint8) color.NRGBA {
	add := func(b, g uint8) uint8 {
		v := int(b) + int(g)*int(strength)/255
		if v > 255 {
			return 255
		}
		return uint8(v)
	}
	return color.NRGBA{add(base.R, glow.R), add(base.G, glow.G), add(base.B, glow.B), base.A}
}

func blendOver(dst color.NRGBA, src color.NRGBA) color.NRGBA {
	a := float64(src.A) / 255
	blend := func(d, s uint8) uint8 {
		return uint8(float64(d)*(1-a) + float64(s)*a)
	}
	return color.NRGBA{blend(dst.R, src.R), blend(dst.G, src.G), blend(dst.B, src.B), dst.A}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// encodeICO produces a .ico file containing one PNG-encoded image per entry.
func encodeICO(imgs []*image.NRGBA) []byte {
	var pngBufs [][]byte
	for _, img := range imgs {
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			log.Fatalf("png encode: %v", err)
		}
		pngBufs = append(pngBufs, buf.Bytes())
	}

	// ICO header: 6 bytes
	// Dir entries: 16 bytes each
	// Image data follows
	headerSize := 6
	dirSize := 16 * len(imgs)
	offset := headerSize + dirSize

	var out bytes.Buffer
	// ICONDIR header
	binary.Write(&out, binary.LittleEndian, uint16(0))         // reserved
	binary.Write(&out, binary.LittleEndian, uint16(1))         // type: icon
	binary.Write(&out, binary.LittleEndian, uint16(len(imgs))) // count

	// directory entries
	for i, img := range imgs {
		sz := img.Bounds().Dx()
		w := uint8(sz)
		h := uint8(sz)
		if sz >= 256 {
			w = 0 // 0 means 256 in ICO format
			h = 0
		}
		binary.Write(&out, binary.LittleEndian, w)          // width
		binary.Write(&out, binary.LittleEndian, h)          // height
		binary.Write(&out, binary.LittleEndian, uint8(0))   // color count
		binary.Write(&out, binary.LittleEndian, uint8(0))   // reserved
		binary.Write(&out, binary.LittleEndian, uint16(1))  // color planes
		binary.Write(&out, binary.LittleEndian, uint16(32)) // bits per pixel
		binary.Write(&out, binary.LittleEndian, uint32(len(pngBufs[i]))) // size
		binary.Write(&out, binary.LittleEndian, uint32(offset))          // offset
		offset += len(pngBufs[i])
	}

	// image data
	for _, buf := range pngBufs {
		out.Write(buf)
	}

	return out.Bytes()
}