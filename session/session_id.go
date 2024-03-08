package session

import (
	"fmt"
	"sync"
	"time"
)

const (
	maxUin32 uint32 = 922337202 // 922337203
	minUin32 uint32 = 1
)

const minTime = 1000000000

var idHandle *sessionIdManager
var idOnce sync.Once

func getSessionId() int64 {
	return getSessionIdManager().GetInt()
}

// getSessionIdManager     生成全局唯一的id生成器,服务端session用
func getSessionIdManager() *sessionIdManager {
	idOnce.Do(func() {
		idHandle = &sessionIdManager{
			mu:       sync.Mutex{},
			timeUnix: minTime,
			u32:      minUin32,
		}
	})
	return idHandle
}

// sessionIdManager     id生成器
type sessionIdManager struct {
	mu       sync.Mutex // 读写锁
	timeUnix int64      // id记录的秒级时间
	u32      uint32     // 顺序的uint32
}

func (id *sessionIdManager) next() {
	id.mu.Lock()
	defer id.mu.Unlock()
	t := time.Now().Unix()
	if t > id.timeUnix {
		id.timeUnix = t
		id.u32 = minUin32
	} else {
		if id.u32 == maxUin32 {
			id.timeUnix += 1
			id.u32 = minUin32
		} else {
			id.u32 += 1
		}
	}
}
func (id *sessionIdManager) GetInt() int64 {
	id.next()
	return int64(id.u32)*10000000000 + id.timeUnix
}
func (id *sessionIdManager) GetString() string {
	id.next()
	return fmt.Sprintf("%d%010d", id.timeUnix, id.u32)
}
