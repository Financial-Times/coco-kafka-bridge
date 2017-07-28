package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Financial-Times/go-logger"
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
}

const (
	plainHTTP = "plainHTTP"
	proxy     = "proxy"
)

func newBridgeApp(consumerAddrs string, consumerGroupID string, consumerOffset string, consumerAutoCommitEnable bool, consumerAuthorizationKey string, topic string, producerHost string, producerHostHeader string, producerVulcanAuth string, producerType string) *BridgeApp {
	consumerConfig := consumer.QueueConfig{}
	consumerConfig.Addrs = strings.Split(consumerAddrs, ",")
	consumerConfig.Group = consumerGroupID
	consumerConfig.Topic = topic
	consumerConfig.Offset = consumerOffset
	consumerConfig.AuthorizationKey = consumerAuthorizationKey
	consumerConfig.AutoCommitEnable = consumerAutoCommitEnable

	producerConfig := producer.MessageProducerConfig{}
	producerConfig.Addr = producerHost
	producerConfig.Topic = topic
	producerConfig.Queue = producerHostHeader
	producerConfig.Authorization = producerVulcanAuth

	var producerInstance producer.MessageProducer
	switch producerType {
	case proxy:
		producerInstance = producer.NewMessageProducer(producerConfig)
	case plainHTTP:
		producerInstance = newPlainHTTPMessageProducer(producerConfig)
	default:
		logger.FatalEvent(fmt.Sprintf("The provided producer type '%v' is invalid", producerType), nil)
	}

	httpClient := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
			Dial: (&net.Dialer{
				KeepAlive: 30 * time.Second,
			}).Dial,
		}}

	bridgeApp := &BridgeApp{
		consumerConfig:   &consumerConfig,
		producerConfig:   &producerConfig,
		producerInstance: producerInstance,
		producerType:     producerType,
		httpClient:       httpClient,
	}
	return bridgeApp
}

func initBridgeApp() *BridgeApp {

	consumerAddrs := flag.String("consumer_proxy_addr", "", "Comma separated kafka proxy hosts for message consuming.")
	consumerGroup := flag.String("consumer_group_id", "", "Kafka qroup id used for message consuming.")
	consumerOffset := flag.String("consumer_offset", "", "Kafka read offset.")
	consumerAutoCommitEnable := flag.Bool("consumer_autocommit_enable", false, "Enable autocommit for small messages.")
	consumerAuthorizationKey := flag.String("consumer_authorization_key", "", "The authorization key required to UCS access.")

	topic := flag.String("topic", "", "Kafka topic.")

	producerHost := flag.String("producer_host", "", "The host the messages are forwarded to.")
	producerHostHeader := flag.String("producer_host_header", "kafka-proxy", "The host header for the forwarder service (ex: cms-notifier or kafka-proxy).")

	producerVulcanAuth := flag.String("producer_vulcan_auth", "", "Authentication string by which you access cms-notifier via vulcand.")
	producerType := flag.String("producer_type", proxy, "Two possible values are accepted: proxy - if the requests are going through the kafka-proxy; or plainHTTP if a normal http request is required.")
	serviceName := flag.String("service_name", "kafka-bridge", "The full name for the bridge app, like: `cms-kafka-bridge-pub-xp`")

	flag.Parse()

	logger.InitDefaultLogger(*serviceName)
	logger.Infof(nil, "Starting Kafka Bridge")

	return newBridgeApp(*consumerAddrs, *consumerGroup, *consumerOffset, *consumerAutoCommitEnable, *consumerAuthorizationKey, *topic, *producerHost, *producerHostHeader, *producerVulcanAuth, *producerType)
}

func (bridgeApp *BridgeApp) enableHealthchecksAndGTG() {
	var gtgHandler func(http.ResponseWriter, *http.Request)
	hc := NewHealthCheck(bridgeApp.consumerConfig, bridgeApp.producerInstance, bridgeApp.producerType, bridgeApp.httpClient)
	http.HandleFunc("/__health", hc.Health())

	gtgHandler = httphandlers.NewGoodToGoHandler(hc.GTG)
	http.HandleFunc(httphandlers.GTGPath, gtgHandler)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		logger.Errorf(nil, "Couldn't set up HTTP listener for healthcheck: %+v", err)
	}
}

func main() {
	bridgeApp := initBridgeApp()
	go bridgeApp.enableHealthchecksAndGTG()
	bridgeApp.consumeMessages()
}
