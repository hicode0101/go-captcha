package captcha

import (
	"image/color"
	"math/rand"
	"strings"

	"github.com/golang/freetype/truetype"
)

// ChineseEngine 中文验证码引擎。
// 候选字集 Charset 支持两种写法：
//
//  1. 纯字符集（无逗号）：直接从字符集中按 rune 随机取字；
//  2. 逗号分隔的候选词组：每次从词组列表中随机抽一个字组词出题，
//     更贴近真实语序、便于朗读。
type ChineseEngine struct {
	// ImageHeight 画布高度（像素）。
	ImageHeight int
	// ImageWidth 画布宽度（像素）。
	ImageWidth int
	// NoiseCount 背景噪点文字数量；0 表示不绘制。
	NoiseCount int
	// LineOptions 干扰线选项，按位或组合；0 表示不绘制。
	LineOptions int
	// TextLength 验证码汉字个数。
	TextLength int
	// Charset 候选字集或逗号分隔的候选词组。
	Charset string
	// BgColor 背景色；nil 表示每次随机浅色背景。
	BgColor *color.RGBA

	// fontBox 字体仓库（nil 时使用 BuiltinFontBox）。
	fontBox FontBox
	// FontNames 参与渲染的字体文件名列表；空列表表示使用全部内嵌字体。
	FontNames   []string
	loadedFonts []*truetype.Font
}

// NewChineseEngine 创建中文验证码引擎。
//
// 参数含义与 StringEngine 一致；渲染中文必须使用中文字体，
// fontNames 建议显式传入 []string{"chinese.ttf"} 或 []string{"pinyin-cn.ttc"}，
// 留空则使用全部内嵌字体（其中包含中文字体，亦可正常渲染）。
func NewChineseEngine(imageHeight int, imageWidth int, noiseCount int, lineOptions int, textLength int, charset string, bgColor *color.RGBA, fontBox FontBox, fontNames []string) *ChineseEngine {
	if fontBox == nil {
		fontBox = BuiltinFontBox
	}
	loaded := loadEngineFonts(fontBox, fontNames)
	return &ChineseEngine{ImageHeight: imageHeight, ImageWidth: imageWidth, NoiseCount: noiseCount, LineOptions: lineOptions, TextLength: textLength, Charset: charset, BgColor: bgColor, fontBox: fontBox, loadedFonts: loaded, FontNames: fontNames}
}

// LoadFonts 重新按 FontNames 加载字体；链式调用。
func (d *ChineseEngine) LoadFonts() *ChineseEngine {
	if d.fontBox == nil {
		d.fontBox = BuiltinFontBox
	}
	d.loadedFonts = loadEngineFonts(d.fontBox, d.FontNames)
	return d
}

// NewChallenge 生成中文验证码挑战。
//
// Charset 无逗号时按字符集随机取 TextLength 个字；
// 逗号分隔时按词组抽取（词组数 <= TextLength 时退化为随机字符集）。
func (d *ChineseEngine) NewChallenge() (id, question, answer string) {
	id = NewSessionID()

	ss := strings.Split(d.Charset, ",")
	length := len(ss)
	// 无逗号：整体视为字符集
	if length == 1 {
		c := RandRunes(d.TextLength, ss[0])
		return id, c, c
	}
	// 词组数不足以支撑抽词：退化为随机数字字母
	if length <= d.TextLength {
		c := RandRunes(d.TextLength, CharsetDigits+CharsetLetters)
		return id, c, c
	}

	// 逐字随机抽取词组
	res := make([]string, d.TextLength)
	for k := range res {
		res[k] = ss[rand.Intn(length)]
	}

	question = strings.Join(res, "")
	return id, question, question
}

// Render 把中文题目渲染为验证码图片（绘制流程与 StringEngine 相同）。
func (d *ChineseEngine) Render(content string) (item Artifact, err error) {
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
func (d *ChineseEngine) backgroundColor() color.RGBA {
	if d.BgColor != nil {
		return *d.BgColor
	}
	return RandomLightTone()
}

var _ Engine = (*ChineseEngine)(nil)
