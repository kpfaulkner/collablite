package server

import (
	"fmt"
	"sync"
	"time"

	"github.com/kpfaulkner/collablite/pkg/storage"
	"github.com/kpfaulkner/collablite/proto"
	log "github.com/sirupsen/logrus"
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
	objectChannelLock sync.RWMutex

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

	log.Debugf("Registering client %s against object %s", clientID, objectID)

	var oc *ObjectIDChannels
	var ok bool
	if oc, ok = p.objectChannels[objectID]; !ok {
		oc = &ObjectIDChannels{}
		oc.inChannel = make(chan *proto.ObjectChange, 00) // FIXME(kpfaulkner) configure 100000
		oc.outChannels = make(map[string]chan *proto.ObjectConfirmation)
		p.objectChannels[objectID] = oc

		log.Debugf("creating process goroutine %s\n", objectID)

		// this is a new object being processed, so start a go routine to process it.
		go p.ProcessObjectChanges(objectID, oc.inChannel)
	} else {

		// already registered... BUT... will allow this to proceed and not return error.
		// At worst, the client will get the entire document (which they should already have).
		log.Warnf("ProcessObjectChanges for objectID %s but already exists\n", objectID)
	}

	var clientObjectChannel chan *proto.ObjectConfirmation
	if clientObjectChannel, ok = oc.outChannels[clientID]; !ok {
		// create an out channel specific for the client. This will be used to send results.
		oc.outChannels[clientID] = make(chan *proto.ObjectConfirmation, 100000) // FIXME(kpfaulkner) configure 100000
		clientObjectChannel = oc.outChannels[clientID]
	}

	// new client... populate with current state of object.
	// FIXME(kpfaulkner) This is a problem. If the number of properties for this object is greater than the channel
	// buffer size, then the channel will block, populateDocIntoClientChannel wont return...  and we're stuck in
	// a deadlock with the defer NOT unlocking the lock. Need to fix this.
	//p.populateDocIntoClientChannel(objectID, clientObjectChannel)

	return oc.inChannel, clientObjectChannel, nil
}

func (p *Processor) UnregisterClientWithObject(clientID string, objectID string) error {
	p.objectChannelLock.Lock()
	defer p.objectChannelLock.Unlock()

	log.Debugf("Unregistering client %s against object %s", clientID, objectID)

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

// ProcessObjectChanges is purely for reading the incoming changes for a specific object
// writing it to storage and then sending the results to all clients that are listening
func (p *Processor) ProcessObjectChanges(objectID string, inChan chan *proto.ObjectChange) error {

	for objChange := range inChan {

		// do stuff.... then return result.
		err := p.db.Add(objChange.ObjectId, objChange.PropertyId, objChange.Data)
		if err != nil {
			log.Errorf("Unable to add to DB for objectID %s : %+v", objectID, err)
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
		// This might be a point of optimisation. Constantly checking that map is going to be expensive (gut feel, NOT
		// measured). Could have a flag to indicate IF the clients registered for this object have changed.
		// IF there is a change, then we read from map, otherwise we used something we've cached.
		// FIXME(kpfaulkner) major problem!
		chans, ok := p.objectChannels[objectID]
		p.objectChannelLock.Unlock()
		if ok {
			for _, oc := range chans.outChannels {
				select {
				case oc <- &res:
					// nothing...  body required
				case <-time.After(10 * time.Millisecond):
					// if we cannot send the data to the client for some reason... just drop the message?
					log.Warnf("Unable to send to client. Channel full? Dropping message")
				}
			}
		}

	}
	return nil
}

// populateDocIntoClientChannel populates an entire object into a channel.
// This is used when clients are freshly registered for a document. Before they can make use of the updates
// they need to know the current state of the document.
// It is assumed ( FIXME(kpfaulkner) sort this out!) that the client is already registered with the objectID
// and that a lock is already across the objectID from the caller. Bet this will bite me one day...
func (p *Processor) populateDocIntoClientChannel(objectID string, clientObjectChannel chan *proto.ObjectConfirmation) error {
	// read entire object here and push to clientObjectChannel? Doesn't feel like this is the right place to do this?
	obj, err := p.db.Get(objectID)
	if err != nil {
		log.Errorf("unable to retrieve object %s during client registration", objectID)
		return err
	}

	log.Debugf("POPULATING ENTIRE OBJECT %s : no props %d", obj.ObjectID, len(obj.Properties))

	count := 0
	for pName, pValue := range obj.Properties {
		res := proto.ObjectConfirmation{}
		res.ObjectId = objectID
		res.PropertyId = pName
		res.UniqueId = "" // empty unique id means the client should just accept the property. HACK
		res.Data = pValue

		count++
		log.Debugf("prop %d", count)
		clientObjectChannel <- &res
	}
	log.Debugf("FINISHED POPULATING ENTIRE OBJECT %s", obj.ObjectID)

	return nil
}
