package captcha

import "math"

// pcmSampleRate 是语音验证码的 PCM 采样率（8kHz，8bit 无符号单声道）。
const pcmSampleRate = 8000 // Hz

// tailBeep 是结尾提示音（主提示音加速 1.4 倍得到）。
var tailBeep []byte

func init() {
	tailBeep = resampleSpeed(beepTone, 1.4)
}

// mixPCM 把 src 混入 dst（逐样本非线性叠加混音）。
// 要求 len(dst) >= len(src)。
func mixPCM(dst, src []byte) {
	for i, v := range src {
		av := int(v)
		bv := int(dst[i])
		if av < 128 && bv < 128 {
			dst[i] = byte(av * bv / 128)
		} else {
			dst[i] = byte(2*(av+bv) - av*bv/128 - 256)
		}
	}
}

// scaleLevel 按比例 level 调整 PCM 数据的音量（以 128 为静音中轴）。
func scaleLevel(a []byte, level float64) {
	for i, v := range a {
		av := float64(v)
		switch {
		case av > 128:
			if av = (av-128)*level + 128; av < 128 {
				av = 128
			}
		case av < 128:
			if av = 128 - (128-av)*level; av > 128 {
				av = 128
			}
		default:
			continue
		}
		a[i] = byte(av)
	}
}

// resampleSpeed 按指定倍速重采样 PCM 数据，同时改变语速与音高。
// speed 必须在 (0, +∞) 内。
func resampleSpeed(a []byte, speed float64) []byte {
	b := make([]byte, int(math.Floor(float64(len(a))*speed)))
	var p float64
	for _, v := range a {
		// 一个源样本覆盖 speed 个目标样本，不足部分重复填充
		for i := int(p); i < int(p+speed); i++ {
			b[i] = v
		}
		p += speed
	}
	return b
}

// silencePCM 生成长度为 length 的静音样本（8bit 无符号静音值 128）。
func silencePCM(length int) []byte {
	b := make([]byte, length)
	for i := range b {
		b[i] = 128
	}
	return b
}

// reversePCM 返回时间上完全倒置的 PCM 数据（用作背景人声碎片）。
func reversePCM(a []byte) []byte {
	n := len(a)
	b := make([]byte, n)
	for i, v := range a {
		b[n-1-i] = v
	}
	return b
}
