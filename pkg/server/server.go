package server

import (
	"io"
	"sync"
	"time"

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
}

func NewCollabLiteServer(db storage.DB) *CollabLiteServer {
	cls := CollabLiteServer{}
	cls.db = db
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

	// current* are used to push/receive changes from RPC stream to code that will
	// actually process the changes and return the results.
	var currentObjectID string
	var currentProcessChannel chan *proto.DocChange
	var currentResultChannel chan *proto.DocConfirmation

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
			currentObjectID = docChange.ObjectId
			currentProcessChannel, currentResultChannel = getChannelsForObjectID(docChange.ObjectId)
		}

		// send change to be stored and processed.
		currentProcessChannel <- docChange

		// cant just have reading of currentResultChannel in separate goroutine since that channel CAN
		// change if the user switches which document they're on.
		// For now, just loop with a select.
		for {
			select {
			case msg := <-currentResultChannel:
				if err := stream.Send(msg); err != nil {
					return err
				}

			// FIXME(kpfaulkner) make 100ms configurable...
			case <-time.After(100 * time.Millisecond):
				// do nothing and break out for reading results loop.
				break
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
