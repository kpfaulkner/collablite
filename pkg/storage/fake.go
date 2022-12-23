package storage

import (
	"strings"

	"github.com/cockroachdb/pebble"
)

// fakePebble used to mock out set and iter calls for test
type fakePebble struct {

	// fake data set here...
	Objects map[string]Object

	// fake iter
	Iter *pebble.Iterator
}

func NewFakePebble() *fakePebble {
	f := fakePebble{}
	f.Objects = make(map[string]Object)
	return &f
}

func (f *fakePebble) Set(key, value []byte, opts *pebble.WriteOptions) error {

	// make an object if doesn't already exist.
	objectPropertyID := string(key)
	sp := strings.Split(objectPropertyID, ":")
	objectID := sp[0]
	propertyID := sp[1]
	var obj Object
	var ok bool
	if obj, ok = f.Objects[objectID]; !ok {
		f.Objects[objectID] = Object{ObjectID: objectID, Properties: make(map[string][]byte)}
		obj = f.Objects[objectID]
	}
	obj.Properties[propertyID] = value
	f.Objects[objectID] = obj
	return nil
}

func (f *fakePebble) NewIter(o *pebble.IterOptions) *pebble.Iterator {
	return f.Iter
}

type fakePebbleIterator struct {

	// number of items
	ItemCount  int
	currentPos int

	data map[string][]byte

	// keyList purely used so we can iterate in a predictable order
	keyList []string
}

func NewFakePebbleIterator(data map[string][]byte) *fakePebbleIterator {
	f := fakePebbleIterator{}
	f.data = data

	for k, _ := range data {
		f.keyList = append(f.keyList, k)
	}

	return &f
}

func (f *fakePebbleIterator) First() bool {
	return f.currentPos == 0
}

func (f *fakePebbleIterator) Valid() bool {
	return f.currentPos < len(f.data)
}

func (f *fakePebbleIterator) Next() bool {
	f.currentPos++
	return f.currentPos < len(f.data)
}

func (f *fakePebbleIterator) Key() []byte {

	key := f.keyList[f.currentPos]
	return []byte(key)
}

func (f *fakePebbleIterator) ValueAndErr() ([]byte, error) {
	key := f.keyList[f.currentPos]
	return f.data[key], nil
}
