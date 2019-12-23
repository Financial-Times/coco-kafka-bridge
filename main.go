package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	log "github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/service-status-go/httphandlers"
)

// BridgeApp wraps the config and represents the API for the bridge
type BridgeApp struct {
	consumerConfig   *consumer.QueueConfig
	producerConfig   *producer.MessageProducerConfig
	producerInstance producer.MessageProducer
	producerType     string
	httpClient       *http.Client
	serviceName      string
	logger           *log.UPPLogger
}

const (
	plainHTTP = "plainHTTP"
	proxy     = "proxy"
)

func newBridgeApp(consumerAddrs string, consumerGroupID string, consumerOffset string, consumerAutoCommitEnable bool, consumerAuthorizationKey string, topic string, producerAddress string, producerVulcanAuth string, producerType string, serviceName string, logger *log.UPPLogger) *BridgeApp {
	consumerConfig := consumer.QueueConfig{}
	consumerConfig.Addrs = strings.Split(consumerAddrs, ",")
	consumerConfig.Group = consumerGroupID
	consumerConfig.Topic = topic
	consumerConfig.Offset = consumerOffset
	consumerConfig.AuthorizationKey = consumerAuthorizationKey
	consumerConfig.AutoCommitEnable = consumerAutoCommitEnable

	producerConfig := producer.MessageProducerConfig{}
	producerConfig.Addr = producerAddress
	producerConfig.Topic = topic
	producerConfig.Authorization = producerVulcanAuth

	var producerInstance producer.MessageProducer
	switch producerType {
	case proxy:
		producerInstance = producer.NewMessageProducer(producerConfig)
	case plainHTTP:
		producerInstance = newPlainHTTPMessageProducer(producerConfig, logger)
	default:
		logger.WithError(fmt.Errorf("unknown producer type %s", producerType)).Fatalf("The provided producer type '%v' is invalid", producerType)
	}

	httpClient := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
			Dial: (&net.Dialer{
				KeepAlive: 30 * time.Second,
			}).Dial,
		}}

	app := &BridgeApp{
		consumerConfig:   &consumerConfig,
		producerConfig:   &producerConfig,
		producerInstance: producerInstance,
		producerType:     producerType,
		httpClient:       httpClient,
		serviceName:      serviceName,
		logger:           logger,
	}
	return app
}

func initBridgeApp() *BridgeApp {
	consumerAddrs := flag.String("consumer_proxy_addr", "", "Comma separated kafka proxy hosts for message consuming.")
	consumerGroup := flag.String("consumer_group_id", "", "Kafka qroup id used for message consuming.")
	consumerOffset := flag.String("consumer_offset", "", "Kafka read offset.")
	consumerAutoCommitEnable := flag.Bool("consumer_autocommit_enable", false, "Enable autocommit for small messages.")
	consumerAuthorizationKey := flag.String("consumer_authorization_key", "", "The authorization key required to UCS access.")

	topic := flag.String("topic", "", "Kafka topic.")

	producerAddress := flag.String("producer_address", "", "The address the messages are forwarded to.")

	producerVulcanAuth := flag.String("producer_vulcan_auth", "", "Authentication string by which you access cms-notifier via vulcand.")
	producerType := flag.String("producer_type", proxy, "Two possible values are accepted: proxy - if the requests are going through the kafka-proxy; or plainHTTP if a normal http request is required.")
	serviceName := flag.String("service_name", "kafka-bridge", "The full name for the bridge app, like: `cms-kafka-bridge-pub-xp`")

	flag.Parse()

	logConf := log.KeyNamesConfig{KeyTime: "@time"}
	logger := log.NewUPPLogger(*serviceName, "INFO", logConf)
	logger.Info("Starting Kafka Bridge")

	return newBridgeApp(*consumerAddrs, *consumerGroup, *consumerOffset, *consumerAutoCommitEnable, *consumerAuthorizationKey, *topic, *producerAddress, *producerVulcanAuth, *producerType, *serviceName, logger)
}

func (app *BridgeApp) enableHealthchecksAndGTG(serviceName string) {
	hc := NewHealthCheck(app.consumerConfig, app.producerInstance, app.producerType, app.httpClient, app.logger)
	http.HandleFunc("/__health", hc.Health(serviceName))
	http.HandleFunc(httphandlers.GTGPath, httphandlers.NewGoodToGoHandler(hc.GTG))

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		app.logger.WithError(err).Error("Couldn't set up HTTP listener for healthcheck")
	}
}

func main() {
	app := initBridgeApp()
	go app.enableHealthchecksAndGTG(app.serviceName)
	app.consumeMessages()
}
