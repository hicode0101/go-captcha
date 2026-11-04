package captcha

import (
	"encoding/binary"
	"image/color"
	"math"
	"math/rand"
	"strings"
)

// RandRunes 从候选字符集 charset 中随机取 size 个字符（按 rune 计）。
// charset 为空或 size 为 0 时返回空串；size 大于字符集长度时自动重复扩展，
// 以保证有足够的采样空间。
func RandRunes(size int, charset string) string {
	if charset == "" || size == 0 {
		return ""
	}

	if size >= len(charset) {
		charset = strings.Repeat(charset, size)
	}

	sourceRunes := []rune(charset)
	sourceLength := len(sourceRunes)

	text := make([]rune, size)
	for i := range text {
		text[i] = sourceRunes[rand.Intn(sourceLength)]
	}
	return string(text)
}

// randFloatBetween 生成 [min, max) 区间内的随机浮点数（内部换算用）。
func randFloatBetween(min int64, max int64) float64 {
	return float64(min) + rand.Float64()*float64(max-min)
}

// RandomDarkColor 随机生成一个深色系颜色（用于绘制主体文字，
// 与浅色背景形成对比）。
func RandomDarkColor() color.RGBA {
	base := RandomVividColor()

	increase := float64(30 + rand.Intn(255))

	red := math.Abs(math.Min(float64(base.R)-increase, 255))
	green := math.Abs(math.Min(float64(base.G)-increase, 255))
	blue := math.Abs(math.Min(float64(base.B)-increase, 255))

	return color.RGBA{R: uint8(red), G: uint8(green), B: uint8(blue), A: uint8(255)}
}

// RandomLightTone 随机生成一个浅色系颜色（用于背景或浅色干扰元素）。
func RandomLightTone() color.RGBA {
	red := rand.Intn(55) + 200
	green := rand.Intn(55) + 200
	blue := rand.Intn(55) + 200
	return color.RGBA{R: uint8(red), G: uint8(green), B: uint8(blue), A: uint8(255)}
}

// RandomVividColor 随机生成一个饱和度较高的基础颜色（作为其他配色函数的种子）。
func RandomVividColor() color.RGBA {
	red := rand.Intn(255)
	green := rand.Intn(255)
	var blue int
	if (red + green) > 400 {
		blue = 0
	} else {
		blue = 400 - green - red
	}
	if blue > 255 {
		blue = 255
	}
	return color.RGBA{R: uint8(red), G: uint8(green), B: uint8(blue), A: uint8(255)}
}

// randIntBetween 生成 [from, to) 区间内的随机整数；区间非法时退化为 from，
// 避免极端参数下 rand.Intn panic。
func randIntBetween(from, to int) int {
	if to-from <= 0 {
		return from
	}
	return rand.Intn(to-from) + from
}

// randFloat64Between 生成 [from, to) 区间内的随机浮点数。
func randFloat64Between(from, to float64) float64 {
	return rand.Float64()*(to-from) + from
}

// pseudoRandomBytes 生成 n 字节的伪随机数据（数学噪声场景使用，
// 与安全无关；校验 id 走 cryptoRandomBytes）。
func pseudoRandomBytes(n int) []byte {
	numBlocks := (n + 8 - 1) / 8
	b := make([]byte, numBlocks*8)
	for i := 0; i < len(b); i += 8 {
		binary.LittleEndian.PutUint64(b[i:], rand.Uint64())
	}
	return b[:n]
}

// NewSessionID 生成验证码会话 id：长度 codeLen，取自数字与字母表，
// 随机源为 CSPRNG 并做了模偏差消除。
func NewSessionID() string {
	b := cryptoRandomBytesMod(codeLen, byte(len(codeAlphabet)))
	for i, c := range b {
		b[i] = codeAlphabet[c]
	}
	return string(b)
}
