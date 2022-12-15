package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/kpfaulkner/collablite/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is the client API for CollabLite service.
type Client struct {
	client proto.CollabLiteClient

	// is called when we receive a confirmation from the server.
	objectConfirmationCallback func(*ChangeConfirmation) error

	// stream to gRPC server
	stream proto.CollabLite_ProcessObjectChangesClient

	// used to track local changes (yet to be confirmed by server)
	unconfirmedLocalChanges map[string][]string

	// and associated lock
	unconfirmedLock sync.Mutex
}

func NewClient(serverAddr string) *Client {

	// insecure for now.
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}

	client := proto.NewCollabLiteClient(conn)

	// Make channel size configurable FIXME(kpfaulkner)
	c := &Client{client: client,
		unconfirmedLocalChanges: make(map[string][]string),
	}
	return c
}

// RegisterCallback is called whenever a change for the object being watched is received.
func (c *Client) RegisterCallback(cb func(*ChangeConfirmation) error) error {
	c.objectConfirmationCallback = cb
	return nil
}

// SendChange sends the change to the server for processing
func (c *Client) SendChange(outgoingChange *OutgoingChange) error {
	//t := time.Now()

	// convert to proto struct
	objChange := convertOutgoingChangeToProto(outgoingChange)

	// store change details for comparison with incoming confirmation
	objectProperty := fmt.Sprintf("%s-%s", objChange.ObjectId, objChange.PropertyId)
	c.unconfirmedLock.Lock()
	if ids, ok := c.unconfirmedLocalChanges[objectProperty]; ok {
		ids = append(ids, objChange.UniqueId)
		c.unconfirmedLocalChanges[objectProperty] = ids
	} else {
		c.unconfirmedLocalChanges[objectProperty] = []string{objChange.UniqueId}
	}
	c.unconfirmedLock.Unlock()

	//tt := time.Now()
	fmt.Printf("start send\n")
	if err := c.stream.Send(objChange); err != nil {
		// FIXME(kpfaulkner) shouldn't be fatal...
		log.Errorf("%v.Send(%v) = %v", c.stream, objChange, err)
		return err
	}
	fmt.Printf("end send\n")
	//log.Debugf("stream send took %d ms", time.Since(tt).Milliseconds())

	//log.Debugf("send took: %v", time.Since(t).Milliseconds())

	return nil
}

// Connect creates the stream against the server
func (c *Client) Connect(ctx context.Context) error {

	stream, err := c.client.ProcessObjectChanges(ctx)
	if err != nil {
		log.Errorf("%v.ProcessObjectChanges(_) = _, %v", c.client, err)
		return err
	}

	c.stream = stream
	return nil
}

// RegisterToObject sends a message to the server indicating that the client is listening for changes
// for a particular ObjectID. This only needs to be done once and only if we're passively listening and
// NOT sending data ourselves. If we're sending data we do not need to perform this.
func (c *Client) RegisterToObject(ctx context.Context, objectID string) error {

	// even if not sending changes... send an empty one to indicate what we want to listen to.
	req := OutgoingChange{
		ObjectID:   objectID,
		PropertyID: "",
		Data:       nil,
	}

	if err := c.SendChange(&req); err != nil {
		log.Errorf("failed to send change: %v", err)
		return err
	}

	return nil
}

// Listen will loop for incoming changes from the server. Any changes that are received
// and are NOT discarded (due to modifying the same object/property as a local change)
// will be passed to the callback registered via RegisterCallback
func (c *Client) Listen(ctx context.Context) error {

	var origUniqueIDs []string
	var hasLocalChange bool

	// receive object confirmation
	for {
		objectConfirmation, err := c.stream.Recv()
		if err != nil {
			log.Errorf("%v.Recv() got error %v, want %v", c.stream, err, nil)
			return err
		}

		objectProperty := fmt.Sprintf("%s-%s", objectConfirmation.ObjectId, objectConfirmation.PropertyId)

		// way too much happening in this lock. FIXME(kpfaulkner)
		c.unconfirmedLock.Lock()

		// If this change doesn't match any property change performed locally, then allow it through and
		// call the callback
		if origUniqueIDs, hasLocalChange = c.unconfirmedLocalChanges[objectProperty]; !hasLocalChange {
			// do not have a local change for this object/property combo, so allow this through.
			c.objectConfirmationCallback(convertProtoToChangeConfirmation(objectConfirmation))
		}

		// If we've got here, then we know we have a local change for this object/property combo.
		for i, origUniqueID := range origUniqueIDs {
			// find the specific message.
			if origUniqueID == objectConfirmation.UniqueId {

				// remove from slice. Doing all this in a lock is stoooopid
				c.unconfirmedLocalChanges[objectProperty] = append(origUniqueIDs[:i], origUniqueIDs[i+1:]...)
				if len(c.unconfirmedLocalChanges[objectProperty]) == 0 {
					delete(c.unconfirmedLocalChanges, objectProperty)
				}

				// this is our change... pass it through.
				c.objectConfirmationCallback(convertProtoToChangeConfirmation(objectConfirmation))
				break
			}
		}

		// if we get here it means that we DO have a similar local change that has not been confirmed
		// so it means that we drop this. Our unconfirmed local change is still yet to arrive which
		// means it was generated after...  so this change will get wiped over anyway.
		c.unconfirmedLock.Unlock()
	}

	return nil
}

// convert models to proto structs
func convertOutgoingChangeToProto(outgoingChange *OutgoingChange) *proto.ObjectChange {
	return &proto.ObjectChange{
		ObjectId:   outgoingChange.ObjectID,
		PropertyId: outgoingChange.PropertyID,
		Data:       outgoingChange.Data,

		// add unique id to track local changes.
		UniqueId: uuid.New().String(),
	}
}

func convertProtoToChangeConfirmation(confirmedChange *proto.ObjectConfirmation) *ChangeConfirmation {
	return &ChangeConfirmation{
		ObjectID:   confirmedChange.ObjectId,
		PropertyID: confirmedChange.PropertyId,
		Data:       confirmedChange.Data,
	}
}
