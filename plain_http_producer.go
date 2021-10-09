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

	"github.com/Financial-Times/go-logger"
	queueProducer "github.com/Financial-Times/message-queue-go-producer/producer"
)

type plainHTTPMessageProducer struct {
	config queueProducer.MessageProducerConfig
	client plainHTTPClient
}

type plainHTTPClient interface {
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

	req.Header.Add("X-Request-Id", message.Headers["X-Request-Id"])

	originSystem, found := message.Headers["Origin-System-Id"]
	if !found {
		logger.NewEntry(message.Headers["X-Request-Id"]).WithUUID(uuid).Info("Couldn't extract origin system id. Going on.")
	} else {
		req.Header.Add("X-Origin-System-Id", originSystem)
	}

	timestamp := message.Headers["Message-Timestamp"]
	if timestamp != "" {
		req.Header.Add("Message-Timestamp", timestamp)
	}
	if len(c.config.Authorization) > 0 {
		req.Header.Add("Authorization", c.config.Authorization)
	}

	nativeHash, found := message.Headers["Native-Hash"]
	if found {
		req.Header.Add("X-Native-Hash", nativeHash)
	}

	contentType, found := message.Headers["Content-Type"]
	if found {
		req.Header.Add("Content-Type", contentType)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf("Error executing POST request to the ELB: %v", err.Error())
		return errors.New(errMsg)
	}
	defer func() {
		if n, err := io.Copy(ioutil.Discard, resp.Body); err != nil {
			logger.Fatalf(map[string]interface{}{"written": n, "body": resp.Body}, err, "discarding body")
		}
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
		return "Forwarding messages is broken. Error creating new plainHttp producer healthcheck request", err
	}
	req.Header.Add("Authorization", c.config.Authorization)

	resp, err := c.client.Do(req)
	if err != nil {
		return "Forwarding messages is broken. Error executing GET request. ", err
	}

	defer func() {
		if n, err := io.Copy(ioutil.Discard, resp.Body); err != nil {
			logger.Fatalf(map[string]interface{}{"written": n, "body": resp.Body}, err, "discarding body")
		}
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Healthcheck: Request to plainHTTP producer /__health endpoint failed. Status: %d.", resp.StatusCode)
		return "Forwarding messages is broken.", errors.New(errMsg)
	}

	return "", nil
}
