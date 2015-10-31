package ws

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/gotestgx/pubsub/datastore"
	"github.com/gotestgx/pubsub/errors"
	"log"
	"net/http"
	"os"
	"fmt"
	"io/ioutil"
	"strings"
)


var (
	wsChannelCapacity = 100
	Debug *log.Logger
)

type operation int

const (
	pub   = iota   
	sub   = iota
	unsub = iota
	get   = iota
)

/*
 * Structure to hold Web Services
 */
type WebService struct {
	host string
	port uint
	wsChan chan wsCommand
	topics map[string]*datastore.PubSubDS
	router *mux.Router
	debug  *log.Logger
}

/*
 *  Message struct
 */
type message_struct struct {
	Message string
	Published    string
}

/* 
 *   Structure holding a web service command (will be put onto servicing channel)
 */
type wsCommand struct {
	op           operation          // the operation 
	topic        string             // topic name
	subscriber   string             // subscriber name
	msg          *message_struct    // message name
	replyChannel chan WSReply       // reply channel
}

type WSReply struct {
	status  int                     // status
	message *message_struct         // optional message (for get)
}


/*
 *  Publish a message
 */
func (s *WebService) publish(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	topicName := vars["topic_name"]
	
	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
    	    http.Error(rw, "invalid json", http.StatusBadRequest)
            return
	}
	
	// TODO remove this conversion to string, was for debug purposes
	body := string(bytes)
	
	s.debug.Printf("publish() request body %s", body)
	
	decoder := json.NewDecoder(strings.NewReader(body))
	var msg message_struct
	err = decoder.Decode(&msg)
	if err != nil {
		http.Error(rw, "invalid json", http.StatusBadRequest)
		return
	}

	// s.debug.Printf("message %#v\n", msg)
	
	replyChannel := make(chan WSReply, 1)
	s.wsChan <- wsCommand{op: pub, topic: topicName, msg: &msg, replyChannel: replyChannel}
	reply := <-replyChannel

	switch reply.status {
	case errors.Ok:
		rw.WriteHeader(204)
		break

	case errors.NonExistentTopic:
		http.Error(rw, fmt.Sprint("topic %s doesn't exist", topicName), http.StatusBadRequest)
		break

	default:
            http.Error(rw, fmt.Sprint("bad request"), http.StatusBadRequest)
            break
	}
}

/* 
 *  Subscribe
 */
func (s *WebService) subscribe(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	topicName := vars["topic_name"]
	subscriberName := vars["subscriber_name"]

	// s.debug.Println("WS_Subscribe_Topic", topicName, "Subscriber", subscriberName)
	
	replyChannel := make(chan WSReply, 1)
	s.wsChan <- wsCommand{op: sub, topic: topicName, subscriber: subscriberName, replyChannel: replyChannel}
	
	reply := <-replyChannel

	switch reply.status {
	case errors.Ok:
		rw.WriteHeader(201)
		break

	case errors.AlreadySubscribed:
		http.Error(rw, fmt.Sprint("already subscribed to topic %s", topicName), http.StatusBadRequest)
		break

	default:
		panic("unknown status value")
	}
}

/* 
 *  Unsubscribe
 */
func (s *WebService) unSubscribe(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	topicName := vars["topic_name"]
	subscriberName := vars["subscriber_name"]

	s.debug.Println("WS Unsubscribe from Topic", topicName, "Subscriber", subscriberName)
	replyChannel := make(chan WSReply, 1)
	s.wsChan <- wsCommand{op: unsub, topic: topicName, subscriber: subscriberName, replyChannel: replyChannel}

	reply := <-replyChannel

	switch reply.status {
	case errors.Ok:
		rw.WriteHeader(204)
		break

	case errors.NonExistentTopic:
		http.Error(rw, fmt.Sprint("topic %s doesn't exist", topicName), http.StatusBadRequest)
		break

	case errors.NotSubscribed:
		http.Error(rw, fmt.Sprint("not subscribed to topic %s", topicName), http.StatusBadRequest)
		break

	default:
		panic("unknown status value")
	}
}

/* 
 *  GetMessages 
 */
func (s *WebService) getMessages(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	topicName := vars["topic_name"]
	subscriberName := vars["subscriber_name"]

	s.debug.Println("Get Messages", topicName, "Subscriber", subscriberName)
	replyChannel := make(chan WSReply, 1)
	s.wsChan <- wsCommand{op: get, topic: topicName, subscriber: subscriberName, replyChannel: replyChannel}

	reply := <-replyChannel

	switch reply.status {
	case errors.Ok:
		s, err := json.Marshal(*reply.message)
		if err != nil {
			http.Error(rw, fmt.Sprint("topic %s doesn't exist", topicName), http.StatusInternalServerError)
			return
		}
		bytes := []byte(s)
		rw.WriteHeader(200)
		rw.Header().Set("Content-Type", "application/jsonl; charset=utf-8")
		rw.Header().Set("Content-Length", string(len(bytes)))
		rw.Write(bytes)
		break

	case errors.NonExistentTopic:
		http.Error(rw, fmt.Sprint("topic %s doesn't exist", topicName), 404)
		break

	case errors.NoNewMessage:
	    http.Error(rw, fmt.Sprint("topic %s doesn't exist", topicName), 204)
		break

	case errors.NotSubscribed:
		http.Error(rw, fmt.Sprint("not subscribed to topic %s", topicName), 404)
		break

	default:
		panic("unknown status value")
	}
}


/*
 * Start services
 */
func (s *WebService) startServices() {
	s.router = mux.NewRouter()

	publish := func(rw http.ResponseWriter, req *http.Request) {
		s.publish(rw, req)
	}

	subscribe := func(rw http.ResponseWriter, req *http.Request) {
		s.subscribe(rw, req)
	}

	unSubscribe := func(rw http.ResponseWriter, req *http.Request) {
		s.unSubscribe(rw, req)
	}

	getMessages := func(rw http.ResponseWriter, req *http.Request) {
		s.getMessages(rw, req)
	}

	s.router.HandleFunc("/{topic_name}", publish).Methods("POST")
	s.router.HandleFunc("/{topic_name}/{subscriber_name}", subscribe).Methods("POST")
	s.router.HandleFunc("/{topic_name}/{subscriber_name}", unSubscribe).Methods("DELETE")
	s.router.HandleFunc("/{topic_name}/{subscriber_name}", getMessages).Methods("GET")

	http.ListenAndServe(fmt.Sprintf("%s:%d", s.host, s.port), s.router)
}

/* 
 * Get or Create topic
 */
func (s *WebService) getOrCreateTopic(name string) *datastore.PubSubDS {
	topic, ok := s.topics[name]
	if !ok {
		topic = datastore.NewPubSubDS()
		s.topics[name] = topic
	}

	return topic
}

/*
 *  Service Web Service requests
 */
func (s *WebService) processWsChan() {
	s.debug.Println("Starting procceinsg on ", s.wsChan)

	for {

		select {

		case wsCmd, _ := <-s.wsChan:
			s.debug.Printf("got msg on channel %#v\n", wsCmd)
			switch wsCmd.op {
			case pub:
				s.debug.Println("PUB MSG ON CHAN", wsCmd.topic, "msg", wsCmd.msg)
				topic, ok := s.topics[wsCmd.topic]
				if ok {
					s.debug.Printf("publishing %#v\n", wsCmd.msg)
					topic.Publish(wsCmd.msg)
					wsCmd.replyChannel <- WSReply{status: errors.Ok, message: nil}
				} else {
					wsCmd.replyChannel <- WSReply{status: errors.NonExistentTopic, message: nil}
				}

			case sub:
				s.debug.Println("SUB MSG ON CHAN ", wsCmd.topic, "subscriber", wsCmd.subscriber)
				topic := s.getOrCreateTopic(wsCmd.topic)
				status := topic.Subscribe(wsCmd.subscriber)
				wsCmd.replyChannel <- WSReply{status: status, message: nil}

			case unsub:
				s.debug.Println("UNSUB MSG ON CHAN", wsCmd.topic, "subscriber", wsCmd.subscriber)
				topic, ok := s.topics[wsCmd.topic]
				if ok {
					status := topic.Unsubscribe(wsCmd.subscriber)
					wsCmd.replyChannel <- WSReply{status: status, message: nil}
				} else {
					wsCmd.replyChannel <- WSReply{status: errors.NonExistentTopic, message: nil}
				}

			case get:
				s.debug.Println("GET MSG ON CHAN ", wsCmd.topic, "subscriber", wsCmd.subscriber)
				topic, ok := s.topics[wsCmd.topic]
				if ok {
					x, status := topic.GetMessage(wsCmd.subscriber)
					if x == nil {
						wsCmd.replyChannel <- WSReply{status: status, message: nil}
					} else {
						msg := x.(*message_struct)
						s.debug.Printf("retrieved message %#v\n", msg)
						wsCmd.replyChannel <- WSReply{status: status, message: msg}
					}
				} else {
					wsCmd.replyChannel <- WSReply{status: errors.NonExistentTopic, message: nil}
				}

			default:
				s.debug.Println("could not process")
			}
		}
	}
}

func NewWebService(host string, port uint) *WebService {
	x := new(WebService)
	x.host = host
	x.port = port
	x.debug = log.New(os.Stderr, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	x.wsChan = make(chan wsCommand, wsChannelCapacity)
	x.topics = make(map[string]*datastore.PubSubDS)
	return x
}

func (s *WebService) Run() {
	go s.processWsChan()
	s.startServices()
}
