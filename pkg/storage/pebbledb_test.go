package storage

import (
	"fmt"
	"testing"

	log "github.com/sirupsen/logrus"
)

func setupTest(tb testing.TB) func(tb testing.TB) {
	log.Println("setup test")

	return func(tb testing.TB) {
		log.Println("teardown test")
	}
}

func assertEqual(t *testing.T, a interface{}, b interface{}, message string) {
	if a == b {
		return
	}
	if len(message) == 0 {
		message = fmt.Sprintf("%v != %v", a, b)
	}
	t.Error(message)
}

func TestAdd(t *testing.T) {
	pb := NewFakePebble()
	pdb, err := NewPebbleDB(pb)
	if err != nil {
		t.Errorf(err.Error())
	}

	pdb.Add("test", "prop1", []byte("prop1"))
	pdb.Add("test", "prop2", []byte("prop2"))

	if len(pb.Objects) != 1 {
		t.Errorf("Expected 1 object map, got %d", len(pb.Objects))
	}

	if len(pb.Objects["test"].Properties) != 2 {
		t.Errorf("Expected 2 properties, got %d", len(pb.Objects["test"].Properties))
	}
}

/*
func TestGet(t *testing.T) {
	assert := assert.New(t)

	dummyData := map[string][]byte{"test:prop1": []byte("prop1"), "test:prop2": []byte("prop2")}
	fpi := NewFakePebbleIterator(dummyData)
	pb := NewFakePebble()
	pb.Iter = fpi

	pdb, err := NewPebbleDB(pb)
	if err != nil {
		t.Errorf(err.Error())
	}

	obj, err := pdb.Get("test")

	assert.NotNil(obj, "object is nil")
	assert.EqualValues(2, len(obj.Properties), "expected 2 properties")
}
*/
