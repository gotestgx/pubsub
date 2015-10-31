package datastore


import (
	"fmt"
	"github.com/gotestgx/pubsub/errors"
	"log"
	"os"
)

type sequenceNumber uint64

/*
 *  DataStore structure (***** each Topic has its own DataStore *****)
 *     
 */
type PubSubDS struct {
	subscriberNextSequenceNumber map[string]sequenceNumber     // key: subscriber_name,  value: seq number of next message to be delivered on get 
	nextSequenceNumber           sequenceNumber                // sequence number of next message that is published
	messages                     map[sequenceNumber]message    // key: sequence number,  value: the message (struct below)
	debug                        *log.Logger
}

/*
 *  Struct containing message and meta info
 */
type message struct {
	number         sequenceNumber    // sequence number of message
	payload        interface{}       // the message
	referenceCount int               // number of subscribers that have not polled this message (when 0, message will be deleted)
}

/*
 *  Creates the data store
 */ 
func NewPubSubDS() *PubSubDS {
	x := new(PubSubDS)
	x.subscriberNextSequenceNumber = make(map[string]sequenceNumber)
	x.nextSequenceNumber = 1
	x.messages = map[sequenceNumber]message{}
	x.debug = log.New(os.Stderr, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	return x
}

/*
 *    Susbcribe
 *       subscriberName: subscriber name
 */
func (s *PubSubDS) Subscribe(subscriberName string) int {
	if s.isSubscribed(subscriberName) {
		return errors.AlreadySubscribed
	}

	s.subscriberNextSequenceNumber[subscriberName] = s.nextSequenceNumber

	return errors.Ok
}

/*
 *     Unsubscribe
 */
func (s *PubSubDS) Unsubscribe(subscriberName string) int {
	if !s.isSubscribed(subscriberName) {
		return errors.NotSubscribed
	}

	delete(s.subscriberNextSequenceNumber, subscriberName)

	return errors.Ok
}

/*
 *      Publish message
 */
func (s *PubSubDS) Publish(payload interface{}) {
	numSubscribers := len(s.subscriberNextSequenceNumber)
	m := message{s.nextSequenceNumber, payload, numSubscribers}
	s.messages[m.number] = m
	s.nextSequenceNumber += 1
	s.debug.Printf("ds.Publish() published %#v\n", payload)
}

/*
 *     GetMessage - get last message for subscriber
 *       
 */
func (s *PubSubDS) GetMessage(subscriberName string) (interface{}, int) {
	n, ok := s.subscriberNextSequenceNumber[subscriberName]
	if !ok {
		return nil, errors.NotSubscribed
	}

	if n == s.nextSequenceNumber {
		return nil, errors.NoNewMessage
	}

	message, ok := s.messages[n]
	if !ok {
        	panic(fmt.Sprintf("failed to find message number %d", n))
	}

	s.subscriberNextSequenceNumber[subscriberName] = n + 1
	message.referenceCount -= 1
	if message.referenceCount == 0 {
		delete(s.messages, message.number)
	}

	return message.payload, errors.Ok
}

/*
 *     GetAllMessages - REturns all messages for subscriber
 */
func (s *PubSubDS) GetAllMessages(subscriberName string) ([]interface{}, int) {
	n, ok := s.subscriberNextSequenceNumber[subscriberName]
	if !ok {
		return nil, errors.NotSubscribed
	}

	var rv []interface{}
	for n < s.nextSequenceNumber {
		message, ok := s.messages[n]
		if !ok {
		     panic(fmt.Sprintf("failed to find message number %d", n))
		}

		s.subscriberNextSequenceNumber[subscriberName] = n + 1
		message.referenceCount -= 1
		if message.referenceCount == 0 {
			delete(s.messages, message.number)
		}

		rv = append(rv, message.payload)
	}

	return rv, errors.Ok
}

/*
 *  Returns whether subscriber is subscribed
 */
func (s *PubSubDS) isSubscribed(subscriber string) bool {
	_, ok := s.subscriberNextSequenceNumber[subscriber]
	return ok
}
