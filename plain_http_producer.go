package main

import (
	"errors"
	"fmt"
	queueProducer "github.com/Financial-Times/message-queue-go-producer/producer"
	"net/http"
	"strings"
	"time"
	"regexp"
)

type plainHTTPMessageProducer struct {
	config queueProducer.MessageProducerConfig
	client plainHttpClient
}

const systemIDValidRegexp = `[a-zA-Z-]*$`

type plainHttpClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
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
		errMsg := fmt.Sprintf("Error creating new request: %v", err.Error())
		return errors.New(errMsg)
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
	resp, err := c.client.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf("Error executing POST request to the ELB: %v", err.Error())
		return errors.New(errMsg)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Forwarding message with tid: %s is not successful. Status: %d", message.Headers["X-Request-Id"], resp.StatusCode)
		return errors.New(errMsg)
	}
	return nil
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
