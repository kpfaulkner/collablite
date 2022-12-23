package storage

import "github.com/cockroachdb/pebble"

// fakePebble used to mock out set and iter calls for test
type fakePebble struct {

	// fake data set here...
	Data map[string][]byte
}

func NewFakePebble() *fakePebble {
	f := fakePebble{}
	f.Data = make(map[string][]byte)
	return &f
}

func (f *fakePebble) Set(key, value []byte, opts *pebble.WriteOptions) error {

	// convert to string...  cant use a byte array as a map key
	f.Data[string(key)] = value

	return nil
}

func (f *fakePebble) NewIter(o *pebble.IterOptions) *pebble.Iterator {
	return nil
}
