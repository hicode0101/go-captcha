package captcha

import "errors"

// DigitEngine 数字点阵验证码引擎。
// 数字由圆点字模拼成，经过倾斜、波浪扭曲与背景圆点噪声处理，
// 输出 PNG（DotMatrixImage）。
type DigitEngine struct {
	// ImageHeight 画布高度（像素）。
	ImageHeight int
	// ImageWidth 画布宽度（像素）。
	ImageWidth int
	// DigitCount 验证码包含的数字个数。
	DigitCount int
	// MaxSkew 单个数字的最大倾斜系数（0 表示不倾斜）。
	MaxSkew float64
	// NoiseDots 背景噪声圆点数量。
	NoiseDots int
}

// NewDigitEngine 创建数字验证码引擎。
//
// 参数：
//
//	imageHeight 画布高度（像素），常用 80
//	imageWidth  画布宽度（像素），常用 240
//	digitCount  数字个数，常用 4~6；<=0 时按 4 个出题
//	maxSkew     数字最大倾斜系数，常用 0.7
//	noiseDots   背景圆点数量，常用 80；0 表示纯数字
func NewDigitEngine(imageHeight int, imageWidth int, digitCount int, maxSkew float64, noiseDots int) *DigitEngine {
	return &DigitEngine{ImageHeight: imageHeight, ImageWidth: imageWidth, DigitCount: digitCount, MaxSkew: maxSkew, NoiseDots: noiseDots}
}

// DefaultDigitEngine 是数字引擎的推荐默认配置。
var DefaultDigitEngine = NewDigitEngine(80, 240, 5, 0.7, 80)

// NewChallenge 生成数字验证码挑战：question 与 answer 均为数字字符串。
func (d *DigitEngine) NewChallenge() (id, question, answer string) {
	id = NewSessionID()
	length := d.DigitCount
	if length <= 0 {
		length = 4
	}
	digits := randomDigitBytes(length)
	answer = digitsToString(digits)
	return id, answer, answer
}

// Render 把数字字符串渲染为点阵验证码图片。
//
// 流程：按画布与数字个数推算字模尺寸 → 随机定位 → 逐个盖印数字字形
// → 波浪删除线 → 正弦扭曲 → 背景圆点填充。
// content 必须由 '0'~'9' 组成且非空，否则返回错误。
func (d *DigitEngine) Render(content string) (item Artifact, err error) {
	if len(content) == 0 {
		return nil, errors.New("captcha: content is empty, there is nothing to draw")
	}
	img := newDotMatrixImage(d.ImageWidth, d.ImageHeight, d.NoiseDots, d.MaxSkew)
	digits := stringToDigitBytes(content)

	img.fitGlyphs(d.ImageWidth, d.ImageHeight, len(digits))
	// 数字整体在画面中随机摆放；极端参数下 randIntBetween 会钳制到边界，不会 panic
	maxx := d.ImageWidth - (img.width+img.dotSize)*len(digits) - img.dotSize
	maxy := d.ImageHeight - img.height - img.dotSize*2
	var border int
	if d.ImageWidth > d.ImageHeight {
		border = d.ImageHeight / 5
	} else {
		border = d.ImageWidth / 5
	}
	x := randIntBetween(border, maxx-border)
	y := randIntBetween(border, maxy-border)
	// 逐个盖印数字字形
	for _, n := range digits {
		img.stampGlyph(digitGlyphs[n], x, y)
		x += img.width + img.dotSize
	}
	// 波浪删除线
	img.drawStrikeLine()
	// 正弦波扭曲（振幅 5~10，周期 100~200）
	img.waveDistort(randFloat64Between(5, 10), randFloat64Between(100, 200))
	// 背景圆点噪声
	img.scatterDiscs(d.NoiseDots, img.dotSize)
	return img, nil
}

// 编译期确保 DigitEngine 实现 Engine 接口。
var _ Engine = (*DigitEngine)(nil)
