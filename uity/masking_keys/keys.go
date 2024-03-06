package masking_keys

import "sync"

const (
	MinKey uint32 = 1
	MaxKey uint32 = 4294967295
)

func NewKeys() *MaskingKeyHandle {
	return &MaskingKeyHandle{
		mu: sync.Mutex{},
		n:  0,
	}
}

type MaskingKeyHandle struct {
	mu sync.Mutex
	n  uint32
}

func (k *MaskingKeyHandle) next() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.n == MaxKey {
		k.n = MinKey
	} else {
		k.n += 1
	}
}
func (k *MaskingKeyHandle) Key() uint32 {
	k.next()
	return k.n
}
