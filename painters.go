package captcha

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log"
	"math"
	"math/rand"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

// CharImage 是字符串/算术/中文验证码的画布：一张 NRGBA 图片，
// 提供干扰线、噪点文字与主体文字的绘制能力。
type CharImage struct {
	bgColor color.Color
	width   int
	height  int
	nrgba   *image.NRGBA
}

// newCharImage 创建画布并用 bgColor 平铺背景。
func newCharImage(w int, h int, bgColor color.RGBA) *CharImage {
	d := CharImage{width: w, height: h}
	m := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(m, m.Bounds(), &image.Uniform{bgColor}, image.ZP, draw.Src)
	d.nrgba = m
	return &d
}

// strokeHollow 绘制一条横贯画面的粗空心正弦线。
// 起点/终点与振幅随机，方向（上弯/下弯）由 multiple 的符号控制。
func (item *CharImage) strokeHollow() *CharImage {
	first := item.width / 20
	end := first * 19

	lineColor := RandomLightTone()

	x1 := float64(rand.Intn(first))
	x2 := float64(rand.Intn(first) + end)

	// 频率倍数，1/10 概率为负 → 曲线上下翻转
	multiple := float64(rand.Intn(5)+3) / float64(5)
	if int(multiple*10)%3 == 0 {
		multiple = multiple * -1.0
	}

	w := item.height / 20

	for ; x1 < x2; x1++ {
		// 沿正弦曲线走，纵坐标限制在画面中部
		y := math.Sin(x1*math.Pi*multiple/float64(item.width)) * float64(item.height/3)

		if multiple < 0 {
			y = y + float64(item.height/2)
		}
		item.nrgba.Set(int(x1), int(y), lineColor)

		// 沿 y 方向加粗线宽
		for i := 0; i <= w; i++ {
			item.nrgba.Set(int(x1), int(y)+i, lineColor)
		}
	}

	return item
}

// strokeSineWave 绘制若干条小振幅正弦波浪线（随机相位、周期与颜色）。
func (item *CharImage) strokeSineWave() *CharImage {
	var py float64

	a := rand.Intn(item.height / 2)                                    // 振幅
	b := randFloatBetween(int64(-item.height/4), int64(item.height/4)) // Y 轴偏移
	f := randFloatBetween(int64(-item.height/4), int64(item.height/4)) // X 轴相位偏移

	// 周期 t：随画布宽高比例取随机值
	var t float64
	if item.height > item.width/2 {
		t = randFloatBetween(int64(item.width/2), int64(item.height))
	} else if item.height == item.width/2 {
		t = float64(item.height)
	} else {
		t = randFloatBetween(int64(item.height), int64(item.width/2))
	}
	w := float64((2 * math.Pi) / t)

	// 曲线的横坐标范围
	px1 := 0
	px2 := int(randFloatBetween(int64(float64(item.width)*0.8), int64(item.width)))

	c := RandomDarkColor()

	for px := px1; px < px2; px++ {
		if w != 0 {
			py = float64(a)*math.Sin(w*float64(px)+f) + b + (float64(item.width) / float64(5))
			i := item.height / 5
			for i > 0 {
				item.nrgba.Set(px+i, int(py), c)
				i--
			}
		}
	}

	return item
}

// strokeSlants 绘制 num 条随机颜色的斜细线，起止点分别取自画面左右两侧。
func (item *CharImage) strokeSlants(num int) *CharImage {
	first := item.width / 10
	end := first * 9

	y := item.height / 3

	for i := 0; i < num; i++ {
		point1 := point{X: rand.Intn(first), Y: rand.Intn(y)}
		point2 := point{X: rand.Intn(first) + end, Y: rand.Intn(y)}

		// 奇偶交替决定起止点在画面上/下半区，形成交叉效果
		if i%2 == 0 {
			point1.Y = rand.Intn(y) + y*2
			point2.Y = rand.Intn(y)
		} else {
			point1.Y = rand.Intn(y) + y*(i%2)
			point2.Y = rand.Intn(y) + y*2
		}

		item.drawStraight(point1, point2, RandomDarkColor())
	}
	return item
}

// drawStraight 用 Bresenham 算法在两点之间画一条加粗直线。
func (item *CharImage) drawStraight(point1 point, point2 point, lineColor color.RGBA) {
	dx := math.Abs(float64(point1.X - point2.X))
	dy := math.Abs(float64(point2.Y - point1.Y))
	sx, sy := 1, 1
	if point1.X >= point2.X {
		sx = -1
	}
	if point1.Y >= point2.Y {
		sy = -1
	}
	err := dx - dy
	for {
		// 横向加粗 5 像素，形成细线粗细
		item.nrgba.Set(point1.X, point1.Y, lineColor)
		item.nrgba.Set(point1.X+1, point1.Y, lineColor)
		item.nrgba.Set(point1.X-1, point1.Y, lineColor)
		item.nrgba.Set(point1.X+2, point1.Y, lineColor)
		item.nrgba.Set(point1.X-2, point1.Y, lineColor)
		if point1.X == point2.X && point1.Y == point2.Y {
			return
		}
		e2 := err * 2
		if e2 > -dy {
			err -= dy
			point1.X += sx
		}
		if e2 < dx {
			err += dx
			point1.Y += sy
		}
	}
}

// paintNoiseText 把一串噪声字符随机撒在画布上作背景干扰，
// 每个字符独立随机位置、随机字号（约高度一半）与浅色颜色。
func (item *CharImage) paintNoiseText(noiseText string, fonts []*truetype.Font) error {
	c := freetype.NewContext()
	c.SetDPI(renderDPI)

	c.SetClip(item.nrgba.Bounds())
	c.SetDst(item.nrgba)
	c.SetHinting(font.HintingFull)
	rawFontSize := float64(item.height) / (1 + float64(rand.Intn(7))/float64(10))

	for _, char := range noiseText {
		rw := rand.Intn(item.width)
		rh := rand.Intn(item.height)
		fontSize := rawFontSize/2 + float64(rand.Intn(5))
		c.SetSrc(image.NewUniform(RandomLightTone()))
		c.SetFontSize(fontSize)
		c.SetFont(pickFont(fonts))
		pt := freetype.Pt(rw, rh)
		if _, err := c.DrawString(string(char), pt); err != nil {
			log.Println(err)
		}
	}
	return nil
}

// paintText 把验证码主体文字逐字符绘制到画布：
// 每个字符独立随机字号（高度的 7/16 ~ 13/16）、深色随机颜色与纵向抖动。
// 文本为空时返回错误（没有可绘制的内容）。
func (item *CharImage) paintText(text string, fonts []*truetype.Font) error {
	c := freetype.NewContext()
	c.SetDPI(renderDPI)
	c.SetClip(item.nrgba.Bounds())
	c.SetDst(item.nrgba)
	c.SetHinting(font.HintingFull)

	if len(text) == 0 {
		return errors.New("text must not be empty, there is nothing to draw")
	}

	fontWidth := item.width / len(text)

	for i, s := range text {
		fontSize := item.height * (rand.Intn(7) + 7) / 16
		if fontSize < 1 {
			fontSize = 1 // 极端小图保护，避免后续整除零 panic
		}
		c.SetSrc(image.NewUniform(RandomDarkColor()))
		c.SetFontSize(float64(fontSize))
		c.SetFont(pickFont(fonts))
		x := fontWidth*i + fontWidth/fontSize
		y := item.height/2 + fontSize/2 - rand.Intn(item.height/16*3)
		pt := freetype.Pt(x, y)
		if _, err := c.DrawString(string(s), pt); err != nil {
			return err
		}
	}
	return nil
}

// encodePNGBytes 把画布编码为 PNG 字节流。
func (item *CharImage) encodePNGBytes() []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, item.nrgba); err != nil {
		panic(err.Error())
	}
	return buf.Bytes()
}

// SaveTo 实现 Artifact：把 PNG 数据写入 w。
func (item *CharImage) SaveTo(w io.Writer) (int64, error) {
	n, err := w.Write(item.encodePNGBytes())
	return int64(n), err
}

// Base64String 实现 Artifact：返回 data:image/png;base64,... 字符串。
func (item *CharImage) Base64String() string {
	return fmt.Sprintf("data:%s;base64,%s", MimeImagePng, base64.StdEncoding.EncodeToString(item.encodePNGBytes()))
}

type point struct {
	X int
	Y int
}
