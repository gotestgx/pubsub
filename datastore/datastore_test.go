package datastore

/*
 *   Each topic has it's own datastore
 */


import (
    "testing"
    "github.com/gotestgx/pubsub/errors"
)


func  Test_Topic_Subscribe(t *testing.T) {
     ds := NewPubSubDS()
     rtn :=  ds.Subscribe("sub1")


     if rtn != errors.Ok {
         t.Fail()
     }

     if ds.subscriberNextSequenceNumber["sub1"] != 1 {
          t.Fail()
     }
}


func  Test_Topic_UnSubscribe(t *testing.T) {

     ds := NewPubSubDS()

     rtn := ds.Unsubscribe("sub1")
    
     if rtn != errors.NotSubscribed {
         t.Fail()        
     }

     ds.Subscribe("sub1")
     rtn = ds.Unsubscribe("sub1")      

     _, hasVal := ds.subscriberNextSequenceNumber["sub1"]
     if hasVal  {
          t.Fail()
     }
}

//TODO more test for other operations