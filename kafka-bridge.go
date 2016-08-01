package main

import (
	"flag"
	"fmt"
	fthealth "github.com/Financial-Times/go-fthealth"
	queueProducer "github.com/Financial-Times/message-queue-go-producer/producer"
	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
	"net/http"
	"strings"
	"os"
	"log"
)

// BridgeApp wraps the config and represents the API for the bridge
type BridgeApp struct {
	consumerConfig   *queueConsumer.QueueConfig
	producerConfig   *queueProducer.MessageProducerConfig
	producerInstance queueProducer.MessageProducer
	producerType     string
}

const (
	plainHTTP = "plainHTTP"
	proxy     = "proxy"
)

func newBridgeApp(consumerAddrs string, consumerGroupID string, consumerOffset string, consumerAutoCommitEnable bool, consumerAuthorizationKey string, topic string, producerHost string, producerHostHeader string, producerVulcanAuth string, producerType string) *BridgeApp {
	consumerConfig := queueConsumer.QueueConfig{}
	consumerConfig.Addrs = strings.Split(consumerAddrs, ",")
	consumerConfig.Group = consumerGroupID
	consumerConfig.Topic = topic
	consumerConfig.Offset = consumerOffset
	consumerConfig.AuthorizationKey = consumerAuthorizationKey
	consumerConfig.AutoCommitEnable = consumerAutoCommitEnable

	producerConfig := queueProducer.MessageProducerConfig{}
	producerConfig.Addr = producerHost
	producerConfig.Topic = topic
	producerConfig.Queue = producerHostHeader
	producerConfig.Authorization = producerVulcanAuth

	var producerInstance queueProducer.MessageProducer
	if producerType == proxy {
		producerInstance = queueProducer.NewMessageProducer(producerConfig)
	} else if producerType == plainHTTP {
		producerInstance = newPlainHTTPMessageProducer(producerConfig)
	}

	bridgeApp := &BridgeApp{
		consumerConfig:   &consumerConfig,
		producerConfig:   &producerConfig,
		producerInstance: producerInstance,
		producerType:     producerType,
	}
	return bridgeApp
}

func initBridgeApp() (bridgeApp *BridgeApp, group string) {

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

	flag.Parse()

	return newBridgeApp(*consumerAddrs, *consumerGroup, *consumerOffset, *consumerAutoCommitEnable, *consumerAuthorizationKey, *topic, *producerHost, *producerHostHeader, *producerVulcanAuth, *producerType), *consumerGroup
}

func (bridgeApp *BridgeApp) enableHealthchecks() {
	//create healthcheck service according to the producer type
	if bridgeApp.producerType == proxy {
		http.HandleFunc("/__health", fthealth.Handler("Dependent services healthcheck", "Services: kafka-rest-proxy@ucs, kafka-rest-proxy@aws", bridgeApp.consumeHealthcheck(), bridgeApp.proxyForwarderHealthcheck()))
	} else if bridgeApp.producerType == plainHTTP {
		http.HandleFunc("/__health", fthealth.Handler("Dependent services healthcheck", "Services: kafka-rest-proxy@ucs, cms-notifier@aws", bridgeApp.consumeHealthcheck(), bridgeApp.httpForwarderHealthcheck()))
	}

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		logger.error(fmt.Sprintf("Couldn't set up HTTP listener for healthcheck: %+v", err))
	}
}

func main() {
	initLoggers()

	bridgeApp, group := initBridgeApp()
	fmt.Printf("group is %v\n", group)

	//dont log to file in any containerized environment
	if strings.Contains(group, "ucs") {
		f, err := os.OpenFile("/var/log/apps/coco-kafka-bridge-app.log", os.O_WRONLY | os.O_CREATE | os.O_APPEND, 0755)
		fmt.Printf("Error is %v\n", err)
		if err == nil {
			log.SetOutput(f)
		} else {
			log.Fatalf("Failed to initialise log file, %v", err)
		}
		defer f.Close()
	}
	logger.info("Starting Kafka Bridge\n")

	go bridgeApp.enableHealthchecks()

	bridgeApp.consumeMessages()
}
