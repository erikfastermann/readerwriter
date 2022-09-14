// Package readerwriter implements a concurrency primitive
// for many lock-free Reader's and one Writer.
//
// The idea is that the Writer does some work and at some
// point calls Swap. New Reader's can see the new writes
// immediately. The Writer automatically waits until all
// old readers are done. Then Swap returns. Thereafter
// the caller usually copies all the changes from the
// new reader-part (previously writer-part) to the new
// writer-part (previously reader-part). Now the Writer
// can apply new changes and the cycle repeats.
//
// Check the documentation of the types/methods for more
// information about the correct usage.
package readerwriter

import (
	"sync"
	"sync/atomic"
)

const (
	messageUsageOldReaderDetected  = "usage of an old reader detected"
	messageMultipleWritersDetected = "multiple writers detected"
)

type current[T any] struct {
	sync.RWMutex
	v T
}

// Writer represents the core abstraction of this package.
//
// In general all methods are not threadsafe unless specified
// otherwise.
type Writer[T any] struct {
	current atomic.Pointer[current[T]]

	unsyncWriterCheck sync.Mutex
	writerValue       T
}

// New returns a new Writer with the specified
// reader and writer parts.
func New[T any](reader, writer T) *Writer[T] {
	w := &Writer[T]{
		writerValue: writer,
	}
	w.current.Store(&current[T]{v: reader})
	return w
}

// Get returns the current writer portion. The returned value
// should only be used until calling Swap.
func (w *Writer[T]) Get() T {
	if !w.unsyncWriterCheck.TryLock() {
		panic(messageMultipleWritersDetected)
	}
	defer w.unsyncWriterCheck.Unlock()
	return w.writerValue
}

// Set sets the current writer portion.
func (w *Writer[T]) Set(v T) (previous T) {
	if !w.unsyncWriterCheck.TryLock() {
		panic(messageMultipleWritersDetected)
	}
	defer w.unsyncWriterCheck.Unlock()
	previous = w.writerValue
	w.writerValue = v
	return previous
}

// Reader represents the reader portion. A Reader is
// not threadsafe.
type Reader[T any] struct {
	mu   *sync.RWMutex
	done bool
	v    T
}

// Reader returns the current reader portion. This operation
// is lock-free.
//
// The returned Reader is valid until Reader.Done is called.
// Users should not hold onto the Reader for too long
// to not stall the Writer unnecessarily.
//
// Calling Reader is threadsafe.
func (w *Writer[T]) Reader() *Reader[T] {
	for {
		current := w.current.Load()
		if !current.TryRLock() {
			// the writer is waiting for the readers to perform the swap,
			// which means we should load again.
			continue
		}
		afterRLock := w.current.Load()
		if current != afterRLock {
			// in case the writer swaps and unlocks
			// between our load and lock attempt.
			current.RUnlock()
			continue
		}
		return &Reader[T]{mu: &current.RWMutex, v: current.v}
	}
}

// Get returns the value of the current Reader.
//
// Usually the caller should not modify the
// returned value or use it after calling Done.
func (r *Reader[T]) Get() T {
	if r.done {
		panic(messageUsageOldReaderDetected)
	}
	return r.v
}

// Done must be called when finished reading,
// so the Writer can make progress.
func (r *Reader[T]) Done() {
	if r.done {
		panic(messageUsageOldReaderDetected)
	}
	r.done = true
	r.mu.RUnlock()
}

// Swap exchanges the reader and writer portion and waits for
// all old Reader's to complete.
//
// Usually the accumulated writes are copied by the caller
// to the new writer portion after this method returns.
func (w *Writer[T]) Swap() {
	if !w.unsyncWriterCheck.TryLock() {
		panic(messageMultipleWritersDetected)
	}
	defer w.unsyncWriterCheck.Unlock()

	// new readers can use the new value immediately,
	// but the next writer has to wait until everybody is done reading.
	oldReader := w.current.Swap(&current[T]{v: w.writerValue})
	oldReader.Lock()
	_ = "noop" // silence static analysis
	oldReader.Unlock()
	w.writerValue = oldReader.v

	// do stuff after this ...
}
