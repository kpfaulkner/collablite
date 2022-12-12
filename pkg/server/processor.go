package server

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/kpfaulkner/collablite/pkg/storage"
	"github.com/kpfaulkner/collablite/proto"
)

// ObjectIDChannels holds input channel (original objectChange) and slice of
// output channels, one per client that is working on that object
type ObjectIDChannels struct {
	inChannel chan *proto.ObjectChange

	// map of unique id (related to client, somehow) and outgoing channel with results
	outChannels map[string]chan *proto.ObjectConfirmation
}

// Processor takes the objectChange (from channel), stores to the DB and return objectConfirmation via channel
type Processor struct {
	objectChannelLock sync.Mutex

	// map of object id to channels used for input and output.
	objectChannels map[string]*ObjectIDChannels

	// DB for storing objects
	db storage.DB
}

func NewProcessor(db storage.DB) *Processor {
	p := Processor{}
	p.objectChannels = make(map[string]*ObjectIDChannels)
	p.db = db
	return &p
}

// RegisterClientWithObject registers a clientID and objectID with the processor.
// This is used when an object is processed... it will contain a list of clients/channels
// that need to get the results of the processing of a given object.
// Will return inChan (specific for object) and results channel (specific for object AND clientid) to caller.
func (p *Processor) RegisterClientWithObject(clientID string, objectID string) (chan *proto.ObjectChange, chan *proto.ObjectConfirmation, error) {
	p.objectChannelLock.Lock()
	defer p.objectChannelLock.Unlock()

	var oc *ObjectIDChannels
	var ok bool
	if oc, ok = p.objectChannels[objectID]; !ok {
		oc = &ObjectIDChannels{}
		oc.inChannel = make(chan *proto.ObjectChange, 1000) // FIXME(kpfaulkner) configure 1000
		oc.outChannels = make(map[string]chan *proto.ObjectConfirmation)
		p.objectChannels[objectID] = oc

		fmt.Printf("creating process goroutine %s\n", objectID)
		// this is a new object being processed, so start a go routine to process it.
		go p.ProcessObjectChanges(objectID)
	}

	var clientObjectChannel chan *proto.ObjectConfirmation
	if clientObjectChannel, ok = oc.outChannels[clientID]; !ok {
		// create an out channel specific for the client. This will be used to send results.
		oc.outChannels[clientID] = make(chan *proto.ObjectConfirmation, 1000) // FIXME(kpfaulkner) configure 1000
		clientObjectChannel = oc.outChannels[clientID]
	}

	return oc.inChannel, clientObjectChannel, nil
}

func (p *Processor) UnregisterClientWithObject(clientID string, objectID string) error {
	p.objectChannelLock.Lock()
	defer p.objectChannelLock.Unlock()

	if oc, ok := p.objectChannels[objectID]; ok {
		if clientOutChan, ok := oc.outChannels[clientID]; ok {
			close(clientOutChan)
			delete(oc.outChannels, clientID)
		}

		// if no clients listening to channel... then close it. This will also stop the goroutine processing it.
		if len(oc.outChannels) == 0 {
			close(oc.inChannel)
			delete(p.objectChannels, objectID)
		}
	}

	return nil
}

func (p *Processor) getInChanForObjectID(objectID string) (chan *proto.ObjectChange, error) {
	p.objectChannelLock.Lock()
	defer p.objectChannelLock.Unlock()

	if oc, ok := p.objectChannels[objectID]; ok {
		return oc.inChannel, nil
	}

	return nil, fmt.Errorf("objectID not found")
}

func (p *Processor) ProcessObjectChanges(objectID string) error {

	// get in chan
	inChan, err := p.getInChanForObjectID(objectID)
	if err != nil {
		fmt.Printf("Unable to process objectID %s\n", objectID)
		return err
	}

	for objChange := range inChan {

		//fmt.Printf("processing %v\n", objChange)
		fmt.Printf("no goroutines %d\n", runtime.NumGoroutine())
		// do stuff.... then return result.
		err := p.db.Add(objChange.ObjectId, objChange.PropertyId, objChange.Data)
		if err != nil {
			fmt.Printf("Unable to add to DB for objectID %s\n", objectID)
			return err
		}
		res := proto.ObjectConfirmation{}
		res.ObjectId = objChange.ObjectId
		res.PropertyId = objChange.PropertyId
		res.UniqueId = objChange.UniqueId
		res.Data = objChange.Data

		// loop through all out channels and send result.
		// this REALLY sucks holding the lock for this long, but will do for now.
		// FIXME(kpfaulkner) MUST optimise this!
		p.objectChannelLock.Lock()

		// do a check for the objectID since the objects/clients might be nuked
		if chans, ok := p.objectChannels[objectID]; ok {
			for _, oc := range chans.outChannels {
				oc <- &res
			}
		}
		p.objectChannelLock.Unlock()
	}
	return nil
}
