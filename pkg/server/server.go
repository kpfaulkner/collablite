package server

import (
	"context"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/kpfaulkner/collablite/pkg/storage"
	"github.com/kpfaulkner/collablite/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// CollabLiteServer receives gRPC requests from clients and modifies the
// object/data accordingly.
type CollabLiteServer struct {
	proto.UnimplementedCollabLiteServer
	db storage.DB

	processor *Processor

	// channel specifically for DB writing
	// Will probably need to do something for sqlite multithreaded support, but
	// will try this for now.
	dbWriterChannel chan proto.ObjectChange
}

// NewCollabLiteServer create instance of CollabLiteServer with supplied DB client
func NewCollabLiteServer(db storage.DB) *CollabLiteServer {
	cls := CollabLiteServer{}
	cls.db = db

	// can make buffered and solve lots of perf issues... BUT...  means possible loss of data.
	cls.dbWriterChannel = make(chan proto.ObjectChange) // NOT A BUFFERED CHANNEL ON PURPOSE!
	cls.processor = NewProcessor(db, cls.dbWriterChannel)

	go cls.startDBWriter(cls.dbWriterChannel)
	return &cls
}

// startDBWriter reads the change channel, writes to the DB.
func (cls *CollabLiteServer) startDBWriter(changeCh chan proto.ObjectChange) error {
	for change := range changeCh {
		err := cls.db.Add(change.ObjectId, change.PropertyId, change.Data)
		if err != nil {
			log.Errorf("unable to add object %s : %s to db", change.ObjectId, change.PropertyId)
		}
	}
	return nil
}

// ProcessObjectChanges main loop of processing object changes.
// Process is:
//   - Receive change from client
//   - If new objectID, then register client against new Object
//   - If new objectID unregister client from old object
//   - If new objectID start a goroutine to read from client specific channel and send to client over gRPC
//   - Send the change to be processed via channel.
func (cls *CollabLiteServer) ProcessObjectChanges(stream proto.CollabLite_ProcessObjectChangesServer) error {

	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Errorf(codes.DataLoss, "failed to get metadata")
	}
	objId := md["x-object-id"]
	if len(objId) == 0 {
		return status.Errorf(codes.InvalidArgument, "missing 'x-object-id' header")
	}
	if strings.Trim(objId[0], " ") == "" {
		return status.Errorf(codes.InvalidArgument, "empty 'x-object-id' header")
	}

	log.Debugf("Connected with objectID %s in header", objId[0])
	incomingChangeCount := 0
	clientID := uuid.New().String()

	// current* are used to push/receive changes from RPC stream to code that will
	// actually process the changes and return the results.
	var currentObjectID string
	var currentResultChannel chan *proto.ObjectConfirmation
	var currentProcessChannel chan *proto.ObjectChange

	for {

		objChange, err := stream.Recv()
		if err == io.EOF {
			// Change this to attempt reconnect (if server crashed). TODO(kpfaulkner)
			cls.processor.UnregisterClientWithObject(clientID, currentObjectID)
			return nil
		}
		if err != nil {
			// Change this to attempt reconnect (if server crashed). TODO(kpfaulkner)
			cls.processor.UnregisterClientWithObject(clientID, currentObjectID)
			return err
		}
		incomingChangeCount++

		// if not currentObjectID then go get channels for this objectID
		if objChange.ObjectId != currentObjectID {

			// Register this client against the ObjectID.
			// RegisterClientWithObject also spins up a goroutine for processing changes associated with the object
			// IF one does not already exist.
			inChan, outChan, err := cls.processor.RegisterClientWithObject(clientID, objChange.ObjectId)
			if err != nil {
				return err
			}

			// unregister the old client/object
			cls.processor.UnregisterClientWithObject(clientID, currentObjectID)

			currentObjectID = objChange.ObjectId
			currentResultChannel = outChan
			currentProcessChannel = inChan

			// Goroutine is specific to this client. Read the outChan and send to client.
			// Outchan is populated by ProcessObjectChanges
			go func(outChan chan *proto.ObjectConfirmation, clientID string) {
				log.Debugf("starting send goroutine for objectID %s", currentObjectID)
				for msg := range outChan {
					if err := stream.Send(msg); err != nil {
						log.Errorf("unable to send message to client: %v", err)
						return
					}
				}
				log.Debugf("Sending send goroutine for objectID %s", currentObjectID)
			}(currentResultChannel, clientID)
		}

		// send change to be stored and processed.
		// Potential blocking point. FIXME(kpfaulkner) investigate
		currentProcessChannel <- objChange
	}

	return nil
}

// GetObject retrieves an entire object from the DB and returns it
// via gRPC
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
