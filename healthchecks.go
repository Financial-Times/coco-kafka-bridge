package main

import (
	"net/http"
	"time"

	fthealth "github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/service-status-go/gtg"
)

const requestTimeout = 4500

type Healthcheck struct {
	consumer     consumer.MessageConsumer
	producer     producer.MessageProducer
	producerType string
}

func newHealthcheck(consumerConfig *consumer.QueueConfig, producerConfig *producer.MessageProducerConfig, producerType string) *Healthcheck {
	httpClient := &http.Client{Timeout: requestTimeout * time.Millisecond}
	consumerInstance := consumer.NewConsumer(*consumerConfig, func(m consumer.Message) {}, httpClient)
	producerInstance := producer.NewMessageProducerWithHTTPClient(*producerConfig, httpClient)
	return &Healthcheck{
		consumer:     consumerInstance,
		producer:     producerInstance,
		producerType: producerType,
	}
}

func (hc Healthcheck) Health() func(w http.ResponseWriter, r *http.Request) {
	if hc.producerType == proxy {
		return fthealth.HandlerParallel("Dependent services healthcheck", "Services: kafka-rest-proxy@ucs, kafka-rest-proxy@aws", hc.consumeHealthcheck(), hc.proxyForwarderHealthcheck())
	}
	return fthealth.HandlerParallel("Dependent services healthcheck", "Services: kafka-rest-proxy@ucs, cms-notifier@aws", hc.consumeHealthcheck(), hc.httpForwarderHealthcheck())
}

func (hc Healthcheck) consumeHealthcheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Consuming messages through kafka-proxy won't work. Publishing in the containerised stack won't work.",
		Name:             "Consume from UCS kafka through the proxy",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/kafka-bridge-run-book",
		Severity:         1,
		TechnicalSummary: "Consuming messages is broken. Check if kafka-proxy in aws is reachable.",
		Checker:          hc.consumer.ConnectivityCheck,
	}
}

func (hc Healthcheck) proxyForwarderHealthcheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Forwarding messages to kafka-proxy in coco won't work. Publishing in the containerised stack won't work.",
		Name:             "Forward messages to kafka-proxy.",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/kafka-bridge-run-book",
		Severity:         1,
		TechnicalSummary: "Forwarding messages is broken. Check if kafka-proxy in coco is reachable.",
		Checker:          hc.producer.ConnectivityCheck,
	}
}

func (hc Healthcheck) httpForwarderHealthcheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Forwarding messages to cms-notifier in coco won't work. Publishing in the containerised stack won't work.",
		Name:             "Forward messages to cms-notifier",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/kafka-bridge-run-book",
		Severity:         1,
		TechnicalSummary: "Forwarding messages is broken. Check networking, aws cluster reachability and/or coco cms-notifier state.",
		Checker:          hc.producer.ConnectivityCheck,
	}
}

func (hc Healthcheck) GTG() gtg.Status {
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
