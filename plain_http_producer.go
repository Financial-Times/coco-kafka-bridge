package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	log "github.com/Financial-Times/go-logger/v2"
	queueProducer "github.com/Financial-Times/message-queue-go-producer/producer"
)

type plainHTTPMessageProducer struct {
	config queueProducer.MessageProducerConfig
	client plainHttpClient
	logger *log.UPPLogger
}

type plainHttpClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

// newPlainHTTPMessageProducer returns a plain-http-producer which behaves as a producer for kafka (writes messages to kafka), but it's actually making a simple http call to an endpoint
func newPlainHTTPMessageProducer(config queueProducer.MessageProducerConfig, logger *log.UPPLogger) queueProducer.MessageProducer {
	cmsNotifier := &plainHTTPMessageProducer{
		config: config,
		client: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 100,
				Dial: (&net.Dialer{
					KeepAlive: 30 * time.Second,
				}).Dial,
			}},
		logger: logger}
	return cmsNotifier
}

func (p *plainHTTPMessageProducer) SendMessage(uuid string, message queueProducer.Message) (err error) {
	req, err := http.NewRequest("POST", p.config.Addr+"/notify", strings.NewReader(message.Body))
	if err != nil {
		return fmt.Errorf("error creating new request: %w", err)
	}

	req.Header.Add("X-Request-Id", message.Headers["X-Request-Id"])

	originSystem, found := message.Headers["Origin-System-Id"]
	if !found {
		p.logger.WithTransactionID(message.Headers["X-Request-Id"]).WithUUID(uuid).Info("Couldn't extract origin system id. Going on.")
	} else {
		req.Header.Add("X-Origin-System-Id", originSystem)
	}

	timestamp := message.Headers["Message-Timestamp"]
	if timestamp != "" {
		req.Header.Add("Message-Timestamp", timestamp)
	}
	if len(p.config.Authorization) > 0 {
		req.Header.Add("Authorization", p.config.Authorization)
	}

	nativeHash, found := message.Headers["Native-Hash"]
	if found {
		req.Header.Add("X-Native-Hash", nativeHash)
	}

	contentType, found := message.Headers["Content-Type"]
	if found {
		req.Header.Add("Content-Type", contentType)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("error executing POST request to the ELB: %w", err)
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("forwarding message with tid: %s is not successful. Status: %d", message.Headers["X-Request-Id"], resp.StatusCode)
	}
	return nil
}

func (p *plainHTTPMessageProducer) ConnectivityCheck() (string, error) {
	req, err := http.NewRequest("GET", p.config.Addr+"/__health", nil)
	if err != nil {
		return "Forwarding messages is broken. Error creating new plainHttp producer healthcheck request", err
	}
	req.Header.Add("Authorization", p.config.Authorization)

	resp, err := p.client.Do(req)
	if err != nil {
		return "Forwarding messages is broken. Error executing GET request. ", err
	}

	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("healthcheck: Request to plainHTTP producer /__health endpoint failed. Status: %d", resp.StatusCode)
		return "Forwarding messages is broken.", err
	}

	return "", nil
}
