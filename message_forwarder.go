package main

import (
	"errors"
	"fmt"
	queueProducer "github.com/Financial-Times/message-queue-go-producer/producer"
	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/dchest/uniuri"
	"regexp"
)

const tidValidRegexp = "(tid|SYNTHETIC-REQ-MON)[a-zA-Z0-9_-]*$"

func (bridge BridgeApp) forwardMsg(msg queueConsumer.Message) {
	tid, err := extractTID(msg.Headers)
	if err != nil {
		logger.info(fmt.Sprintf("Couldn't extract transaction id: %s", err.Error()))
		tid = "tid_" + uniuri.NewLen(10) + "_kafka_bridge"
		logger.info("Generated tid: " + tid)
	}
	msg.Headers["X-Request-Id"] = tid
	ctxLogger := txCombinedLogger{logger, tid}
	err = bridge.producerInstance.SendMessage("", queueProducer.Message{Headers: msg.Headers, Body: msg.Body})
	if err != nil {
		ctxLogger.error("Error happened during message forwarding. " + err.Error())
	} else {
		ctxLogger.info("Message forwarded")
	}
}

func extractTID(headers map[string]string) (string, error) {
	header := headers["X-Request-Id"]
	if header == "" {
		return "", errors.New("X-Request-Id header could not be found.")
	}
	validRegexp := regexp.MustCompile(tidValidRegexp)
	tid := validRegexp.FindString(header)
	if tid == "" {
		return "", fmt.Errorf("Transaction ID is in unknown format: %s.", header)
	}
	return tid, nil
}
