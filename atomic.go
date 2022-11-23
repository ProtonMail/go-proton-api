package proton

import "sync/atomic"

type atomicUint64 struct {
	v uint64
}

func (x *atomicUint64) Load() uint64 { return atomic.LoadUint64(&x.v) }

func (x *atomicUint64) Store(val uint64) { atomic.StoreUint64(&x.v, val) }

func (x *atomicUint64) Swap(new uint64) (old uint64) { return atomic.SwapUint64(&x.v, new) }

func (x *atomicUint64) Add(delta uint64) (new uint64) { return atomic.AddUint64(&x.v, delta) }

type atomicBool struct {
	v uint32
}

func (x *atomicBool) Load() bool { return atomic.LoadUint32(&x.v) != 0 }

func (x *atomicBool) Store(val bool) { atomic.StoreUint32(&x.v, b32(val)) }

func (x *atomicBool) Swap(new bool) (old bool) { return atomic.SwapUint32(&x.v, b32(new)) != 0 }

func b32(b bool) uint32 {
	if b {
		return 1
	}

	return 0
}
