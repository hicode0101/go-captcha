package captcha

import (
	"image/color"
	"strings"

	"github.com/golang/freetype/truetype"
)

// StringEngine 字符串验证码引擎。
// 从候选字符集（Charset）中随机取 TextLength 个字符渲染成 PNG，
// 支持干扰线、背景噪点文字与自定义背景色。
type StringEngine struct {
	// ImageHeight 画布高度（像素）。
	ImageHeight int
	// ImageWidth 画布宽度（像素）。
	ImageWidth int
	// NoiseCount 背景噪点文字数量；0 表示不绘制。
	NoiseCount int
	// LineOptions 干扰线选项，取 LineOptionHollow / LineOptionSlime /
	// LineOptionSine 的按位或组合；0 表示不绘制干扰线。
	LineOptions int
	// TextLength 验证码字符个数。
	TextLength int
	// Charset 候选字符集（按 rune 采样）。
	Charset string
	// BgColor 背景色；nil 表示每次随机浅色背景。
	BgColor *color.RGBA

	// fontBox 字体仓库（nil 时使用 BuiltinFontBox）。
	fontBox FontBox
	// FontNames 参与渲染的字体文件名列表；空列表表示使用全部内嵌字体。
	FontNames   []string
	loadedFonts []*truetype.Font
}

// NewStringEngine 创建字符串验证码引擎。
//
// 参数：
//
//	imageHeight  画布高度（像素），常用 60
//	imageWidth   画布宽度（像素），常用 240
//	noiseCount   背景噪点文字数量，常用 0~5
//	lineOptions  干扰线位组合（LineOptionHollow|LineOptionSlime|...），0 为不画线
//	textLength   字符个数，常用 4~6
//	charset      候选字符集，如 CharsetDigits + CharsetLetters
//	bgColor      背景色指针，nil 表示随机浅色背景
//	fontBox      字体仓库，nil 表示使用内嵌字体仓库
//	fontNames    字体文件名列表，nil/空 表示使用全部内嵌字体
func NewStringEngine(imageHeight int, imageWidth int, noiseCount int, lineOptions int, textLength int, charset string, bgColor *color.RGBA, fontBox FontBox, fontNames []string) *StringEngine {
	if fontBox == nil {
		fontBox = BuiltinFontBox
	}

	loaded := loadEngineFonts(fontBox, fontNames)

	return &StringEngine{ImageHeight: imageHeight, ImageWidth: imageWidth, NoiseCount: noiseCount, LineOptions: lineOptions, TextLength: textLength, Charset: charset, BgColor: bgColor, fontBox: fontBox, loadedFonts: loaded, FontNames: fontNames}
}

// loadEngineFonts 按名称加载字体；列表为空时回退为全部内嵌字体。
func loadEngineFonts(box FontBox, names []string) []*truetype.Font {
	tfs := []*truetype.Font{}
	for _, name := range names {
		tf := box.OpenFont("fonts/" + name)
		tfs = append(tfs, tf)
	}
	if len(tfs) == 0 {
		tfs = allBuiltinFonts
	}
	return tfs
}

// LoadFonts 重新按 FontNames 加载字体。
// 适用于先以 struct 字面量构造引擎、再手动加载字体的用法；链式调用。
func (d *StringEngine) LoadFonts() *StringEngine {
	if d.fontBox == nil {
		d.fontBox = BuiltinFontBox
	}
	d.loadedFonts = loadEngineFonts(d.fontBox, d.FontNames)
	return d
}

// NewChallenge 生成字符串验证码挑战：question 与 answer 均为随机字符集文本。
func (d *StringEngine) NewChallenge() (id, question, answer string) {
	id = NewSessionID()
	question = RandRunes(d.TextLength, d.Charset)
	return id, question, question
}

// Render 把题目文本渲染为字符串验证码图片。
//
// 绘制顺序：干扰线（可选）→ 背景噪点文字（可选）→ 主体文字。
func (d *StringEngine) Render(content string) (item Artifact, err error) {
	img := newCharImage(d.ImageWidth, d.ImageHeight, d.backgroundColor())

	// 粗空心正弦线
	if d.LineOptions&LineOptionHollow == LineOptionHollow {
		img.strokeHollow()
	}

	// 斜细线
	if d.LineOptions&LineOptionSlime == LineOptionSlime {
		img.strokeSlants(3)
	}

	// 小波浪线
	if d.LineOptions&LineOptionSine == LineOptionSine {
		img.strokeSineWave()
	}

	// 背景噪点文字
	if d.NoiseCount > 0 {
		source := CharsetDigits + CharsetLetters + ",.[]<>"
		noise := RandRunes(d.NoiseCount, strings.Repeat(source, d.NoiseCount))
		err = img.paintNoiseText(noise, d.loadedFonts)
		if err != nil {
			return
		}
	}

	// 主体文字
	err = img.paintText(content, d.loadedFonts)
	if err != nil {
		return
	}

	return img, nil
}

// backgroundColor 返回背景色：未配置时随机浅色。
func (d *StringEngine) backgroundColor() color.RGBA {
	if d.BgColor != nil {
		return *d.BgColor
	}
	return RandomLightTone()
}

var _ Engine = (*StringEngine)(nil)
