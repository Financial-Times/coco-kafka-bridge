package main

import (
	"errors"
	"fmt"
	queueProducer "github.com/Financial-Times/message-queue-go-producer/producer"
	"net/http"
	"strings"
	"time"
)

type plainHTTPMessageProducer struct {
	config queueProducer.MessageProducerConfig
	client *http.Client
}

// newPlainHTTPMessageProducer returns a plain-http-producer which behaves as a producer for kafka (writes messages to kafka), but it's actually making a simple http call to an endpoint
func newPlainHTTPMessageProducer(config queueProducer.MessageProducerConfig) queueProducer.MessageProducer {
	cmsNotifier := &plainHTTPMessageProducer{config, &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		}}}
	return cmsNotifier
}

func (c *plainHTTPMessageProducer) SendMessage(uuid string, message queueProducer.Message) (err error) {
	req, err := http.NewRequest("POST", c.config.Addr+"/notify", strings.NewReader(message.Body))
	if err != nil {
		logger.error(fmt.Sprintf("Error creating new request: %v", err.Error()))
		return
	}
	originSystem, err := extractOriginSystem(message.Headers)
	if err != nil {
		logger.info(fmt.Sprintf("Couldn't extract origin system id: %s . Going on.", err.Error()))
	} else {
		req.Header.Add("X-Origin-System-Id", originSystem)
	}
	req.Header.Add("X-Request-Id", message.Headers["X-Request-Id"])
	if len(c.config.Authorization) > 0 {
		req.Header.Add("Authorization", c.config.Authorization)
	}
	if len(c.config.Queue) > 0 {
		req.Host = c.config.Queue
	}
	ctxLogger := txCombinedLogger{logger, message.Headers["X-Request-Id"]}
	resp, err := c.client.Do(req)
	if err != nil {
		ctxLogger.error(fmt.Sprintf("Error executing POST request to the ELB: %v", err.Error()))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Forwarding message with tid: %s is not successful. Status: %d", message.Headers["X-Request-Id"], resp.StatusCode)
		ctxLogger.error(errMsg)
		return errors.New(errMsg)
	}
	return nil
}
