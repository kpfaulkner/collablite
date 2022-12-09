package server

import (
	"fmt"
	"sync"

	"github.com/kpfaulkner/collablite/pkg/storage"
	"github.com/kpfaulkner/collablite/proto"
)

// ObjectIDChannels holds input channel (original docChange) and slice of
// output channels, one per client that is working on that document.
type ObjectIDChannels struct {
	inChannel chan *proto.DocChange

	// map of unique id (related to client, somehow) and outgoing channel with results
	outChannels map[string]chan *proto.DocConfirmation
}

// Processor takes the docChange (from channel), stores to the DB and return docConfirmation via channel
type Processor struct {
	objectChannelLock sync.Mutex

	// map of object/document id to channels used for input and output.
	objectChannels map[string]*ObjectIDChannels

	// DB for storing docs.
	db storage.DB
}

func NewProcessor(db storage.DB) *Processor {
	p := Processor{}
	p.objectChannels = make(map[string]*ObjectIDChannels)
	p.db = db
	return &p
}

// RegisterClientWithDoc registers a clientID and objectID with the processor.
// This is used when a document is processed... it will contain a list of clients/channels
// that need to get the results of the processing of a given document/object.
// Will return inChan (specific for document) and results channel (specific for document AND clientid) to caller.
func (p *Processor) RegisterClientWithDoc(clientID string, objectID string) (chan *proto.DocChange, chan *proto.DocConfirmation, error) {
	p.objectChannelLock.Lock()
	defer p.objectChannelLock.Unlock()

	var oc *ObjectIDChannels
	var ok bool
	if oc, ok = p.objectChannels[objectID]; !ok {
		oc = &ObjectIDChannels{}
		oc.inChannel = make(chan *proto.DocChange, 1000) // FIXME(kpfaulkner) configure 1000
		oc.outChannels = make(map[string]chan *proto.DocConfirmation)
		p.objectChannels[objectID] = oc

		// this is a new document being processed, so start a go routine to process it.
		go p.ProcessDocChanges(objectID)
	}

	var clientObjectChannel chan *proto.DocConfirmation
	if clientObjectChannel, ok = oc.outChannels[clientID]; !ok {
		// create an out channel specific for the client. This will be used to send results.
		oc.outChannels[clientID] = make(chan *proto.DocConfirmation, 1000) // FIXME(kpfaulkner) configure 1000
		clientObjectChannel = oc.outChannels[clientID]
	}

	return oc.inChannel, clientObjectChannel, nil
}

func (p *Processor) UnregisterClientWithDoc(clientID string, objectID string) error {
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

func (p *Processor) getInChanForObjectID(objectID string) (chan *proto.DocChange, error) {
	p.objectChannelLock.Lock()
	defer p.objectChannelLock.Unlock()

	if oc, ok := p.objectChannels[objectID]; ok {
		return oc.inChannel, nil
	}

	return nil, fmt.Errorf("objectID not found")
}

func (p *Processor) ProcessDocChanges(docID string) error {

	// get in chan
	inChan, err := p.getInChanForObjectID(docID)
	if err != nil {
		fmt.Printf("Unable to process docid %s\n", docID)
		return err
	}

	for docChange := range inChan {
		// do stuff.... then return result.
		res := proto.DocConfirmation{}
		res.ObjectId = docChange.ObjectId
		res.PropertyPath = docChange.PropertyPath
		res.UniqueId = docChange.UniqueId
		res.Data = docChange.Data

		// loop through all out channels and send result.
		// this REALLY sucks holding the lock for this long, but will do for now.
		// FIXME(kpfaulkner) MUST optimise this!
		p.objectChannelLock.Lock()
		for _, oc := range p.objectChannels[docID].outChannels {
			oc <- &res
		}
		p.objectChannelLock.Unlock()
	}
	return nil
}
