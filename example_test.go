package readerwriter

import (
	"reflect"
	"sort"
	"strings"
)

func Example() {
	// init
	readerInit, writerInit := make([]string, 0), make([]string, 0)
	searchIndex := New(readerInit, writerInit)

	// empty read
	emptyReadDone := make(chan struct{}) // to create a reliable example
	go func() {
		r := searchIndex.Reader()
		v := r.Get()
		if len(v) != 0 {
			panic("unreachable")
		}
		r.Done()
		close(emptyReadDone)
	}()
	<-emptyReadDone

	// add some values
	searchIndex.Set(append(searchIndex.Get(), "foo", "bar", "foobar"))

	// read after update
	readAfterUpdate := make(chan struct{})
	readAfterUpdateDone := make(chan struct{})
	go func() {
		<-readAfterUpdate
		r := searchIndex.Reader()
		v := r.Get()
		_, found := sort.Find(len(v), func(i int) int {
			return strings.Compare("bar", v[i])
		})
		if !found {
			panic("unreachable")
		}
		r.Done()
		close(readAfterUpdateDone)
	}()

	// decide we added enough values
	// sort the index, swap and copy
	sort.Strings(searchIndex.Get())
	searchIndex.Swap()
	close(readAfterUpdate) // now the new values are visible
	newReader := searchIndex.Reader()
	searchIndex.Set(append(searchIndex.Get()[:0], newReader.Get()...))
	if !reflect.DeepEqual(searchIndex.Get(), newReader.Get()) {
		panic("unreachable")
	}
	newReader.Done()

	<-readAfterUpdateDone

	// and repeat ...

	// Output:
}
