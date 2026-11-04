package captcha

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"math/rand"
)

// DotMatrixImage 是数字验证码的画布：基于调色板的点阵图片。
// 数字由圆形"墨点"拼成，再经过波浪扭曲与圆点噪声填充。
type DotMatrixImage struct {
	width  int
	height int
	*image.Paletted
	dotSize  int     // 单个墨点的直径（像素）
	dotCount int     // 背景噪声圆点数量
	maxSkew  float64 // 单个数字允许的最大倾斜系数
}

// newDotMatrixImage 创建点阵画布，并按 dotCount 构建随机调色板。
func newDotMatrixImage(width int, height int, dotCount int, maxSkew float64) *DotMatrixImage {
	itemDigit := &DotMatrixImage{width: width, height: height, dotCount: dotCount, maxSkew: maxSkew}
	itemDigit.Paletted = image.NewPaletted(image.Rect(0, 0, width, height), buildPalette(dotCount))
	return itemDigit
}

// buildPalette 构建随机调色板：0 号为透明色，1 号为主色，
// 2..dotCount 为主色的随机明度变体（用作背景圆点）。
func buildPalette(dotCount int) color.Palette {
	p := make([]color.Color, dotCount+1)
	// 透明色
	p[0] = color.RGBA{0xFF, 0xFF, 0xFF, 0x00}
	// 主色
	prim := color.RGBA{
		uint8(rand.Intn(129)),
		uint8(rand.Intn(129)),
		uint8(rand.Intn(129)),
		0xFF,
	}

	if dotCount == 0 {
		p[0] = prim
		return p
	}

	p[1] = prim
	// 背景圆点颜色族
	for i := 2; i <= dotCount; i++ {
		p[i] = shiftBrightness(prim, 255)
	}
	return p
}

// fitGlyphs 根据画布尺寸与数字个数推算单个字形的点阵尺寸：
// 先按宽度均分，若换算出的高度超出可用高度则改为按高度约束。
func (m *DotMatrixImage) fitGlyphs(width, height, ncount int) {
	var border int
	if width > height {
		border = height / 4
	} else {
		border = width / 4
	}
	w := float64(width - border*2)
	h := float64(height - border*2)
	// fw 预留了 1 点的字间距
	fw := float64(glyphCellWidth + 1)
	fh := float64(glyphCellHeight)
	nc := float64(ncount)

	nw := w / nc
	nh := nw * fh / fw
	// 高度方向放不下 → 改为按高度适配
	if nh > h {
		nh = h
		nw = fw / fh * nh
	}
	m.dotSize = int(nh / fh)
	if m.dotSize < 1 {
		m.dotSize = 1
	}
	// 实际字宽减去 1 个点的间距
	m.width = int(nw) - m.dotSize
	m.height = int(nh)
}

// fillScanRow 沿水平方向填充一段调色板颜色。
func (m *DotMatrixImage) fillScanRow(fromX, toX, y int, colorIdx uint8) {
	for x := fromX; x <= toX; x++ {
		m.SetColorIndex(x, y, colorIdx)
	}
}

// plotDisc 以 (x, y) 为圆心、radius 为半径绘制实心圆。
func (m *DotMatrixImage) plotDisc(x, y, radius int, colorIdx uint8) {
	f := 1 - radius
	dfx := 1
	dfy := -2 * radius
	xo := 0
	yo := radius

	m.SetColorIndex(x, y+radius, colorIdx)
	m.SetColorIndex(x, y-radius, colorIdx)
	m.fillScanRow(x-radius, x+radius, y, colorIdx)

	for xo < yo {
		if f >= 0 {
			yo--
			dfy += 2
			f += dfy
		}
		xo++
		dfx += 2
		f += dfx
		m.fillScanRow(x-xo, x+xo, y+yo, colorIdx)
		m.fillScanRow(x-xo, x+xo, y-yo, colorIdx)
		m.fillScanRow(x-yo, x+yo, y+xo, colorIdx)
		m.fillScanRow(x-yo, x+yo, y-xo, colorIdx)
	}
}

// scatterDiscs 在画布上随机撒 n 个半径不超过 maxradius 的噪声圆点。
func (m *DotMatrixImage) scatterDiscs(n, maxradius int) {
	maxx := m.Bounds().Max.X
	maxy := m.Bounds().Max.Y
	for i := 0; i < n; i++ {
		colorIdx := uint8(randIntBetween(1, m.dotCount-1))
		r := randIntBetween(1, maxradius)
		m.plotDisc(randIntBetween(r, maxx-r), randIntBetween(r, maxy-r), r, colorIdx)
	}
}

// drawStrikeLine 画一条贯穿画面的波浪删除线。
func (m *DotMatrixImage) drawStrikeLine() {
	maxx := m.Bounds().Max.X
	maxy := m.Bounds().Max.Y
	y := randIntBetween(maxy/3, maxy-maxy/3)
	amplitude := randFloat64Between(5, 20)
	period := randFloat64Between(80, 180)
	dx := 2.0 * math.Pi / period
	for x := 0; x < maxx; x++ {
		xo := amplitude * math.Cos(float64(y)*dx)
		yo := amplitude * math.Sin(float64(x)*dx)
		for yn := 0; yn < m.dotSize; yn++ {
			r := rand.Intn(m.dotSize)
			m.plotDisc(x+int(xo), y+int(yo)+(yn*m.dotSize), r/2, 1)
		}
	}
}

// stampGlyph 把一个数字字模以墨点形式"盖"到 (x, y) 处，
// 每行按 skf 做水平错切，模拟手写倾斜。
func (m *DotMatrixImage) stampGlyph(digit []byte, x, y int) {
	skf := randFloat64Between(-m.maxSkew, m.maxSkew)
	xs := float64(x)
	r := m.dotSize / 2
	y += randIntBetween(-r, r)
	for yo := 0; yo < glyphCellHeight; yo++ {
		for xo := 0; xo < glyphCellWidth; xo++ {
			if digit[yo*glyphCellWidth+xo] != glyphInkBit {
				continue
			}
			m.plotDisc(x+xo*m.dotSize, y+yo*m.dotSize, r, 1)
		}
		xs += skf
		x = int(xs)
	}
}

// waveDistort 对整幅图做正弦波扭曲：新图的每个像素从老图的
// (x+sin, y+cos) 偏移处取样，产生水面波纹效果。
func (m *DotMatrixImage) waveDistort(amplude float64, period float64) {
	w := m.Bounds().Max.X
	h := m.Bounds().Max.Y

	oldm := m.Paletted
	newm := image.NewPaletted(image.Rect(0, 0, w, h), oldm.Palette)

	dx := 2.0 * math.Pi / period
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			xo := amplude * math.Sin(float64(y)*dx)
			yo := amplude * math.Cos(float64(x)*dx)
			newm.SetColorIndex(x, y, oldm.ColorIndexAt(x+int(xo), y+int(yo)))
		}
	}
	m.Paletted = newm
}

// shiftBrightness 在不超过 max 明度的前提下，给颜色 c 施加随机明度偏移，
// 生成同一色系的变体。
func shiftBrightness(c color.RGBA, max uint8) color.RGBA {
	minc := minU8(c.R, c.G, c.B)
	maxc := maxU8(c.R, c.G, c.B)
	if maxc > max {
		return c
	}
	n := rand.Intn(int(max-maxc)) - int(minc)
	return color.RGBA{
		uint8(int(c.R) + n),
		uint8(int(c.G) + n),
		uint8(int(c.B) + n),
		uint8(c.A),
	}
}

func minU8(x, y, z uint8) (m uint8) {
	m = x
	if y < m {
		m = y
	}
	if z < m {
		m = z
	}
	return
}

func maxU8(x, y, z uint8) (m uint8) {
	m = x
	if y > m {
		m = y
	}
	if z > m {
		m = z
	}
	return
}

// encodePNGBytes 把点阵图编码为 PNG 字节流。
func (m *DotMatrixImage) encodePNGBytes() []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, m.Paletted); err != nil {
		panic(err.Error())
	}
	return buf.Bytes()
}

// SaveTo 实现 Artifact：把 PNG 数据写入 w。
func (m *DotMatrixImage) SaveTo(w io.Writer) (int64, error) {
	n, err := w.Write(m.encodePNGBytes())
	return int64(n), err
}

// Base64String 实现 Artifact：返回 data:image/png;base64,... 字符串。
func (m *DotMatrixImage) Base64String() string {
	return fmt.Sprintf("data:%s;base64,%s", MimeImagePng, base64.StdEncoding.EncodeToString(m.encodePNGBytes()))
}
