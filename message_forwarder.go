package main

import (
	"errors"

	queueProducer "github.com/Financial-Times/message-queue-go-producer/producer"
	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/dchest/uniuri"
)

const tidValidRegexp = "(tid|SYNTHETIC-REQ-MON)[a-zA-Z0-9_-]*$"

func (app BridgeApp) forwardMsg(msg queueConsumer.Message) {
	tid, err := extractTID(msg.Headers)
	if err != nil {
		tid = "tid_" + uniuri.NewLen(10) + "_kafka_bridge"
		app.logger.WithError(err).Info("Couldn't extract transaction id. TID was generated.")
	}
	msg.Headers["X-Request-Id"] = tid
	err = app.producerInstance.SendMessage("", queueProducer.Message{Headers: msg.Headers, Body: msg.Body})
	if err != nil {
		app.logger.WithError(err).WithMonitoringEvent("Forwarding", tid, "").Error("Error happened during message forwarding")
	} else {
		app.logger.WithMonitoringEvent("Forwarding", tid, "").Info("Message has been forwarded")
	}
}

func extractTID(headers map[string]string) (string, error) {
	header := headers["X-Request-Id"]
	if header == "" {
		return "", errors.New("X-Request-Id header could not be found.")
	}
	return header, nil
}
