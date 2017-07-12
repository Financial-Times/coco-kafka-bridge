package main

import (
	"net/http"

	fthealth "github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/service-status-go/gtg"
)

type HealthCheck struct {
	consumer     consumer.MessageConsumer
	producer     producer.MessageProducer
	producerType string
}

func NewHealthCheck(consumerConf *consumer.QueueConfig, p producer.MessageProducer, producerType string, client *http.Client) *HealthCheck {
	c := consumer.NewConsumer(*consumerConf, func(m consumer.Message) {}, client)
	return &HealthCheck{
		consumer:     c,
		producer:     p,
		producerType: producerType,
	}
}

func (hc HealthCheck) Health() func(w http.ResponseWriter, r *http.Request) {
	if hc.producerType == proxy {
		return fthealth.HandlerParallel("Dependent services healthcheck", "Services: source-kafka-proxy, destination-kafka-proxy", hc.consumeHealthcheck(), hc.proxyForwarderHealthcheck())
	}
	return fthealth.HandlerParallel("Dependent services healthcheck", "Services: source-kafka-proxy, cms-notifier", hc.consumeHealthcheck(), hc.httpForwarderHealthcheck())
}

func (hc HealthCheck) consumeHealthcheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Consuming messages through kafka-proxy won't work. Publishing in the containerised stack won't work.",
		Name:             "Consume messages from kafka-proxy",
		PanicGuide:       "https://dewey.ft.com/kafka-bridge.html",
		Severity:         1,
		TechnicalSummary: "Consuming messages is broken. Check if source proxy is reachable.",
		Checker:          hc.consumer.ConnectivityCheck,
	}
}

func (hc HealthCheck) proxyForwarderHealthcheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Forwarding messages to kafka-proxy in coco won't work. Publishing in the containerised stack won't work.",
		Name:             "Forward messages to kafka-proxy.",
		PanicGuide:       "https://dewey.ft.com/kafka-bridge.html",
		Severity:         1,
		TechnicalSummary: "Forwarding messages is broken. Check if destination proxy is reachable.",
		Checker:          hc.producer.ConnectivityCheck,
	}
}

func (hc HealthCheck) httpForwarderHealthcheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Forwarding messages to cms-notifier in coco won't work. Publishing in the containerised stack won't work.",
		Name:             "Forward messages to cms-notifier",
		PanicGuide:       "https://dewey.ft.com/kafka-bridge.html",
		Severity:         1,
		TechnicalSummary: "Forwarding messages is broken. Check networking, cluster reachability and/or cms-notifier state.",
		Checker:          hc.producer.ConnectivityCheck,
	}
}

func (hc HealthCheck) GTG() gtg.Status {
	consumerCheck := func() gtg.Status {
		return gtgCheck(hc.consumer.ConnectivityCheck)
	}

	producerCheck := func() gtg.Status {
		return gtgCheck(hc.producer.ConnectivityCheck)
	}

	return gtg.FailFastParallelCheck([]gtg.StatusChecker{
		consumerCheck,
		producerCheck,
	})()
}

func gtgCheck(handler func() (string, error)) gtg.Status {
	if _, err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}
