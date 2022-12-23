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

func TestGet(t *testing.T) {
	pb := NewFakePebble()
	pdb, err := NewPebbleDB(pb)
	if err != nil {
		t.Errorf(err.Error())
	}

	pdb.Add("test", "prop1", []byte("prop1"))
	pdb.Add("test", "prop2", []byte("prop2"))

	if len(pb.Data) != 2 {
		t.Errorf("Expected 2 items in map, got %d", len(pb.Data))
	}

}
