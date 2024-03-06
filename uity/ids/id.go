package ids

import (
	"fmt"
	"sync"
	"time"
)

const (
	MaxUin32 uint32 = 922337202 // 922337203
	MinUin32 uint32 = 1
)

const MinTime = 1000000000

var idHandle *Uint64Id
var idOnce sync.Once

func GetId() int64 {
	return NewUint64Id().GetInt()
}

// NewUint64Id     生成全局唯一的id生成器
func NewUint64Id() *Uint64Id {
	idOnce.Do(func() {
		idHandle = &Uint64Id{
			mu:       sync.Mutex{},
			timeUnix: MinTime,
			u32:      MinUin32,
		}
	})
	return idHandle
}

// Uint64Id     id生成器
type Uint64Id struct {
	mu       sync.Mutex // 读写锁
	timeUnix int64      // id记录的秒级时间
	u32      uint32     // 顺序的uint32
}

func (id *Uint64Id) next() {
	id.mu.Lock()
	defer id.mu.Unlock()
	t := time.Now().Unix()
	if t > id.timeUnix {
		id.timeUnix = t
		id.u32 = MinUin32
	} else {
		if id.u32 == MaxUin32 {
			id.timeUnix += 1
			id.u32 = MinUin32
		} else {
			id.u32 += 1
		}
	}
}
func (id *Uint64Id) GetInt() int64 {
	id.next()
	return int64(id.u32)*10000000000 + id.timeUnix
}
func (id *Uint64Id) GetString() string {
	id.next()
	return fmt.Sprintf("%d%010d", id.timeUnix, id.u32)
}
