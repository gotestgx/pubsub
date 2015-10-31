package main

import (
	"github.com/gotestgx/pubsub/ws"
)

func main() {
	ws.NewWebService("localhost", 8080).Run()
}
