package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	logger "github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/service-status-go/httphandlers"

	cli "github.com/jawher/mow.cli"
)

// BridgeApp wraps the config and represents the API for the bridge
type BridgeApp struct {
	consumerConfig   *consumer.QueueConfig
	producerConfig   *producer.MessageProducerConfig
	producerInstance producer.MessageProducer
	producerType     string
	httpClient       *http.Client
	serviceName      string
}

const (
	plainHTTP = "plainHTTP"
	proxy     = "proxy"
)

func newBridgeApp(consumerAddrs string, consumerGroupID string, consumerOffset string, consumerAutoCommitEnable bool, consumerAuthorizationKey string, topic string, producerAddress string, producerAuth string, producerType string, serviceName string, producerEncoding int) *BridgeApp {
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
	producerConfig.Authorization = producerAuth

	var producerInstance producer.MessageProducer
	switch producerType {
	case proxy:
		producerInstance = producer.NewMessageProducerWithEncoder(producerConfig, producer.NewEncoder(producerEncoding))
	case plainHTTP:
		producerInstance = newPlainHTTPMessageProducer(producerConfig)
	default:
		logger.Fatalf(nil, fmt.Errorf("unknown producer type %s", producerType), "The provided producer type '%v' is invalid", producerType)
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
		serviceName:      serviceName,
	}
	return bridgeApp
}

func (bridgeApp *BridgeApp) enableHealthchecksAndGTG(serviceName string) {
	hc := NewHealthCheck(bridgeApp.consumerConfig, bridgeApp.producerInstance, bridgeApp.producerType, bridgeApp.httpClient)
	http.HandleFunc("/__health", hc.Health(serviceName))
	http.HandleFunc(httphandlers.GTGPath, httphandlers.NewGoodToGoHandler(hc.GTG))

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		logger.Errorf(nil, err, "Couldn't set up HTTP listener for healthcheck")
	}
}

func main() {
	appDescription := "The purpose of the Kafka Bridge is to replicate (bridge) messages from one UPP Kubernetes cluster to another."
	appName := "kafka-bridge"

	app := cli.App(appName, appDescription)

	consumerAddrs := app.String(cli.StringOpt{
		Name:   "consumer_proxy_addr",
		Value:  "",
		Desc:   "Comma separated kafka proxy hosts for message consuming.",
		EnvVar: "QUEUE_PROXY_ADDRS",
	})
	consumerGroup := app.String(cli.StringOpt{
		Name:   "consumer_group_id",
		Value:  "",
		Desc:   "Kafka group id used for message consuming.",
		EnvVar: "GROUP_ID",
	})
	consumerOffset := app.String(cli.StringOpt{
		Name:   "consumer_offset",
		Value:  "largest",
		Desc:   "Kafka read offset.",
		EnvVar: "CONSUMER_OFFSET",
	})
	consumerAutoCommitEnable := app.Bool(cli.BoolOpt{
		Name:   "consumer_autocommit_enable",
		Value:  false,
		Desc:   "Enable autocommit for small messages.",
		EnvVar: "CONSUMER_AUTOCOMMIT_ENABLE",
	})
	consumerAuthorizationKey := app.String(cli.StringOpt{
		Name:   "consumer_authorization_key",
		Value:  "",
		Desc:   "The authorization key required to UCS access.",
		EnvVar: "AUTHORIZATION_KEY",
	})
	topic := app.String(cli.StringOpt{
		Name:   "topic",
		Value:  "",
		Desc:   "Kafka topic.",
		EnvVar: "TOPIC",
	})
	producerAddress := app.String(cli.StringOpt{
		Name:   "producer_address",
		Value:  "",
		Desc:   "The address the messages are forwarded to.",
		EnvVar: "PRODUCER_ADDRESS",
	})
	producerAuth := app.String(cli.StringOpt{
		Name:   "producer_auth",
		Value:  "",
		Desc:   "Producer authentication string.",
		EnvVar: "PRODUCER_AUTH",
	})
	producerType := app.String(cli.StringOpt{
		Name:   "producer_type",
		Value:  proxy,
		Desc:   "Two possible values are accepted: proxy - if the requests are going through the kafka-proxy; or plainHTTP if a normal http request is required.",
		EnvVar: "PRODUCER_TYPE",
	})
	serviceName := app.String(cli.StringOpt{
		Name:   "service_name",
		Value:  appName,
		Desc:   "The full name for the bridge app, like: `cms-kafka-bridge-pub-xp`",
		EnvVar: "SERVICE_NAME",
	})

	logger.InitDefaultLogger(*serviceName)
	argument2Encoding := map[string]int{
		"base64": producer.Base64E,
		"model":  producer.CombinedModelE,
	}
	producerEncoding := app.String(cli.StringOpt{
		Name:   "producer_encoding",
		Value:  reflect.ValueOf(argument2Encoding).MapKeys()[0].String(),
		Desc:   "Two possible values are accepted: model - if you want to send data encoded in a json mapping the model; or base64 if you want a base64 encoding.",
		EnvVar: "PRODUCER_ENCODING",
	})
	if _, ok := argument2Encoding[*producerEncoding]; !ok {
		var options []string
		for k := range argument2Encoding {
			options = append(options, k)
		}
		logger.Fatalf(map[string]interface{}{"valid options": options}, errors.New("producer_encoding"), "invalid argument")
	}
	logger.Infof(nil, "Starting Kafka Bridge")

	app.Action = func() {
		bridgeApp := newBridgeApp(
			*consumerAddrs,
			*consumerGroup,
			*consumerOffset,
			*consumerAutoCommitEnable,
			*consumerAuthorizationKey,
			*topic,
			*producerAddress,
			*producerAuth,
			*producerType,
			*serviceName,
			argument2Encoding[*producerEncoding],
		)
		go bridgeApp.enableHealthchecksAndGTG(bridgeApp.serviceName)
		bridgeApp.consumeMessages()
	}

	err := app.Run(os.Args)
	if err != nil {
		logger.Errorf(nil, err, "App could not start")
		return
	}
}
