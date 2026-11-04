package captcha

import (
	"strings"
	"sync"
	"time"
)

// SyncRepository 是基于 sync.Map 的内存答案仓库，
// 采用惰性过期：每次写入时顺手清扫已超时的记录。
type SyncRepository struct {
	liveTime time.Duration
	m        *sync.Map
}

// ttlEntry 是仓库中的带时间戳的值。
type ttlEntry struct {
	t     time.Time
	Value string
}

// NewSyncRepository 创建 sync.Map 仓库，liveTime 为记录存活时长。
func NewSyncRepository(liveTime time.Duration) *SyncRepository {
	return &SyncRepository{liveTime: liveTime, m: new(sync.Map)}
}

// purgeExpired 清扫超过 liveTime 的过期记录。
func (s SyncRepository) purgeExpired() {
	expireTime := time.Now().Add(-s.liveTime)
	s.m.Range(func(key, value interface{}) bool {
		if sv, ok := value.(*ttlEntry); ok && sv.t.Before(expireTime) {
			s.m.Delete(key)
		}
		return true
	})
}

// Put 实现 Repository：写入前先惰性清扫一次过期数据。
func (s SyncRepository) Put(id string, value string) error {
	s.purgeExpired()
	s.m.Store(id, &ttlEntry{t: time.Now(), Value: value})
	return nil
}

// Load 实现 Repository：consume 为 true 时读取后删除。
func (s SyncRepository) Load(id string, consume bool) string {
	v, ok := s.m.Load(id)
	if !ok {
		return ""
	}
	if consume {
		s.m.Delete(id)
	}
	if sv, ok := v.(*ttlEntry); ok {
		return sv.Value
	}
	return ""
}

// Match 实现 Repository：读取并忽略大小写比对。
func (s SyncRepository) Match(id, answer string, consume bool) bool {
	vv := s.Load(id, consume)
	return strings.EqualFold(vv, answer)
}

var _ Repository = (*SyncRepository)(nil)
