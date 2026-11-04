package captcha

import "io"

// Engine 是验证码引擎的统一抽象：负责出一道"挑战题"，并把题目渲染成
// 可传输的产物（图片或音频）。五种内置验证码均实现该接口，也可自行扩展。
type Engine interface {
	// NewChallenge 生成一道新的验证码挑战。
	//
	// 返回值：
	//   id       会话唯一标识，调用方需原样传回 Check/Match 做比对
	//   question 题目内容（即绘制到图片/音频中的原文）
	//   answer   正确答案；与 question 不同（如算术题 question 为 "3+5=?"，
	//            answer 为 "8"），应存入 Repository 而非下发给前端
	NewChallenge() (id, question, answer string)

	// Render 把题目内容渲染为验证码产物。
	//
	// question 必须是 NewChallenge 返回的 question（或调用方自定的同构文本），
	// 渲染失败（如空白文本、字体缺失）时返回 error。
	Render(question string) (Artifact, error)
}

// Artifact 是验证码产物的统一抽象：一张 PNG 图片或一段 WAV 音频。
type Artifact interface {
	// SaveTo 把产物以二进制（PNG 或 WAV）格式写入 w，
	// 返回写入的字节数；编码失败以 panic 形式暴露（属于编程错误）。
	SaveTo(w io.Writer) (n int64, err error)

	// Base64String 返回形如 "data:image/png;base64,..." 的完整 base64 字符串，
	// 可直接赋给 <img src> 或 <audio src>，无需前端再拼接。
	Base64String() string
}

// Repository 是验证码答案的存储抽象，负责 id → answer 的存取与比对。
// 内置内存仓库（NewMemRepository / DefaultRepository）与
// sync.Map 仓库（NewSyncRepository）；接入 Redis 等外部存储时实现本接口即可。
type Repository interface {
	// Put 保存一道挑战的正确答案。
	Put(id string, answer string) error

	// Load 读取 id 对应的答案；consume 为 true 时读取后立即删除（一次性校验）。
	// 不存在时返回空字符串。
	Load(id string, consume bool) string

	// Match 直接完成 "读取 + 忽略大小写比对（+ 可选删除）" 的组合动作。
	Match(id, answer string, consume bool) bool
}
