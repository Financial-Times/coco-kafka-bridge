package main

import (
	"errors"
	"github.com/Financial-Times/message-queue-go-producer/producer"
	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

const (
	kafkaTopicsResponseBody = "[\"Concept\", \"NativeCmsMetadataPublicationEvents\"]"
	presentTopic            = "Concept"
	nonExistingTopic        = "invalid"
)

type mockTransport struct {
	responseStatusCode int
	responseBody       string
}

type mockProducerInstance struct {
	isConnectionHealthy bool
}

func (p *mockProducerInstance) SendMessage(string, producer.Message) error {
	return nil
}

func (p *mockProducerInstance) ConnectivityCheck() (string, error) {
	if p.isConnectionHealthy {
		return "", nil
	}

	return "", errors.New("test")
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response := &http.Response{
		Header:     make(http.Header),
		Request:    req,
		StatusCode: t.responseStatusCode,
	}

	response.Header.Set("Content-Type", "application/json")
	response.Body = ioutil.NopCloser(strings.NewReader(t.responseBody))

	return response, nil
}

func initializeMockedHTTPClient(responseStatusCode int, responseBody string) *http.Client {
	client := http.DefaultClient
	client.Transport = &mockTransport{
		responseStatusCode: responseStatusCode,
		responseBody:       responseBody,
	}

	return client
}

func initializeHappyBridge() BridgeApp {
	return initializeBridge(http.StatusOK, true)
}

func initializeBrokenProxyBridge() BridgeApp {
	return initializeBridge(http.StatusInternalServerError, false)
}

func initializeBrokenProducerBridge() BridgeApp {
	return initializeBridge(http.StatusOK, false)
}

func initializeBridge(statusCode int, isProducerConnectionHealthy bool) BridgeApp {
	httpClient := initializeMockedHTTPClient(statusCode, kafkaTopicsResponseBody)
	consumerConfig := &queueConsumer.QueueConfig{AuthorizationKey: "dummy", Topic: presentTopic, Addrs: []string{"abc"}}
	return BridgeApp{
		httpClient:       httpClient,
		consumerConfig:   consumerConfig,
		producerInstance: &mockProducerInstance{isConnectionHealthy: isProducerConnectionHealthy},
	}
}

func TestCheckIfTopicIsPresentHappyFlow(t *testing.T) {
	requestBody := []byte(kafkaTopicsResponseBody)
	err := checkIfTopicIsPresent(requestBody, presentTopic)

	assert.Nil(t, err)
}

func TestCheckIfTopicIsPresentTopicIsNotPresent(t *testing.T) {
	requestBody := []byte(kafkaTopicsResponseBody)
	err := checkIfTopicIsPresent(requestBody, nonExistingTopic)

	assert.NotNil(t, err)
}

func TestCheckProxyConnectionInternalServerError(t *testing.T) {
	bridge := initializeBrokenProxyBridge()
	_, err := bridge.checkProxyConnection("dummy", "dummy", "dummy")

	assert.NotNil(t, err)
}

func TestCheckProxyConnectionHappyFlow(t *testing.T) {
	bridge := initializeHappyBridge()
	body, err := bridge.checkProxyConnection("dummy", "dummy", "dummy")

	assert.Nil(t, err)
	assert.NotNil(t, body)
}

func TestCheckConsumableProxyReturnsInternalServerError(t *testing.T) {
	initLoggers()
	bridge := initializeBrokenProxyBridge()
	err := bridge.checkConsumable("dummy")

	assert.NotNil(t, err)
}

func TestCheckConsumableHappyFlow(t *testing.T) {
	initLoggers()
	bridge := initializeHappyBridge()
	err := bridge.checkConsumable("dummy")

	assert.Nil(t, err)
}

func TestAggregateConsumableResults(t *testing.T) {
	initLoggers()
	bridge := initializeHappyBridge()
	_, err := bridge.aggregateConsumableResults()

	assert.Nil(t, err)
}

func TestAggregateConsumableResultsBrokenProxy(t *testing.T) {
	initLoggers()
	bridge := initializeBrokenProxyBridge()
	_, err := bridge.aggregateConsumableResults()

	assert.NotNil(t, err)
}

func TestAggregateConsumableNoAddressProxy(t *testing.T) {
	initLoggers()
	bridge := initializeBrokenProxyBridge()
	_, err := bridge.aggregateConsumableResults()

	assert.NotNil(t, err)
}

func TestGtgBrokenProxy(t *testing.T) {
	initLoggers()
	bridge := initializeBrokenProxyBridge()
	gtgStatus := bridge.gtgCheck()

	assert.False(t, gtgStatus.GoodToGo)
}

func TestGtgHappyFlow(t *testing.T) {
	initLoggers()
	bridge := initializeHappyBridge()
	gtgStatus := bridge.gtgCheck()

	assert.True(t, gtgStatus.GoodToGo)
}

func TestGtgConnectionDown(t *testing.T) {
	initLoggers()
	bridge := initializeBrokenProducerBridge()
	gtgStatus := bridge.gtgCheck()

	assert.False(t, gtgStatus.GoodToGo)
}
