package client

import (
	"context"
	"errors"
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

	// function used to convert from client structure to our internal InternalObject type
	convertToObject func(objectID string, exitingObject *InternalObject, clientObject any) (*InternalObject, error)

	// function used to convert from out internal InternalObject type to the clients structure
	convertFromObject func(object *InternalObject) error

	// stream to gRPC server
	stream proto.CollabLite_ProcessObjectChangesClient

	// used to track local changes (yet to be confirmed by server)
	unconfirmedLocalChanges map[string]int

	// and associated lock
	unconfirmedLock sync.Mutex

	// clientID used to help identify traffic from this client
	clientID string

	// number of conflicts, purely for stats collecting.
	numConflicts int

	// InternalObject the client is dealing with..
	object *InternalObject

	// Lock for the above.
	objectLock sync.Mutex
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
		unconfirmedLocalChanges: make(map[string]int),
		clientID:                uuid.New().String(),
	}
	return c
}

func (c *Client) RegisterConverters(convertFromObject func(object *InternalObject) error,
	convertToObject func(objectID string, exitingObject *InternalObject, clientObject any) (*InternalObject, error)) error {
	c.convertFromObject = convertFromObject
	c.convertToObject = convertToObject
	return nil
}

// SendObject takes the users object, converts to what the server expects and sends it.
// Steps:
//   - Update our internal object (c.object) with the new clientObject. This may just be updating a single property
//     Or populating the entire object.
//   - Loop through the dirty properties in the internal object and send to server.
func (c *Client) SendObject(objectID string, clientObject any) error {
	var err error
	c.objectLock.Lock()
	c.object, err = c.convertToObject(objectID, c.object, clientObject)
	c.objectLock.Unlock()
	if err != nil {
		log.Errorf("failed to convert object: %v", err)
		return err
	}

	for k, v := range c.object.Properties {
		if v.Dirty {
			outgoingChange := &OutgoingChange{
				ObjectID:   objectID,
				PropertyID: k,
				Data:       v.Data,
			}
			err := c.sendChange(outgoingChange)
			if err != nil {
				log.Errorf("failed to send change: %v", err)
				return err
			}

			// no longer dirty.
			v.Dirty = false
			c.objectLock.Lock()
			c.object.Properties[k] = v
			c.objectLock.Unlock()
		}
	}

	return nil
}

// SendChange sends the change to the server for processing
func (c *Client) sendChange(outgoingChange *OutgoingChange) error {
	// convert to proto struct
	objChange := convertOutgoingChangeToProto(outgoingChange, c.clientID)

	// store change details for comparison with incoming confirmation
	objectProperty := fmt.Sprintf("%s-%s", objChange.ObjectId, objChange.PropertyId)
	c.unconfirmedLock.Lock()
	if _, ok := c.unconfirmedLocalChanges[objectProperty]; ok {
		c.unconfirmedLocalChanges[objectProperty] = 1
	} else {
		c.unconfirmedLocalChanges[objectProperty] = c.unconfirmedLocalChanges[objectProperty] + 1
	}
	c.unconfirmedLock.Unlock()

	if err := c.stream.Send(objChange); err != nil {
		log.Errorf("%v.Send(%v) = %v", c.stream, objChange, err)
		return err
	}

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

// GetObject returns the entire object from the server.
// Used for initial loading etc.
func (c *Client) GetObject(objectID string) ([]ChangeConfirmation, error) {
	resp, err := c.client.GetObject(context.Background(), &proto.GetRequest{
		ObjectId: objectID,
	})

	if err != nil {
		log.Errorf("failed to get object: %v", err)
		return nil, err
	}

	changes := make([]ChangeConfirmation, len(resp.Properties))
	i := 0
	for k, v := range resp.Properties {
		changes[i] = ChangeConfirmation{
			ObjectID:   resp.ObjectId,
			PropertyID: k,
			Data:       v,
		}
		i++
	}

	return changes, nil
}

// RegisterToObject sends a message to the server indicating that the client is listening for changes
// for a particular ObjectID. This only needs to be done when the client is switching between objects.
func (c *Client) RegisterToObject(ctx context.Context, objectID string) error {

	// even if not sending changes... send an empty one to indicate what we want to listen to.
	req := OutgoingChange{
		ObjectID:   objectID,
		PropertyID: "", // empty property used to register interest of object with server.
		Data:       nil,
	}

	// brand new internal object.
	c.object = &InternalObject{
		ObjectID:   objectID,
		Properties: make(map[string]Property),
	}

	if err := c.sendChange(&req); err != nil {
		log.Errorf("failed to send change: %v", err)
		return err
	}

	return nil
}

// Listen will loop for incoming changes from the server. Any changes that are received
// and are NOT discarded (due to modifying the same object/property as a local change)
// will be passed to the callback registered via RegisterCallback
func (c *Client) Listen(ctx context.Context) error {

	//var origUniqueIDs []string
	var hasLocalChange bool

	count := 0
	// receive object confirmation
	for {
		objectConfirmation, err := c.stream.Recv()
		if err != nil {
			log.Errorf("%v.Recv() got error %v, want %v", c.stream, err, nil)
			return err
		}
		count++
		if count%100 == 0 {
			log.Debugf("Received %d", count)
		}
		objectProperty := fmt.Sprintf("%s-%s", objectConfirmation.ObjectId, objectConfirmation.PropertyId)

		// way too much happening in this lock. FIXME(kpfaulkner)
		c.unconfirmedLock.Lock()

		confirmedLocalChange := false

		// If this change doesn't match any property change performed locally, then allow it through and
		// call the callback
		if _, hasLocalChange = c.unconfirmedLocalChanges[objectProperty]; !hasLocalChange {
			// do not have a local change for this object/property combo, so allow this through.

			err := c.convertAndExecuteCallback(objectConfirmation)
			if err != nil {
				return err
			}
		} else {
			// check if change is from this client. If so, modify unconfirmedLocalChanges
			if objectConfirmation.UniqueId == c.clientID {
				if c.unconfirmedLocalChanges[objectProperty] > 0 {

					// does have local changes.. decrement count of changes.
					c.unconfirmedLocalChanges[objectProperty]--
				}

				// if no more left, then delete from map
				if c.unconfirmedLocalChanges[objectProperty] == 0 {
					delete(c.unconfirmedLocalChanges, objectProperty)
				}
				confirmedLocalChange = true
				err := c.convertAndExecuteCallback(objectConfirmation)
				if err != nil {
					return err
				}
			}
		}

		if hasLocalChange && !confirmedLocalChange {
			c.numConflicts++
		}
		// if we get here it means that we DO have a similar local change that has not been confirmed
		// so it means that we drop this. Our unconfirmed local change is still yet to arrive which
		// means it was generated after...  so this change will get wiped over anyway.
		c.unconfirmedLock.Unlock()
	}

	return nil
}

func (c *Client) convertAndExecuteCallback(objectConfirmation *proto.ObjectConfirmation) error {
	confirmation := convertProtoToChangeConfirmation(objectConfirmation)

	if objectConfirmation.ObjectId != c.object.ObjectID {
		log.Errorf("incorrect object ID returned. Expected %s, got %s", c.object.ObjectID, objectConfirmation.ObjectId)
		return errors.New("incorrect object ID returned")
	}

	if confirmation.PropertyID != "" {

		c.objectLock.Lock()
		// indicate its been updated from the server.
		c.object.Properties[confirmation.PropertyID] = Property{Data: confirmation.Data, Dirty: false, Updated: true}

		err := c.convertFromObject(c.object)
		c.objectLock.Unlock()
		if err != nil {
			log.Errorf("unable to convert incoming change to object: %v", err)
			return err
		}
	}
	return nil
}

// GetConflictsCount returns number of conflicts the client has recorded
func (c *Client) GetConflictsCount() int {
	return c.numConflicts
}

// GetChangeCount lists the number of unconfirmed changes for client
func (c *Client) GetChangeCount() int {

	c.unconfirmedLock.Lock()
	count := 0
	for _, v := range c.unconfirmedLocalChanges {
		count += v // FIXME int change
	}
	c.unconfirmedLock.Unlock()
	return count
}

// convert models to proto structs
func convertOutgoingChangeToProto(outgoingChange *OutgoingChange, clientID string) *proto.ObjectChange {
	return &proto.ObjectChange{
		ObjectId:   outgoingChange.ObjectID,
		PropertyId: outgoingChange.PropertyID,
		Data:       outgoingChange.Data,

		// add unique id to track local changes.
		//UniqueId: fmt.Sprintf("%s-%s", clientID[:8], uuid.New().String()),
		UniqueId: clientID,
	}
}

func convertProtoToChangeConfirmation(confirmedChange *proto.ObjectConfirmation) *ChangeConfirmation {
	return &ChangeConfirmation{
		ObjectID:   confirmedChange.ObjectId,
		PropertyID: confirmedChange.PropertyId,
		Data:       confirmedChange.Data,
	}
}
