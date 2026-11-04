package captcha

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
)

// VoiceClip 是语音验证码产物：一段拼装完成的 8kHz 8bit 单声道 PCM，
// 可输出为 WAV 或带前缀的 base64 字符串。
type VoiceClip struct {
	answer    string
	body      *bytes.Buffer
	digitClip [][]byte // 当前语言的数字发音样本
}

// composeVoiceClip 把数字序列拼装为一段语音验证码。
//
// digits 中的每个元素必须落在 0-9；lang 取 "en"/"ja"/"ru"/"zh"，
// 若该语言缺少发音样本则回退英语。
//
// 音频结构：三声前置提示音 → 随机间隔朗读各数字 → 背景白噪声与人声碎片 → 结尾提示音。
func composeVoiceClip(id string, digits []byte, lang string) *VoiceClip {
	a := new(VoiceClip)

	if sounds, ok := languageDigitSamples[lang]; ok {
		a.digitClip = sounds
	} else {
		a.digitClip = languageDigitSamples["en"]
	}
	numsnd := make([][]byte, len(digits))
	for i, n := range digits {
		snd := a.jitterDigitSample(n)
		scaleLevel(snd, 1.5)
		numsnd[i] = snd
	}
	// 各数字之间（含开头）的随机间隔
	intervals := make([]int, len(digits)+1)
	intdur := 0
	for i := range intervals {
		dur := randIntBetween(pcmSampleRate, pcmSampleRate*2) // 1~2 秒
		intdur += dur
		intervals[i] = dur
	}
	// 生成背景噪声床，长度足够容纳全部数字朗读
	bg := a.buildNoiseBed(a.longestSampleLen()*len(digits) + intdur)
	// 组装缓冲区：前置三声 beep + 背景混合数字 + 结尾 beep
	sil := silencePCM(pcmSampleRate / 5)
	bufcap := 3*len(beepTone) + 2*len(sil) + len(bg) + len(tailBeep)
	a.body = bytes.NewBuffer(make([]byte, 0, bufcap))
	a.body.Write(beepTone)
	a.body.Write(sil)
	a.body.Write(beepTone)
	a.body.Write(sil)
	a.body.Write(beepTone)
	// 把数字朗读混入背景噪声床的对应位置
	pos := intervals[0]
	for i, v := range numsnd {
		mixPCM(bg[pos:], v)
		pos += len(v) + intervals[i+1]
	}
	a.body.Write(bg)
	a.body.Write(tailBeep)
	return a
}

// wavTotalLen 返回 WAV 编码后音频的总长度。
func (a *VoiceClip) wavTotalLen() int {
	return len(wavHeader) + 4 + a.body.Len()
}

// buildNoiseBed 生成长度为 length 的背景噪声床：
// 低电平白噪声之上，随机位置混入若干低音量的倒置数字人声碎片。
func (a *VoiceClip) buildNoiseBed(length int) []byte {
	b := a.whiteNoiseBytes(length, 4)
	for i := 0; i < length/(pcmSampleRate/10); i++ {
		snd := reversePCM(a.digitClip[rand.Intn(10)])
		place := rand.Intn(len(b) - len(snd))
		scaleLevel(snd, randFloat64Between(0.04, 0.08))
		mixPCM(b[place:], snd)
	}
	return b
}

// jitterDigitSample 对单个数字的发音样本做随机变速与音量抖动，
// 使每次播放的音频都不完全相同，增加机器识别难度。
func (a *VoiceClip) jitterDigitSample(n byte) []byte {
	s := a.shiftPitch(a.digitClip[n])
	scaleLevel(s, randFloat64Between(0.85, 1.2))
	return s
}

// longestSampleLen 返回当前语言里最长的数字发音样本长度。
func (a *VoiceClip) longestSampleLen() int {
	n := 0
	for _, v := range a.digitClip {
		if n < len(v) {
			n = len(v)
		}
	}
	return n
}

// shiftPitch 以 0.95~1.1 的随机倍速重采样，微调音高。
func (a *VoiceClip) shiftPitch(b []byte) []byte {
	pitch := randFloat64Between(0.95, 1.1)
	return resampleSpeed(b, pitch)
}

// whiteNoiseBytes 生成长度为 length 的白噪声，电平由 level 控制。
func (a *VoiceClip) whiteNoiseBytes(length int, level uint8) []byte {
	noise := pseudoRandomBytes(length)
	adj := 128 - level/2
	for i, v := range noise {
		v %= level
		v += adj
		noise[i] = v
	}
	return noise
}

// SaveTo 实现 Artifact：把音频以 WAV 格式写入 w，并返回写入字节数。
// WAV 头中的两处长度字段按实际数据长度回填；奇数长度补一个填充字节。
func (a *VoiceClip) SaveTo(w io.Writer) (n int64, err error) {
	bodyLen := uint32(a.body.Len())
	paddedBodyLen := bodyLen
	if bodyLen%2 != 0 {
		paddedBodyLen++
	}
	totalLen := uint32(len(wavHeader)) - 4 + paddedBodyLen
	// 头部（预留 4 字节 chunk size）
	header := make([]byte, len(wavHeader)+4)
	copy(header, wavHeader)
	// RIFF 块总长度
	binary.LittleEndian.PutUint32(header[4:], totalLen)
	// WAVE 数据块长度
	binary.LittleEndian.PutUint32(header[len(wavHeader):], bodyLen)
	nn, err := w.Write(header)
	n = int64(nn)
	if err != nil {
		return
	}
	// PCM 数据
	n, err = a.body.WriteTo(w)
	n += int64(nn)
	if err != nil {
		return
	}
	// 奇数长度补齐
	if bodyLen != paddedBodyLen {
		w.Write([]byte{0})
		n++
	}
	return
}

// Base64String 实现 Artifact：返回 data:audio/wav;base64,... 字符串。
func (a *VoiceClip) Base64String() string {
	var buf bytes.Buffer
	if _, err := a.SaveTo(&buf); err != nil {
		panic(err)
	}
	return fmt.Sprintf("data:%s;base64,%s", MimeAudioWav, base64.StdEncoding.EncodeToString(buf.Bytes()))
}
