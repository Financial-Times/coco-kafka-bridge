package main

import (
	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
	"net/http"
	"sync"
	"os"
	"os/signal"
	"syscall"
)

func (bridge BridgeApp) consumeMessages() {
	consumerConfig := bridge.consumerConfig

	consumer := queueConsumer.NewConsumer(*consumerConfig, bridge.forwardMsg, http.Client{})

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		consumer.Start()
		wg.Done()
	}()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	consumer.Stop()
	wg.Wait()
}