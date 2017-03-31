package main

import (
	"github.com/stretchr/testify/mock"
	"github.com/Financial-Times/message-queue-go-producer/producer"
	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
	queueProducer "github.com/Financial-Times/message-queue-go-producer/producer"
	"testing"
	"errors"
	"net/http/httptest"
	"net/http"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/Financial-Times/service-status-go/httphandlers"
)

const (
	kafkaTopicsResponseBody = "[\"Concept\", \"NativeCmsMetadataPublicationEvents\"]"
	presentTopic = "Concept"
	nonExistingTopic = "invalid"
)

func TestGtgProducerUnhappy(t *testing.T) {
	producer := new(ProducerMock)
	producer.On("ConnectivityCheck").Return("Producer is broken" , errors.New("Producer is broken"))
	bridgeApp := initBridge(producer, &queueConsumer.QueueConfig{})
	req := httptest.NewRequest("GET", "http://example.com/__gtg", nil)
	w := httptest.NewRecorder()

	httphandlers.NewGoodToGoHandler(bridgeApp.gtgCheck)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGtgHappyFlow(t *testing.T) {
	producer := new(ProducerMock)
	producer.On("ConnectivityCheck").Return("Producer is working", nil)
	ts := initConsumerMock()
	defer ts.Close()
	validConsumerConfig := &queueConsumer.QueueConfig{AuthorizationKey:"dummy", Topic:presentTopic, Addrs:[]string{ts.URL}}
	bridgeApp := initBridge(producer, validConsumerConfig)
	req := httptest.NewRequest("GET", "http://example.com/__gtg", nil)
	w := httptest.NewRecorder()

	httphandlers.NewGoodToGoHandler(bridgeApp.gtgCheck)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

type ProducerMock struct {
	mock.Mock
}

func TestUnhappyGTGCheck(t *testing.T) {
	p := new(ProducerMock)
	p.On("ConnectivityCheck").Return("Producer is broken", errors.New("Producer is broken"))
}

func (p *ProducerMock) ConnectivityCheck() (string, error) {
	args := p.Called()
	return args.String(0), args.Error(1)
}

func (p *ProducerMock) SendMessage(uuid string, msg producer.Message) error {
	args := p.Called(uuid, msg)
	return args.Error(0)
}

func initBridge(producer queueProducer.MessageProducer, consumerConfig *queueConsumer.QueueConfig) BridgeApp {
	return BridgeApp{
		producerInstance:producer,
		consumerConfig:consumerConfig,
		httpClient:http.DefaultClient,
	}
}

func initConsumerMock() *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, kafkaTopicsResponseBody)
	}))

	return ts
}
