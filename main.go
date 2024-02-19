package main

import (
	"context"
	server "gossip-price/core/node"
	"log"
	"os"
	"os/signal"
)

func main() {

	server, err := server.NewGossipServer()
	if err != nil {
		log.Print(err)
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	err = server.Start(ctx)
	if err != nil {
		log.Print("Gossip server starting error")
		return
	}
	<-server.Wait()
}
