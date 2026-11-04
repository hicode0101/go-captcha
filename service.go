package captcha

import "strings"

// Service 把"出题引擎"与"答案仓库"组装成完整的验证码服务，
// 是业务方接入的主入口。Service 并发安全，可全局持有一个实例。
type Service struct {
	engine Engine
	repo   Repository
}

// NewService 创建验证码服务。
//
// 参数：
//
//	engine 出题与渲染引擎，见 DigitEngine / StringEngine / MathEngine /
//	       ChineseEngine / AudioEngine
//	repo   答案存储仓库；传 nil 时使用 DefaultRepository
//	       （进程内内存仓库，10 分钟过期）。多实例共享同一仓库即可
//	       实现跨服务的验证码校验。
func NewService(engine Engine, repo Repository) *Service {
	if repo == nil {
		repo = DefaultRepository
	}
	return &Service{engine: engine, repo: repo}
}

// Create 生成一道完整的验证码。
//
// 返回值：
//
//	id      会话 id，需与 encoded 一起下发给前端，校验时原样传回
//	encoded 带前缀的 base64 产物（data:image/png 或 data:audio/wav），
//	        直接用于 <img>/<audio> 展示
//	answer  正确答案明文，仅建议在调试场景打印，生产环境勿下发
//	err     出题或渲染失败时非 nil（此时其余返回值为空）
//
// 每次 Create 都会向 repo 写入一条记录，过期与回收策略由仓库决定。
func (s *Service) Create() (id, encoded, answer string, err error) {
	id, question, answer := s.engine.NewChallenge()
	art, err := s.engine.Render(question)
	if err != nil {
		return "", "", "", err
	}
	if err = s.repo.Put(id, answer); err != nil {
		return "", "", "", err
	}
	encoded = art.Base64String()
	return id, encoded, answer, nil
}

// Check 校验用户答案。
//
// 参数：
//
//	id      Create 时下发的会话 id
//	answer  用户输入的答案；两侧都会去掉首尾空白后忽略大小写比较
//	consume true 表示无论成败都消费掉该验证码（防重放，推荐）；
//	        false 允许反复校验，直到仓库过期
//
// 返回是否校验通过。
func (s *Service) Check(id, answer string, consume bool) bool {
	stored := s.repo.Load(id, consume)
	stored = strings.TrimSpace(stored)
	answer = strings.TrimSpace(answer)
	return strings.EqualFold(stored, answer)
}
