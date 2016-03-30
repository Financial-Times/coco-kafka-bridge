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
const uuidContextRegexp = "\"uuid\":\"[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}\""
const uuidValidRegexp = "[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}"
const systemIDValidRegexp = `[a-zA-Z-]*$`

func (bridge BridgeApp) forwardMsg(msg queueConsumer.Message) {

	tid, err := extractTID(msg.Headers)
	if err != nil {
		logger.warn(fmt.Sprintf("Couldn't extract transaction id: %s", err.Error()))
		tid = "tid_" + uniuri.NewLen(10) + "_kafka_bridge"
		logger.info("Generating tid: " + tid)
	}
	msg.Headers["X-Request-Id"] = tid
	ctxlogger := TxCombinedLogger{logger, tid}

	uuid, err := extractUUID(msg.Body)
	if err != nil {
		logger.error(fmt.Sprintf("Error parsing message for extracting uuid. Skip forwarding message. Reason: %s", err.Error()))
	}

	producerInstance := *bridge.producerInstance
	producerInstance.SendMessage(uuid, queueProducer.Message{Headers: msg.Headers, Body: msg.Body})

	ctxlogger.info("Message forwarded")
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

func extractUUID(msg string) (string, error) {

	if msg == "" {
		return "", errors.New("Message body is empty.")
	}
	contextRegexp := regexp.MustCompile(uuidContextRegexp)
	validRegexp := regexp.MustCompile(uuidValidRegexp)
	uuidContext := contextRegexp.FindString(msg)
	uuid := validRegexp.FindString(uuidContext)

	if uuid == "" {
		return "", fmt.Errorf("UUID is not present.")
	}
	return uuid, nil
}

func extractOriginSystem(headers map[string]string) (string, error) {
	origSysHeader := headers["Origin-System-Id"]
	validRegexp := regexp.MustCompile(systemIDValidRegexp)
	systemID := validRegexp.FindString(origSysHeader)
	if systemID == "" {
		return "", errors.New("Origin system id is not set.")
	}
	return systemID, nil
}
