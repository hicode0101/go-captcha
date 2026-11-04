// go-captcha web 示例后端：接收前端调参请求，实时生成/校验验证码。
package main

import (
	"encoding/json"
	"flag"
	"image/color"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/golang/freetype/truetype"

	captcha "github.com/hicode0101/go-captcha"
)

// generateRequest 汇总了所有引擎的可选参数，按 driver 类型取用。
type generateRequest struct {
	Driver string `json:"driver"` // digit | string | math | chinese | audio

	Height          int      `json:"height"`
	Width           int      `json:"width"`
	Length          int      `json:"length"`
	MaxSkew         float64  `json:"maxSkew"`
	DotCount        int      `json:"dotCount"`
	NoiseCount      int      `json:"noiseCount"`
	ShowLineOptions int      `json:"showLineOptions"`
	Source          string   `json:"source"`
	Language        string   `json:"language"`
	BgColor         *rgba    `json:"bgColor"`
	Fonts           []string `json:"fonts"`
}

// rgba 是前端传来的背景色；四通道全 0 时与库约定一致，视为 nil（随机浅色背景）。
type rgba struct {
	R int `json:"r"`
	G int `json:"g"`
	B int `json:"b"`
	A int `json:"a"`
}

func (c *rgba) toColor() *color.RGBA {
	if c == nil || (c.R == 0 && c.G == 0 && c.B == 0 && c.A == 0) {
		return nil
	}
	return &color.RGBA{R: uint8(c.R), G: uint8(c.G), B: uint8(c.B), A: uint8(c.A)}
}

// fontMode 记录演示服务的字体来源：磁盘目录（可实时增删）或编译期内嵌。
var (
	fontDir string          // 磁盘字体目录；embedded 模式下为空
	fontBox captcha.FontBox // nil 表示使用库内嵌字体仓库
)

// DirFontBox 基于磁盘目录的字体仓库：调试模式下 fonts 目录新增/删除字体文件
// 无需重新编译即可生效。OpenFont 读取或解析失败时 panic（与内嵌仓库语义一致）。
type DirFontBox struct {
	root string
}

// NewDirFontBox 以 root 为字体目录构造仓库。
func NewDirFontBox(root string) *DirFontBox { return &DirFontBox{root: root} }

// OpenFont 实现 captcha.FontBox：name 形如 "fonts/xxx.ttf"，取基名到 root 下读取。
func (b *DirFontBox) OpenFont(name string) *truetype.Font {
	data, err := os.ReadFile(filepath.Join(b.root, path.Base(name)))
	if err != nil {
		panic(err)
	}
	tf, err := truetype.Parse(data)
	if err != nil {
		panic(err)
	}
	return tf
}

// OpenFonts 实现 captcha.FontBox：按名称列表批量加载。
func (b *DirFontBox) OpenFonts(names []string) []*truetype.Font {
	fonts := make([]*truetype.Font, 0, len(names))
	for _, name := range names {
		fonts = append(fonts, b.OpenFont(name))
	}
	return fonts
}

// listFontDir 枚举目录下的 .ttf/.ttc 字体文件名（按名称排序）。
// 目录不存在或不可读时返回 nil。
func listFontDir(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".ttf", ".ttc":
			names = append(names, e.Name())
		}
	}
	return names
}

// currentFontNames 返回当前可用字体清单：
// 磁盘模式实时枚举 fonts 目录（勾选列表随目录内容自动更新），否则回退内嵌清单。
func currentFontNames() []string {
	if fontBox != nil {
		if names := listFontDir(fontDir); len(names) > 0 {
			return names
		}
	}
	return captcha.BuiltinFontNames()
}

// buildEngine 根据请求参数构造对应的验证码引擎。
func buildEngine(req generateRequest) captcha.Engine {
	bg := req.BgColor.toColor()
	fonts := req.Fonts
	if fontBox != nil {
		live := listFontDir(fontDir)
		liveSet := make(map[string]bool, len(live))
		for _, n := range live {
			liveSet[n] = true
		}
		// 过滤已从磁盘删除的字体，避免勾选列表过期导致渲染 panic
		kept := fonts[:0]
		for _, f := range fonts {
			if liveSet[f] {
				kept = append(kept, f)
			} else {
				log.Printf("font %q not in %s, skipped", f, fontDir)
			}
		}
		fonts = kept
		// 未勾选任何字体时使用磁盘字体全集，新建字体自动进入随机池
		if len(fonts) == 0 {
			fonts = live
		}
	} else {
		// 内嵌模式同样过滤：剔除不在内嵌清单里的名字，
		// 防止过期勾选或手工请求触发 OpenFont panic（库约定缺失字体快速失败）
		knownSet := make(map[string]bool, 16)
		for _, n := range captcha.BuiltinFontNames() {
			knownSet[n] = true
		}
		kept := fonts[:0]
		for _, f := range fonts {
			if knownSet[f] {
				kept = append(kept, f)
			} else {
				log.Printf("font %q not builtin, skipped", f)
			}
		}
		fonts = kept
	}
	switch req.Driver {
	case "digit":
		return captcha.NewDigitEngine(req.Height, req.Width, req.Length, req.MaxSkew, req.DotCount)
	case "math":
		return captcha.NewMathEngine(req.Height, req.Width, req.NoiseCount, req.ShowLineOptions, bg, fontBox, fonts)
	case "chinese":
		return captcha.NewChineseEngine(req.Height, req.Width, req.NoiseCount, req.ShowLineOptions, req.Length, req.Source, bg, fontBox, fonts)
	case "audio":
		return captcha.NewAudioEngine(req.Length, req.Language)
	default: // string
		return captcha.NewStringEngine(req.Height, req.Width, req.NoiseCount, req.ShowLineOptions, req.Length, req.Source, bg, fontBox, fonts)
	}
}

type response struct {
	Code    int         `json:"code"` // 0 成功，非 0 失败
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, resp response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, msg string) {
	writeJSON(w, response{Code: 1, Message: msg})
}

// applyDefaults 补齐前端未传的参数，避免生成 0 尺寸图片。
func (req *generateRequest) applyDefaults() {
	if req.Height <= 0 {
		req.Height = 80
	}
	if req.Width <= 0 {
		req.Width = 240
	}
	if req.Length <= 0 {
		req.Length = 4
	}
	if req.MaxSkew <= 0 {
		req.MaxSkew = 0.7
	}
	if req.DotCount < 0 {
		req.DotCount = 0
	}
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid json body: "+err.Error())
		return
	}
	req.applyDefaults()

	svc := captcha.NewService(buildEngine(req), captcha.DefaultRepository)
	id, b64s, _, err := svc.Create()
	if err != nil {
		writeError(w, "generate failed: "+err.Error())
		return
	}
	writeJSON(w, response{Code: 0, Data: map[string]string{"id": id, "b64s": b64s}})
}

type verifyRequest struct {
	ID     string `json:"id"`
	Answer string `json:"answer"`
}

func handleVerify(w http.ResponseWriter, r *http.Request) {
	var req verifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid json body: "+err.Error())
		return
	}
	ok := captcha.DefaultRepository.Match(req.ID, req.Answer, true)
	writeJSON(w, response{Code: 0, Data: map[string]bool{"ok": ok}})
}

func handleFonts(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, response{Code: 0, Data: currentFontNames()})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// noCache 禁用缓存：演示页与字体列表频繁变化，避免浏览器拿到旧页面误判字体清单。
func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		next.ServeHTTP(w, r)
	})
}

func main() {
	addr := flag.String("addr", ":8090", "http listen address")
	staticDir := flag.String("static", "./static", "frontend static files dir")
	fontsFlag := flag.String("fonts", "../../../fonts", "fonts dir; new/deleted font files take effect without rebuild, falls back to embedded fonts if unreadable")
	flag.Parse()

	// 磁盘目录可读且有字体文件时进入磁盘模式，勾选列表随 fonts 目录自动更新；
	// 否则回退编译期内嵌字体。
	fontDir = *fontsFlag
	if names := listFontDir(fontDir); len(names) > 0 {
		fontBox = NewDirFontBox(fontDir)
		log.Printf("fonts: disk mode, %d fonts in %s", len(names), fontDir)
	} else {
		fontDir = ""
		log.Printf("fonts: embedded mode (%s unreadable or has no ttf/ttc)", *fontsFlag)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/generate", handleGenerate)
	mux.HandleFunc("POST /api/verify", handleVerify)
	mux.HandleFunc("GET /api/fonts", handleFonts)
	mux.Handle("/", http.FileServer(http.Dir(*staticDir)))

	log.Printf("captcha demo listening on http://localhost%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, cors(noCache(mux))))
}
