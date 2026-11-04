package captcha

import (
	"embed"
	"math/rand"

	"github.com/golang/freetype/truetype"
)

// builtinFontFS 把仓库内 fonts/ 目录的全部字体嵌入二进制，
// 使用方无需再分发字体文件。
//
//go:embed fonts/*.ttf
//go:embed fonts/*.ttc
var builtinFontFS embed.FS

// BuiltinFontBox 是默认字体仓库（内嵌字体）。
var BuiltinFontBox = NewEmbedFontBox(builtinFontFS)

// builtinSimpleFonts 是西文手写风格字体集合。
var builtinSimpleFonts = BuiltinFontBox.OpenFonts([]string{
	"fonts/3Dumb.ttf",
	"fonts/ApothecaryFont.ttf",
	"fonts/DENNEthree-dee.ttf",
	"fonts/actionj.ttf",
})

// builtinChineseFonts 是中文字体集合（pinyin-cn 汉字带拼音标注），
// 中文/多语言验证码可用。
var builtinChineseFonts = BuiltinFontBox.OpenFonts([]string{
	"fonts/pinyin-cn.ttc",
	"fonts/chinese.ttf",
})

// allBuiltinFonts 是全部内嵌字体的汇总，引擎未指定字体时从中随机挑选。
var allBuiltinFonts = append(builtinSimpleFonts, builtinChineseFonts...)

// BuiltinFontNames 返回全部内嵌字体的文件名（不含 fonts/ 前缀，按名称排序），
// 可用于前端展示字体勾选列表；这些名字可直接传入各引擎的 fontNames 参数。
func BuiltinFontNames() []string {
	entries, err := builtinFontFS.ReadDir("fonts")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// pickFont 从字体集合中随机挑一个；集合为空时回退到全部内嵌字体。
func pickFont(fonts []*truetype.Font) *truetype.Font {
	fontCount := len(fonts)

	if fontCount == 0 {
		fonts = allBuiltinFonts
		fontCount = len(fonts)
	}
	index := rand.Intn(fontCount)
	return fonts[index]
}
