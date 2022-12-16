package server

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/kpfaulkner/collablite/pkg/storage"
	"github.com/kpfaulkner/collablite/proto"
	log "github.com/sirupsen/logrus"
)

type ObjectIDChannelsx struct {
	processChannel chan *proto.ObjectChange
	resultChannel  chan *proto.ObjectConfirmation
}

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
//   - If new objectID, then register client against new Object
//   - If new objectID unregister client from old object
//   - If new objectID start a goroutine to read from client specific channel and send to client over gRPC
//   - Send the change to be processed via channel.
func (cls *CollabLiteServer) ProcessObjectChanges(stream proto.CollabLite_ProcessObjectChangesServer) error {

	count := 0
	// clientID... need to figure out what to do here FIXME(kpfaulkner)
	clientID := uuid.New().String()

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
		count++

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

			// Goroutine is specific to this client. Read the outChan and send to client.
			// Outchan is populated by ProcessObjectChanges
			go func(outChan chan *proto.ObjectConfirmation, clientID string) {
				log.Debugf("starting send goroutine for objectID %s", currentObjectID)
				c := 0
				for msg := range outChan {
					if err := stream.Send(msg); err != nil {
						log.Errorf("unable to send message to client: %v", err)
						// If error then we cannot update the client. Will disconnect client and force them to reconnect.
						// In that reconnect process they'll get the entire document and be up to date.
						return
					}
					c++
					if c%100 == 0 {
						log.Debugf("Sending client %s count %d", clientID, c)
					}
				}
				log.Debugf("Sending send goroutine for objectID %s", currentObjectID)
			}(currentResultChannel, clientID)
		}

		// send change to be stored and processed.
		// Potential blocking point. FIXME(kpfaulkner) investigate
		currentProcessChannel <- objChange

		if count%100 == 0 {
			log.Debugf("Received from client %s : count %d", clientID, count)
		}
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
