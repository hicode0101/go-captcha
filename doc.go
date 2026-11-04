// Package captcha 是一个开箱即用的 Go 图形验证码库。
//
// 支持五种验证码引擎（Engine）：
//
//	DigitEngine    数字点阵验证码（波形扭曲 + 圆点噪声）
//	StringEngine   字符串验证码（可自定义字符集 / 字体 / 干扰线）
//	MathEngine     算术验证码（加、减、乘，答案为计算结果）
//	ChineseEngine  中文验证码（候选字集逗号分隔）
//	AudioEngine    语音验证码（输出 WAV，支持 en/zh/ru/ja 发音）
//
// 典型用法：
//
//	engine := captcha.NewDigitEngine(80, 240, 5, 0.7, 80)
//	svc := captcha.NewService(engine, captcha.DefaultRepository)
//	id, b64s, _, err := svc.Create()
//	// b64s 形如 data:image/png;base64,...，可直接放入 <img src>
//	ok := svc.Check(id, "用户输入", true)
//
// 所有产物（Artifact）输出带 MIME 前缀的 base64 字符串，前端零适配；
// 字体已内嵌（含支持中文的 chinese.ttf / pinyin-cn.ttc），也可通过 FontBox 注入自定义字体。
//
// 在保持绘制/音频算法一致的前提下重新组织了代码结构与 API 命名。
package captcha
