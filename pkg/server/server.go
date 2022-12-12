package server

import (
	"context"
	"io"
	"sync"

	"github.com/google/uuid"
	"github.com/kpfaulkner/collablite/pkg/storage"
	"github.com/kpfaulkner/collablite/proto"
	log "github.com/sirupsen/logrus"
)

type ObjectIDChannelsx struct {
	processChannel chan *proto.ObjectChange
	resultChannel  chan *proto.ObjectConfirmation
}

// used to lock access to objectID -> channels map.
var channelLock sync.Mutex
var objectIDToChannels = make(map[string]ObjectIDChannels)

// CollabLiteServer receives gRPC requests from clients and modifies the
// object/data accordingly.
type CollabLiteServer struct {
	proto.UnimplementedCollabLiteServer
	db storage.DB

	processor *Processor
}

func NewCollabLiteServer(db storage.DB) *CollabLiteServer {
	cls := CollabLiteServer{}
	cls.db = db
	cls.processor = NewProcessor(db)
	return &cls
}

// ProcessObjectChanges main loop of processing object changes.
// Process is:
//   - Receive change from client
//   - Get process channel associated with objectID
//   - Get result channel associated with object ID
//   - Send change to process channel
//   - Read result from result channel
//   - Send result to client
func (cls *CollabLiteServer) ProcessObjectChanges(stream proto.CollabLite_ProcessObjectChangesServer) error {

	// clientID... need to figure out what to do here FIXME(kpfaulkner)
	u, _ := uuid.NewUUID()
	clientID := u.String()

	// current* are used to push/receive changes from RPC stream to code that will
	// actually process the changes and return the results.
	var currentObjectID string
	var currentResultChannel chan *proto.ObjectConfirmation
	var currentProcessChannel chan *proto.ObjectChange

	for {
		objChange, err := stream.Recv()
		if err == io.EOF {
			cls.processor.UnregisterClientWithObject(clientID, currentObjectID)
			return nil
		}
		if err != nil {
			cls.processor.UnregisterClientWithObject(clientID, currentObjectID)
			return err
		}

		// if not currentObjectID then go get channels for this objectID
		if objChange.ObjectId != currentObjectID {
			inChan, outChan, err := cls.processor.RegisterClientWithObject(clientID, objChange.ObjectId)
			if err != nil {
				return err
			}

			// unregister
			cls.processor.UnregisterClientWithObject(clientID, currentObjectID)

			currentObjectID = objChange.ObjectId
			currentResultChannel = outChan
			currentProcessChannel = inChan

			// send message to client
			go func(outChan chan *proto.ObjectConfirmation) {
				log.Debugf("starting send goroutine for objectID %s", currentObjectID)
				for msg := range outChan {
					if err := stream.Send(msg); err != nil {
						log.Errorf("unable to send message to client: %v", err)
						// If error then we cannot update the client. Will disconnect client and force them to reconnect.
						// In that reconnect process they'll get the entire document and be up to date.
						return
					}
				}
				log.Debugf("ending send goroutine for objectID %s", currentObjectID)
			}(currentResultChannel)
		}

		// send change to be stored and processed.
		currentProcessChannel <- objChange

	}

	return nil
}

func (cls *CollabLiteServer) GetObject(ctx context.Context, req *proto.GetRequest) (*proto.GetResponse, error) {

	obj, err := cls.db.Get(req.ObjectId)
	if err != nil {
		return nil, err
	}

	resp := &proto.GetResponse{}
	resp.ObjectId = obj.ObjectID
	resp.Properties = obj.Properties
	return resp, nil
}
