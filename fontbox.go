package captcha

import (
	"embed"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
)

// FontBox 是字体仓库抽象：验证码引擎从仓库按文件名加载字体。
// 内置仓库 BuiltinFontBox 已把 fonts/ 目录下的全部字体嵌入二进制；
// 需要自定义字体（如从磁盘或网络加载）时实现本接口即可。
type FontBox interface {
	// OpenFont 按字体文件名加载并解析单个字体。
	// name 形如 "fonts/pinyin-cn.ttc"；读取或解析失败时 panic
	//（字体缺失属于部署错误，应在启动期暴露）。
	OpenFont(name string) *truetype.Font

	// OpenFonts 按文件名列表批量加载字体，顺序与入参一致。
	OpenFonts(names []string) []*truetype.Font
}

// EmbedFontBox 基于 embed.FS 的字体仓库实现。
type EmbedFontBox struct {
	fs embed.FS
}

// NewEmbedFontBox 用一个 embed.FS 构造字体仓库。
func NewEmbedFontBox(fs embed.FS) *EmbedFontBox {
	return &EmbedFontBox{fs: fs}
}

// OpenFont 实现 FontBox：读取字体文件并解析为 truetype.Font。
func (s *EmbedFontBox) OpenFont(name string) *truetype.Font {
	fontBytes, err := s.fs.ReadFile(name)
	if err != nil {
		panic(err)
	}

	// 字体二进制 → TrueType 字体对象
	trueTypeFont, err := freetype.ParseFont(fontBytes)
	if err != nil {
		panic(err)
	}

	return trueTypeFont
}

// OpenFonts 实现 FontBox：按名称列表批量加载。
func (s *EmbedFontBox) OpenFonts(names []string) []*truetype.Font {
	fonts := make([]*truetype.Font, 0, len(names))
	for _, name := range names {
		fonts = append(fonts, s.OpenFont(name))
	}
	return fonts
}
