package readerwriter

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

// to prevent possible optimizations
var TestReaderWriterValue atomic.Int64

func TestReaderWriter(t *testing.T) {
	// only really useful with -race flag

	w := New([]int64{42}, []int64{-1})

	done := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU()*2; i++ {
		wg.Add(1)
		go func() {
			for i := 0; ; i++ {
				select {
				case <-done:
					t.Log("rounds:", i)
					wg.Done()
					return
				default:
					r := w.Reader()
					TestReaderWriterValue.Store(r.Get()[0])
					r.Done()
				}
			}
		}()
	}

	for i := 0; i < 100; i++ {
		w.Set(append(w.Get(), int64(i)))

		w.Swap()
		r := w.Reader()
		w.Set(append(w.Get()[:0], r.Get()...))
		r.Done()
	}
	close(done)
	wg.Wait()

	for i := int64(-1); i < 100; i++ {
		w.Get()[i+1] = i
		r := w.Reader()
		r.Get()[i+1] = i
		r.Done()
	}
}
