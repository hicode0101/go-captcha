package captcha

// AudioEngine 语音验证码引擎。
// 先朗读三声提示音，然后按随机间隔朗读各数字，背景铺白噪声与
// 倒置人声碎片，最后以一声提示音收尾；输出 WAV（VoiceClip）。
type AudioEngine struct {
	// DigitCount 验证码包含的数字个数。
	DigitCount int
	// Language 数字发音语言，取值 "en"、"ja"、"ru"、"zh"；
	// 不支持的语言回退英语。
	Language string
}

// NewAudioEngine 创建语音验证码引擎。
//
// 参数：
//
//	digitCount 数字个数，常用 6；<=0 时按 6 个出题
//	language   发音语言（en/ja/ru/zh）
func NewAudioEngine(digitCount int, language string) *AudioEngine {
	return &AudioEngine{DigitCount: digitCount, Language: language}
}

// DefaultAudioEngine 是语音引擎的推荐默认配置。
var DefaultAudioEngine = NewAudioEngine(6, "en")

// NewChallenge 生成语音验证码挑战：question 与 answer 均为数字字符串。
func (d *AudioEngine) NewChallenge() (id, question, answer string) {
	id = NewSessionID()
	length := d.DigitCount
	if length <= 0 {
		length = 6
	}
	digits := randomDigitBytes(length)
	answer = digitsToString(digits)
	return id, answer, answer
}

// Render 把数字字符串合成为语音验证码。
// content 必须由 '0'~'9' 组成；返回的 VoiceClip 可输出 WAV 或 base64。
func (d *AudioEngine) Render(content string) (item Artifact, err error) {
	digits := stringToDigitBytes(content)
	clip := composeVoiceClip("", digits, d.Language)
	return clip, nil
}

var _ Engine = (*AudioEngine)(nil)
