package main

import (
	"strconv"
	"sync/atomic"
)

// ----- Int -----

type expvarInt struct {
	i int64
}

func (v *expvarInt) Value() int64 {
	return atomic.LoadInt64(&v.i)
}

func (v *expvarInt) String() string {
	return strconv.FormatInt(atomic.LoadInt64(&v.i), 10)
}

func (v *expvarInt) Add(i int64) int64 {
	return atomic.AddInt64(&v.i, i)
}

// ----- MaxInt -----

type expvarMaxInt struct {
	i int64
}

func (v *expvarMaxInt) Value() int64 {
	return atomic.LoadInt64(&v.i)
}

func (v *expvarMaxInt) String() string {
	return strconv.FormatInt(atomic.LoadInt64(&v.i), 10)
}

func (v *expvarMaxInt) Update(newMax int64) {
	max := v.Value()
	if max < newMax {
		if !atomic.CompareAndSwapInt64(&v.i, max, newMax) {
			v.Update(newMax)
		}
	}
}
