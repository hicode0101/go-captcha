package captcha

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/golang/freetype/truetype"
)

// MathEngine 算术验证码引擎。
// 随机生成一道加/减/乘口算题，题目绘制到图片上，
// 答案为计算结果（因此 question 与 answer 不同）。
type MathEngine struct {
	// ImageHeight 画布高度（像素）。
	ImageHeight int
	// ImageWidth 画布宽度（像素）。
	ImageWidth int
	// NoiseCount 背景噪点文字数量；0 表示不绘制。
	NoiseCount int
	// LineOptions 干扰线选项，按位或组合；0 表示不绘制。
	LineOptions int
	// BgColor 背景色；nil 表示每次随机浅色背景。
	BgColor *color.RGBA

	// fontBox 字体仓库（nil 时使用 BuiltinFontBox）。
	fontBox FontBox
	// FontNames 参与渲染的字体文件名列表；空列表表示使用全部内嵌字体。
	FontNames   []string
	loadedFonts []*truetype.Font
}

// NewMathEngine 创建算术验证码引擎。
//
// 参数含义与 StringEngine 一致（无题目长度/字符集参数）：
//
//	imageHeight 画布高度，常用 60
//	imageWidth  画布宽度，常用 240
//	noiseCount  背景噪点文字数量
//	lineOptions 干扰线位组合
//	bgColor     背景色指针，nil 表示随机浅色背景
//	fontBox     字体仓库，nil 表示内嵌字体
//	fontNames   字体文件名列表，空表示全部内嵌字体
func NewMathEngine(imageHeight int, imageWidth int, noiseCount int, lineOptions int, bgColor *color.RGBA, fontBox FontBox, fontNames []string) *MathEngine {
	if fontBox == nil {
		fontBox = BuiltinFontBox
	}
	loaded := loadEngineFonts(fontBox, fontNames)
	return &MathEngine{ImageHeight: imageHeight, ImageWidth: imageWidth, NoiseCount: noiseCount, LineOptions: lineOptions, loadedFonts: loaded, BgColor: bgColor, FontNames: fontNames, fontBox: fontBox}
}

// LoadFonts 重新按 FontNames 加载字体；链式调用。
func (d *MathEngine) LoadFonts() *MathEngine {
	if d.fontBox == nil {
		d.fontBox = BuiltinFontBox
	}
	d.loadedFonts = loadEngineFonts(d.fontBox, d.FontNames)
	return d
}

// NewChallenge 生成一道算术题：
//
//	加法：a+b，两数均取自 [0,20)
//	乘法：a×b，两数均取自 [0,10)
//	减法：a-b，a 取自 [0,100) 保证结果为正
//
// question 形如 "12+5=?"，answer 为计算结果字符串。
func (d *MathEngine) NewChallenge() (id, question, answer string) {
	id = NewSessionID()
	operators := []string{"+", "-", "x"}
	var mathResult int32
	switch operators[rand.Int31n(3)] {
	case "+":
		a := rand.Int31n(20)
		b := rand.Int31n(20)
		question = fmt.Sprintf("%d+%d=?", a, b)
		mathResult = a + b
	case "x":
		a := rand.Int31n(10)
		b := rand.Int31n(10)
		question = fmt.Sprintf("%dx%d=?", a, b)
		mathResult = a * b
	default:
		a := rand.Int31n(80) + rand.Int31n(20)
		b := rand.Int31n(80)

		question = fmt.Sprintf("%d-%d=?", a, b)
		mathResult = a - b
	}
	answer = fmt.Sprintf("%d", mathResult)
	return
}

// Render 把算术题渲染为验证码图片。
//
// 绘制顺序：波浪线（可选）→ 背景数字噪点（可选）→ 斜细线（可选）
// → 小波浪线（可选）→ 题目文字。
func (d *MathEngine) Render(question string) (item Artifact, err error) {
	img := newCharImage(d.ImageWidth, d.ImageHeight, d.backgroundColor())

	// 粗空心波浪线
	if d.LineOptions&LineOptionHollow == LineOptionHollow {
		img.strokeHollow()
	}

	// 背景数字噪点
	if d.NoiseCount > 0 {
		noise := RandRunes(d.NoiseCount, repeatString(CharsetDigits, d.NoiseCount))
		err = img.paintNoiseText(noise, d.loadedFonts)
		if err != nil {
			return
		}
	}

	// 斜细线
	if d.LineOptions&LineOptionSlime == LineOptionSlime {
		img.strokeSlants(3)
	}

	// 小波浪线
	if d.LineOptions&LineOptionSine == LineOptionSine {
		img.strokeSineWave()
	}

	// 题目文字
	err = img.paintText(question, d.loadedFonts)
	if err != nil {
		return
	}
	return img, nil
}

// backgroundColor 返回背景色：未配置时随机浅色。
func (d *MathEngine) backgroundColor() color.RGBA {
	if d.BgColor != nil {
		return *d.BgColor
	}
	return RandomLightTone()
}

// repeatString 把 s 重复 n 次（n<=0 返回空串）。
func repeatString(s string, n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}

var _ Engine = (*MathEngine)(nil)
