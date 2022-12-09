package server

import (
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kpfaulkner/collablite/pkg/storage"
	"github.com/kpfaulkner/collablite/proto"
)

type ObjectIDChannelsx struct {
	processChannel chan *proto.DocChange
	resultChannel  chan *proto.DocConfirmation
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

// ProcessDocumentChanges main loop of processing doc changes.
// Process is:
//   - Receive change from client
//   - Get process channel associated with objectID
//   - Get result channel associated with object ID
//   - Send change to process channel
//   - Read result from result channel
//   - Send result to client
func (cls *CollabLiteServer) ProcessDocumentChanges(stream proto.CollabLite_ProcessDocumentChangesServer) error {

	// clientID... need to figure out what to do here FIXME(kpfaulkner)
	u, _ := uuid.NewUUID()
	clientID := u.String()

	// current* are used to push/receive changes from RPC stream to code that will
	// actually process the changes and return the results.
	var currentObjectID string
	var currentResultChannel chan *proto.DocConfirmation
	var currentProcessChannel chan *proto.DocChange
	for {
		docChange, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// if not currentObjectID then go get channels for this objectID
		if docChange.ObjectId != currentObjectID {
			inChan, outChan, err := cls.processor.RegisterClientWithDoc(clientID, docChange.ObjectId)
			if err != nil {
				return err
			}
			currentObjectID = docChange.ObjectId
			currentResultChannel = outChan
			currentProcessChannel = inChan
		}

		// send change to be stored and processed.
		currentProcessChannel <- docChange

		// cant just have reading of currentResultChannel in separate goroutine since that channel CAN
		// change if the user switches which document they're on.
		// For now, just loop with a select.
		done := false
		for !done {
			select {
			case msg := <-currentResultChannel:
				if err := stream.Send(msg); err != nil {
					return err
				}

			// FIXME(kpfaulkner) make 100ms configurable...
			case <-time.After(100 * time.Millisecond):
				// do nothing and break out for reading results loop.
				done = true
			}
		}
	}

	return nil
}

// getChannelsForObjectID returns the process and result channels for the objectID
func getChannelsForObjectID(id string) (chan *proto.DocChange, chan *proto.DocConfirmation) {

	channelLock.Lock()
	defer channelLock.Unlock()

	//var channels ObjectIDChannels
	if channels, ok := objectIDToChannels[id]; !ok {
		//channels.processChannel = make(chan *proto.DocChange, 1000) // FIXME(kpfaulkner) config the 1000
		//channels.resultChannel = make(chan *proto.DocConfirmation, 1000)
		objectIDToChannels[id] = channels
	}

	//return channels.processChannel, channels.resultChannel
	return nil, nil
}
