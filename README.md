# pubsub
A simple pub-sub polling service with an HTTP-based interface.
* A topic is identified by a name
* Subscribers to a topic do not receive past messages
* Subscribing to a non-existing topic creates the topic
* Messages are stored in-memory
* A message is deleted after all subscribers have received it

### API
pubsub provides a Restful service HTTP-based interface

#### Publisher
* publish: POST /:topic_name with JSON body as a message (response 204)
   
#### Subscriber
* subscribe 
  POST /:topic_name/:subscriber_name (201 response)
* unsubscribe: DELETE /:topic_name/:subscriber_name (204 response)
* get message: GET /:topic_name/:subscriber_name (200 with JSON body as a message or 404 if no subscription found)

#### Message format
{
“message” : “variable content string”,
“published” : “date” // returned only for polling
}


## Code structure
Code consists of a main.go and three packages:
* ws - Webservice API

  The web service package utilizes the gorilla mux package to service Web Service calls
  The Web Service calls are pushed onto a service channel where they are processed
* datastore - A data structure implementing PubSub semantics for arbitrary messages 

  A Map-backed datastore providing, on average, constant O(1) lookup, insert, and delete times

  Message storage is unbounded. Slow subscribers can cause an Out of Memory condition.

  The topic datastore uses two structures: 'message' and 'PubSubDS':
     * 'message' struct contains:
         * a published message
         * meta data (sequence number of message, number of subscribers who have yet to poll the message)
     * 'PubSubDS' struct contains:
         * a map of sequence number to 'message'
         * a map of subscriber names to the sequence number of their next message
         * sequence number for next published message
  
     * errors - status constants


## TODO
 * Additional testing modules - provided one test function in both ws and datastore packages.  Obviously, more test are needed.
   * Make Http server stoppable, needed for webservice unit testing
 * Fill out the code commenting
 * Logging improvements
 * Configuration
   * Make port and binding configurable
   * Configure the capacity of max Web Service requests, and return 500 when capacity reached
 * Handle edge cases
   * Handle Out of Memory conditions
   * Handle case where a user subscribers, never polls for messages (meaning no message would get deleted)

## Possible enhancements
 * Could trivially have a disk-based backup using go's built-in binary encoding
 * Replace in-memory go datastructures with a clustered key/value store such as Riak, Redis, etc 
 * Could add rate limiting
 * Linux init.d or systems scripts to run as as service  
