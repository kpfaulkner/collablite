package server

import (
	"testing"
	"time"

	"github.com/kpfaulkner/collablite/pkg/storage"
	"github.com/kpfaulkner/collablite/proto"
	"github.com/stretchr/testify/assert"
)

func TestRegisterClientWithObject(t *testing.T) {

	// dummy DB... does nothing
	nullDB, _ := storage.NewNullDB()
	ch := make(chan proto.ObjectChange)
	processor := NewProcessor(nullDB, ch)

	changeChannel, confirmationChannel, err := processor.RegisterClientWithObject("client1", "object1")
	assert.Nil(t, err, "Should not have error when registering")

	assert.NotNil(t, changeChannel, "Should have change channel")
	assert.NotNil(t, confirmationChannel, "Should have confirmation channel")
}

func TestRegisterClientWithObjectMultiple(t *testing.T) {

	// dummy DB... does nothing
	nullDB, _ := storage.NewNullDB()
	ch := make(chan proto.ObjectChange)
	processor := NewProcessor(nullDB, ch)

	changeChannel, confirmationChannel, err := processor.RegisterClientWithObject("client1", "object1")
	assert.Nil(t, err, "Should not have error when registering")
	assert.NotNil(t, changeChannel, "Should have change channel")
	assert.NotNil(t, confirmationChannel, "Should have confirmation channel")

	changeChannel2, confirmationChannel2, err := processor.RegisterClientWithObject("client2", "object2")
	assert.Nil(t, err, "Should not have error when registering")
	assert.NotNil(t, changeChannel2, "Should have change channel")
	assert.NotNil(t, confirmationChannel2, "Should have confirmation channel")

	// channels should NOT be the same
	assert.NotEqual(t, changeChannel, changeChannel2, "Change channels should not be the same channel")
	assert.NotEqual(t, confirmationChannel, confirmationChannel2, "Confirmation channels should not be the same channel")
}

func TestRegisterClientWithObjectMultipleForSameObject(t *testing.T) {

	// dummy DB... does nothing
	nullDB, _ := storage.NewNullDB()
	ch := make(chan proto.ObjectChange)
	processor := NewProcessor(nullDB, ch)

	changeChannel, confirmationChannel, err := processor.RegisterClientWithObject("client1", "object1")
	assert.Nil(t, err, "Should not have error when registering")
	assert.NotNil(t, changeChannel, "Should have change channel")
	assert.NotNil(t, confirmationChannel, "Should have confirmation channel")

	changeChannel2, confirmationChannel2, err := processor.RegisterClientWithObject("client2", "object1")
	assert.Nil(t, err, "Should not have error when registering")
	assert.NotNil(t, changeChannel2, "Should have change channel")
	assert.NotNil(t, confirmationChannel2, "Should have confirmation channel")

	// changeChannel SHOULD be the same (using same object)
	assert.Equal(t, changeChannel, changeChannel2, "Change channels should be the same channel")

	// confirmation channels should still be different
	assert.NotEqual(t, confirmationChannel, confirmationChannel2, "Confirmation channels should not be the same channel")
}

func TestRegisterClientWithObjectMultipleForSameObjectSameClient(t *testing.T) {

	// dummy DB... does nothing
	nullDB, _ := storage.NewNullDB()
	ch := make(chan proto.ObjectChange)
	processor := NewProcessor(nullDB, ch)

	changeChannel, confirmationChannel, err := processor.RegisterClientWithObject("client1", "object1")
	assert.Nil(t, err, "Should not have error when registering")
	assert.NotNil(t, changeChannel, "Should have change channel")
	assert.NotNil(t, confirmationChannel, "Should have confirmation channel")

	changeChannel2, confirmationChannel2, err := processor.RegisterClientWithObject("client1", "object1")
	assert.Nil(t, err, "Should not have error when registering")
	assert.NotNil(t, changeChannel2, "Should have change channel")
	assert.NotNil(t, confirmationChannel2, "Should have confirmation channel")

	// changeChannel SHOULD be the same (using same object)
	assert.Equal(t, changeChannel, changeChannel2, "Change channels should be the same channel")

	// confirmation channels should be the same
	assert.Equal(t, confirmationChannel, confirmationChannel2, "Confirmation channels should be the same channel")
}

func TestUnregisterClientWithObjectSingle(t *testing.T) {

	// dummy DB... does nothing
	nullDB, _ := storage.NewNullDB()
	ch := make(chan proto.ObjectChange)
	processor := NewProcessor(nullDB, ch)

	changeChannel, confirmationChannel, err := processor.RegisterClientWithObject("client1", "object1")
	assert.Nil(t, err, "Should not have error when registering")
	assert.NotNil(t, changeChannel, "Should have change channel")
	assert.NotNil(t, confirmationChannel, "Should have confirmation channel")

	err = processor.UnregisterClientWithObject("client1", "object1")
	assert.Nil(t, err, "Should not have error when registering")

}

func TestUnregisterClientWithObjectButNotRegistered(t *testing.T) {

	// dummy DB... does nothing
	nullDB, _ := storage.NewNullDB()
	ch := make(chan proto.ObjectChange)
	processor := NewProcessor(nullDB, ch)

	err := processor.UnregisterClientWithObject("client1", "object1")
	assert.NotNil(t, err, "Should throw error if not registered")

}

func TestProcessObjectChanges(t *testing.T) {
	db, _ := NewFakeDB()
	ch := make(chan proto.ObjectChange)
	processor := NewProcessor(db, ch)

	changeChannel, confirmationChannel, err := processor.RegisterClientWithObject("client1", "object1")
	assert.Nil(t, err, "Should not have error when registering")
	assert.NotNil(t, changeChannel, "Should have change channel")
	assert.NotNil(t, confirmationChannel, "Should have confirmation channel")
	assert.EqualValues(t, 0, len(confirmationChannel), "Should have no confirmation messages")

	testChange := proto.ObjectChange{
		ObjectId:   "object1",
		PropertyId: "prop1",
		Data:       []byte("prop1value"),
	}
	changeChannel <- &testChange

	//err = processor.ProcessObjectChanges("object1", changeChannel)
	//assert.Nil(t, err, "Should not have error when sending object changes")
	//assert.EqualValues(t, 1, len(confirmationChannel), "Should have one confirmation message")

	time.Sleep(2 * time.Second) // hack.. timing sucketh.

	obj, err := db.Get("object1")
	assert.Nil(t, err, "Should not have error when getting object")
	assert.EqualValues(t, "prop1value", string(obj.Properties["prop1"]), "Should have correct property value")
}
