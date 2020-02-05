package main

import (
	"errors"

	log "github.com/Financial-Times/go-logger/v2"
	producer "github.com/Financial-Times/message-queue-go-producer/producer"
	consumer "github.com/Financial-Times/message-queue-gonsumer"

	"github.com/dchest/uniuri"
)

func (app BridgeApp) forwardMsg(msg consumer.Message) {
	if !isMsgForwardable(msg, app.region) {
		app.logger.WithMonitoringEvent("Forwarding", msg.Headers["X-Request-Id"], "").Info("Message has been skipped. Origin region same as app region.")
		return
	}

	enrichMsg(&msg, app.region, app.logger)

	err := app.producerInstance.SendMessage("", producer.Message{Headers: msg.Headers, Body: msg.Body})
	if err != nil {
		app.logger.WithError(err).WithMonitoringEvent("Forwarding", msg.Headers["X-Request-Id"], "").Error("Error happened during message forwarding")
		return
	}

	app.logger.WithMonitoringEvent("Forwarding", msg.Headers["X-Request-Id"], "").Info("Message has been forwarded")
}

func enrichMsg(msg *consumer.Message, region string, log *log.UPPLogger) {
	tid, err := extractTID(msg.Headers)
	if err != nil {
		tid = "tid_" + uniuri.NewLen(10) + "_kafka_bridge"
		log.WithError(err).Info("Couldn't extract transaction id. TID was generated.")
	}
	msg.Headers["X-Request-Id"] = tid

	// Write region in synchronisation bridges only
	if region == "eu" {
		msg.Headers["Origin-Region"] = "us"
	} else if region == "us" {
		msg.Headers["Origin-Region"] = "eu"
	}
}

func isMsgForwardable(msg consumer.Message, region string) bool {
	origRegion := msg.Headers["Origin-Region"]
	if region != "" && origRegion != "" && region == origRegion {
		return false
	}

	return true
}

func extractTID(headers map[string]string) (string, error) {
	header := headers["X-Request-Id"]
	if header == "" {
		return "", errors.New(`header "X-Request-Id" is empty or missing`)
	}

	return header, nil
}
