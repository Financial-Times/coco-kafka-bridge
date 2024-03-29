package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/stretchr/testify/assert"
)

type mockProducerInstance struct {
	isConnectionHealthy bool
}

type mockConsumerInstance struct {
	isConnectionHealthy bool
}

func (p *mockProducerInstance) SendMessage(string, producer.Message) error {
	return nil
}

func (p *mockProducerInstance) ConnectivityCheck() (string, error) {
	if p.isConnectionHealthy {
		return "", nil
	}

	return "", errors.New("Error connecting to the queue")
}

func (c *mockConsumerInstance) Start() {
}

func (c *mockConsumerInstance) Stop() {
}

func (c *mockConsumerInstance) ConnectivityCheck() (string, error) {
	if c.isConnectionHealthy {
		return "", nil
	}

	return "", errors.New("Error connecting to the queue")
}

func initializeHealthcheck(isProducerConnectionHealthy bool, isConsumerConnectionHealthy bool, producerType string) HealthCheck {
	return HealthCheck{
		consumer:     &mockConsumerInstance{isConnectionHealthy: isConsumerConnectionHealthy},
		producer:     &mockProducerInstance{isConnectionHealthy: isProducerConnectionHealthy},
		producerType: producerType,
	}
}

func TestNewHealthCheck(t *testing.T) {
	hc := NewHealthCheck(
		&consumer.QueueConfig{},
		producer.NewMessageProducer(producer.MessageProducerConfig{}),
		"proxy",
		http.DefaultClient,
	)

	assert.NotNil(t, hc.consumer)
	assert.NotNil(t, hc.producer)
	assert.Equal(t, "proxy", hc.producerType)
}

func TestGTGHappyFlow(t *testing.T) {
	hc := initializeHealthcheck(true, true, proxy)

	status := hc.GTG()
	assert.True(t, status.GoodToGo)
	assert.Empty(t, status.Message)
}

func TestGTGBrokenConsumer(t *testing.T) {
	hc := initializeHealthcheck(true, false, proxy)

	status := hc.GTG()
	assert.False(t, status.GoodToGo)
	assert.Equal(t, "Error connecting to the queue", status.Message)
}

func TestGTGCheckBrokenProducer(t *testing.T) {
	hc := initializeHealthcheck(false, true, proxy)

	status := hc.GTG()
	assert.False(t, status.GoodToGo)
	assert.Equal(t, "Error connecting to the queue", status.Message)
}

func TestHealthHappyFlow(t *testing.T) {
	hc := initializeHealthcheck(true, true, proxy)

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()
	endpoint := hc.Health()

	endpoint(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "HealthCheck should return 200")
	checks, err := parseHealthcheck(w.Body.String())
	assert.NoError(t, err)

	for _, check := range checks {
		assert.True(t, check.Ok)
	}
}

func TestHealthBrokenProxyProducer(t *testing.T) {
	hc := initializeHealthcheck(false, true, proxy)

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()
	endpoint := hc.Health()

	endpoint(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "HealthCheck should return 200")
	checks, err := parseHealthcheck(w.Body.String())
	assert.NoError(t, err)

	for _, check := range checks {
		if check.Name == "Forward messages to kafka-proxy." {
			assert.False(t, check.Ok)
		} else {
			assert.True(t, check.Ok)
		}
	}
}

func TestHealthBrokenPlainHTTPProducer(t *testing.T) {
	hc := initializeHealthcheck(false, true, plainHTTP)

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()
	endpoint := hc.Health()

	endpoint(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "HealthCheck should return 200")
	checks, err := parseHealthcheck(w.Body.String())
	assert.NoError(t, err)

	for _, check := range checks {
		if check.Name == "Forward messages to cms-notifier" {
			assert.False(t, check.Ok)
		} else {
			assert.True(t, check.Ok)
		}
	}
}

func TestHealthBrokenConsumer(t *testing.T) {
	hc := initializeHealthcheck(true, false, proxy)

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()
	endpoint := hc.Health()

	endpoint(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "HealthCheck should return 200")
	checks, err := parseHealthcheck(w.Body.String())
	assert.NoError(t, err)

	for _, check := range checks {
		if check.Name == "Consume messages from kafka-proxy" {
			assert.False(t, check.Ok)
		} else {
			assert.True(t, check.Ok)
		}
	}
}

func parseHealthcheck(healthcheckJSON string) ([]fthealth.CheckResult, error) {
	result := &struct {
		Checks []fthealth.CheckResult `json:"checks"`
	}{}

	err := json.Unmarshal([]byte(healthcheckJSON), result)
	return result.Checks, err
}
