package main

import (
	"errors"
	"fmt"
	queueProducer "github.com/Financial-Times/message-queue-go-producer/producer"
	"net/http"
	"strings"
)

type PlainHttpMessageProducer struct {
	config queueProducer.MessageProducerConfig
	client *http.Client
}

// newPlainHttpMessageProducer returns a plain-http-producer which behaves as a producer for kafka (writes messages to kafka), but it's actually making a simple http call to an endpoint
func newPlainHttpMessageProducer(config queueProducer.MessageProducerConfig) queueProducer.MessageProducer {
	cmsNotifier := &PlainHttpMessageProducer{config, &http.Client{}}
	return cmsNotifier
}

func (c *PlainHttpMessageProducer) SendMessage(uuid string, message queueProducer.Message) (err error) {

	req, err := http.NewRequest("POST", c.config.Addr+"/notify", strings.NewReader(message.Body))
	if err != nil {
		logger.error(fmt.Sprintf("Error creating new request: %v", err.Error()))
		return
	}

	originSystem, err := extractOriginSystem(message.Headers)
	if err != nil {
		logger.error(fmt.Sprintf("Error parsing origin system id. Skip forwarding message. Reason: %s", err.Error()))
		return err
	}

	req.Header.Add("X-Origin-System-Id", originSystem)
	req.Header.Add("X-Request-Id", message.Headers["X-Request-Id"])

	if len(c.config.Authorization) > 0 {
		req.Header.Add("Authorization", c.config.Authorization)
	}
	if len(c.config.Queue) > 0 {
		req.Host = c.config.Queue
	}

	ctxlogger := txCombinedLogger{logger, message.Headers["X-Request-Id"]}

	resp, err := c.client.Do(req)
	if err != nil {
		ctxlogger.error(fmt.Sprintf("Error executing POST request to the ELB: %v", err.Error()))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Forwarding message with tid: %s is not successful. Status: %d", message.Headers["X-Request-Id"], resp.StatusCode)
		ctxlogger.error(errMsg)
		return errors.New(errMsg)
	}
	return nil
}
