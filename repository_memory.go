package captcha

import (
	"container/list"
	"strings"
	"sync"
	"time"
)

// timedID 记录验证码 id 的写入时间，用于按时间顺序回收过期记录。
type timedID struct {
	timestamp time.Time
	id        string
}

// memRepository 是基于 map + 时间链表的内存答案仓库：
// 写入累计超过 collectThreshold 触发一次全量过期回收。
type memRepository struct {
	sync.RWMutex
	answersByID      map[string]string
	idTimeline       *list.List
	numStored        int           // 距上次回收累计写入条数
	collectThreshold int           // 触发回收的写入阈值
	ttl              time.Duration // 验证码存活时间
}

// NewMemRepository 创建内存答案仓库。
//
// 参数：
//
//	collectThreshold 累计写入多少条后触发过期回收，常用 10240
//	ttl              验证码存活时长，常用 10*time.Minute
func NewMemRepository(collectThreshold int, ttl time.Duration) Repository {
	s := new(memRepository)
	s.answersByID = make(map[string]string)
	s.idTimeline = list.New()
	s.collectThreshold = collectThreshold
	s.ttl = ttl
	return s
}

// Put 实现 Repository：写入答案并追加时间线，达到阈值时异步触发回收。
func (s *memRepository) Put(id string, value string) error {
	s.Lock()
	s.answersByID[id] = value
	s.idTimeline.PushBack(timedID{time.Now(), id})
	s.numStored++
	needCollect := s.numStored > s.collectThreshold
	s.Unlock()
	if needCollect {
		go s.purgeExpired()
	}
	return nil
}

// Match 实现 Repository：读取并忽略大小写比对。
// id 或 answer 为空一律不匹配。
func (s *memRepository) Match(id, answer string, consume bool) bool {
	if id == "" || answer == "" {
		return false
	}
	v := s.Load(id, consume)
	return strings.EqualFold(v, answer)
}

// Load 实现 Repository：consume 为 true 加写锁并删除记录（一次性），
// 否则加读锁只读。
func (s *memRepository) Load(id string, consume bool) (value string) {
	if !consume {
		s.RLock()
		defer s.RUnlock()
	} else {
		s.Lock()
		defer s.Unlock()
	}
	value, ok := s.answersByID[id]
	if !ok {
		return
	}
	if consume {
		delete(s.answersByID, id)
	}
	return
}

// purgeExpired 清扫时间线中已过期的记录。
func (s *memRepository) purgeExpired() {
	now := time.Now()
	s.Lock()
	defer s.Unlock()
	for e := s.idTimeline.Front(); e != nil; {
		e = s.purgeOne(e, now)
	}
}

// purgeOne 检查链表头部元素：过期则删除并返回下一个元素；
// 未过期说明后续都未过期，返回 nil 终止本轮清扫。
func (s *memRepository) purgeOne(e *list.Element, specifyTime time.Time) *list.Element {
	ev, ok := e.Value.(timedID)
	if !ok {
		return nil
	}

	if ev.timestamp.Add(s.ttl).Before(specifyTime) {
		delete(s.answersByID, ev.id)
		next := e.Next()
		s.idTimeline.Remove(e)
		s.numStored--
		return next
	}
	return nil
}

var (
	// DefaultCollectThreshold 是默认仓库触发回收的写入条数。
	DefaultCollectThreshold = 10240
	// DefaultTTL 是默认仓库中验证码的存活时长。
	DefaultTTL = 10 * time.Minute
	// DefaultRepository 是进程内共享的默认答案仓库，
	// NewService 未显式指定仓库时使用它。
	DefaultRepository = NewMemRepository(DefaultCollectThreshold, DefaultTTL)
)
