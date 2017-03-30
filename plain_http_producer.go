package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	queueProducer "github.com/Financial-Times/message-queue-go-producer/producer"
)

type plainHTTPMessageProducer struct {
	config queueProducer.MessageProducerConfig
	client plainHttpClient
}

type plainHttpClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

// newPlainHTTPMessageProducer returns a plain-http-producer which behaves as a producer for kafka (writes messages to kafka), but it's actually making a simple http call to an endpoint
func newPlainHTTPMessageProducer(config queueProducer.MessageProducerConfig) queueProducer.MessageProducer {
	cmsNotifier := &plainHTTPMessageProducer{config, &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
			Dial: (&net.Dialer{
				KeepAlive: 30 * time.Second,
			}).Dial,
		}}}
	return cmsNotifier
}

func (c *plainHTTPMessageProducer) SendMessage(uuid string, message queueProducer.Message) (err error) {
	req, err := http.NewRequest("POST", c.config.Addr+"/notify", strings.NewReader(message.Body))
	if err != nil {
		errMsg := fmt.Sprintf("Error creating new request: %v", err.Error())
		return errors.New(errMsg)
	}
	originSystem, found := message.Headers["Origin-System-Id"]
	if !found {
		logger.info("Couldn't extract origin system id. Going on.")
	} else {
		req.Header.Add("X-Origin-System-Id", originSystem)
	}
	req.Header.Add("X-Request-Id", message.Headers["X-Request-Id"])

	timestamp := message.Headers["Message-Timestamp"]
	if timestamp != "" {
		req.Header.Add("Message-Timestamp", timestamp)
	}
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
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Forwarding message with tid: %s is not successful. Status: %d", message.Headers["X-Request-Id"], resp.StatusCode)
		return errors.New(errMsg)
	}
	return nil
}

func (c *plainHTTPMessageProducer) ConnectivityCheck() (string, error) {
	req, err := http.NewRequest("GET", c.config.Addr+"/__health", nil)
	if err != nil {
		logger.error(fmt.Sprintf("Error creating new plainHttp producer healthcheck request: %v", err.Error()))
		return "Forwarding messages is broken.", err
	}
	req.Host = c.config.Queue
	req.Header.Add("Authorization", c.config.Authorization)

	resp, err := c.client.Do(req)
	if err != nil {
		logger.warn(fmt.Sprintf("Healthcheck: Error executing GET request: %v", err.Error()))
		return "Forwarding messages is broken.", err
	}

	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Healthcheck: Request to plainHTTP producer /__health endpoint failed. Status: %d.", resp.StatusCode)
		logger.warn(errMsg)
		return "Forwarding messages is broken.", errors.New(errMsg)
	}

	return "", nil
}
