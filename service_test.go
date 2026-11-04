package captcha

import (
	"image/color"
	"strings"
	"testing"
	"time"
)

// TestCreateAndCheckAllEngines 覆盖五种引擎的完整出题-校验链路。
func TestCreateAndCheckAllEngines(t *testing.T) {
	engines := map[string]Engine{
		"digit":   NewDigitEngine(80, 240, 5, 0.7, 80),
		"string":  NewStringEngine(60, 240, 5, LineOptionHollow|LineOptionSine, 6, CharsetDigits+CharsetLetters, nil, nil, []string{"pinyin-cn.ttc"}),
		"math":    NewMathEngine(60, 240, 5, LineOptionSlime, nil, nil, nil),
		"chinese": NewChineseEngine(60, 320, 5, 0, 2, CharsetChinese, nil, nil, []string{"chinese.ttf"}),
		"audio":   NewAudioEngine(6, "zh"),
	}

	for name, e := range engines {
		t.Run(name, func(t *testing.T) {
			svc := NewService(e, NewMemRepository(10240, 10*time.Minute))
			id, b64s, answer, err := svc.Create()
			if err != nil {
				t.Fatalf("%s create failed: %v", name, err)
			}
			if id == "" {
				t.Fatalf("%s: empty id", name)
			}
			if !strings.HasPrefix(b64s, "data:") {
				t.Fatalf("%s: bad base64 content %q", name, b64s[:30])
			}
			if !svc.Check(id, answer, true) {
				t.Fatalf("%s: check with correct answer failed", name)
			}
			if svc.Check(id, answer, true) {
				t.Fatalf("%s: check must fail after consume", name)
			}
		})
	}
}

// TestDigitEngineExtremes 极端参数（数字个数多、画布小）不得 panic。
func TestDigitEngineExtremes(t *testing.T) {
	e := NewDigitEngine(80, 240, 10, 0.9, 100)
	_, b64s, _, err := NewService(e, DefaultRepository).Create()
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if b64s == "" {
		t.Fatal("empty b64")
	}
}

// TestEngineWithBgColor 显式背景色渲染。
func TestEngineWithBgColor(t *testing.T) {
	bg := color.RGBA{R: 125, G: 125, B: 0, A: 118}
	e := NewStringEngine(60, 240, 0, 0, 4, CharsetDigits, &bg, nil, []string{"actionj.ttf"})
	if _, _, _, err := NewService(e, DefaultRepository).Create(); err != nil {
		t.Fatalf("create failed: %v", err)
	}
}

// TestBuiltinFontNames 内嵌字体清单非空且包含全部中文字体。
func TestBuiltinFontNames(t *testing.T) {
	names := BuiltinFontNames()
	if len(names) == 0 {
		t.Fatal("no builtin fonts")
	}
	have := make(map[string]bool, len(names))
	for _, n := range names {
		have[n] = true
	}
	for _, want := range []string{"chinese.ttf", "pinyin-cn.ttc"} {
		if !have[want] {
			t.Fatalf("font %s missing", want)
		}
	}
}

// TestSyncRepository sync.Map 仓库的存取与消费语义。
func TestSyncRepository(t *testing.T) {
	r := NewSyncRepository(10 * time.Minute)
	if err := r.Put("k1", "v1"); err != nil {
		t.Fatal(err)
	}
	if !r.Match("k1", "v1", true) {
		t.Fatal("match failed")
	}
	if r.Match("k1", "v1", true) {
		t.Fatal("second match must fail after consume")
	}
}
