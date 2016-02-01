package main

import (
	"fmt"
	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
)

func (bridge BridgeApp) startNewConsumer() queueConsumer.MessageIterator {
	consumerConfig := bridge.consumerConfig
	consumer := queueConsumer.NewIterator(*consumerConfig)
	return consumer
}

func (bridge BridgeApp) consumeMessages(iterator queueConsumer.MessageIterator) {
	logger.info("waiting for messages")
	for {
		msgs, err := iterator.NextMessages()
		if err != nil {
			logger.warn(fmt.Sprintf("Could not read messages: %s", err.Error()))
			continue
		}
		for _, m := range msgs {
			logger.info("forwarding message")
			bridge.forwardMsg(m)
		}
	}
}
