package main

import (
	"errors"
	"flag"
	"fmt"
	fthealth "github.com/Financial-Times/go-fthealth"
	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/dchest/uniuri"
	"net/http"
	"regexp"
	"strings"
)

// BridgeApp wraps the config and represents the API for the bridge
type BridgeApp struct {
	consumerConfig *queueConsumer.QueueConfig
	httpClient     *http.Client
	httpHost       string
	httpEndpoint   string
	hostHeader     string
	vulcanAuth     string
}

const tidValidRegexp = "(tid|SYNTHETIC-REQ-MON)[a-zA-Z0-9_-]*$"
const systemIDValidRegexp = `[a-zA-Z-]*$`

func newBridgeApp(addrs string, groupId string, topic string, offset string, authorizationKey string, httpHost string, httpEndPoint string, hostHeader string, vulcanAuth string) *BridgeApp {
	consumerConfig := queueConsumer.QueueConfig{}
	consumerConfig.Addrs = strings.Split(addrs, ",")
	consumerConfig.Group = groupId
	consumerConfig.Topic = topic
	consumerConfig.Offset = offset
	consumerConfig.AuthorizationKey = authorizationKey

	bridgeApp := &BridgeApp{
		consumerConfig: &consumerConfig,
		httpClient:     &http.Client{},
		httpHost:       httpHost,
		httpEndpoint:   httpEndPoint,
		hostHeader:     hostHeader,
		vulcanAuth:     vulcanAuth,
	}
	return bridgeApp
}

func (bridge BridgeApp) startNewConsumer() queueConsumer.MessageIterator {
	consumerConfig := bridge.consumerConfig
	consumer := queueConsumer.NewIterator(*consumerConfig)
	return consumer
}

func (bridge BridgeApp) consumeMessages(iterator queueConsumer.MessageIterator) {
	for {
		msgs, err := iterator.NextMessages()
		if err != nil {
			logger.warn(fmt.Sprintf("Could not read messages: %s", err.Error()))
			continue
		}
		for _, m := range msgs {
			bridge.forwardMsg(m)
		}
	}
}

func (bridge BridgeApp) forwardMsg(msg queueConsumer.Message) error {

	originSystem, err := extractOriginSystem(msg.Headers)
	if err != nil {
		logger.error(fmt.Sprintf("Error parsing origin system id. Skip forwarding message. Reason: %s", err.Error()))
		return err
	}

	tid, err := extractTID(msg.Headers)
	if err != nil {
		logger.warn(fmt.Sprintf("Couldn't extract transaction id: %s", err.Error()))
		tid = "tid_" + uniuri.NewLen(10) + "_kafka_bridge"
		logger.info("Generating tid: " + tid)
	}

	req, err := http.NewRequest("POST", "http://"+bridge.httpHost+"/"+bridge.httpEndpoint, strings.NewReader(msg.Body))
	if err != nil {
		logger.error(fmt.Sprintf("Error creating new request: %v", err.Error()))
		return err
	}
	req.Header.Add("X-Origin-System-Id", originSystem)
	req.Header.Add("X-Request-Id", tid)
	req.Header.Add("Authorization", bridge.vulcanAuth)
	req.Host = bridge.hostHeader

	ctxlogger := TxCombinedLogger{logger, tid}
	resp, err := bridge.httpClient.Do(req)
	if err != nil {
		ctxlogger.error(fmt.Sprintf("Error executing POST request to the ELB: %v", err.Error()))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Forwarding message with tid: %s is not successful. Status: %d", tid, resp.StatusCode)
		ctxlogger.error(errMsg)
		return errors.New(errMsg)
	}

	ctxlogger.info("Message forwarded")
	return nil
}

func extractOriginSystem(headers map[string]string) (string, error) {
	origSysHeader := headers["Origin-System-Id"]
	validRegexp := regexp.MustCompile(systemIDValidRegexp)
	systemID := validRegexp.FindString(origSysHeader)
	if systemID == "" {
		return "", errors.New("Origin system id is not set.")
	}
	return systemID, nil
}

func extractTID(headers map[string]string) (string, error) {
	header := headers["X-Request-Id"]
	if header == "" {
		return "", errors.New("X-Request-Id header could not be found.")
	}
	validRegexp := regexp.MustCompile(tidValidRegexp)
	tid := validRegexp.FindString(header)
	if tid == "" {
		return "", fmt.Errorf("Transaction ID is in unknown format: %s.", header)
	}
	return tid, nil
}

func initBridgeApp() *BridgeApp {

	addrs := flag.String("queue_proxy_addr", "", "Comma separated kafka proxy hosts.")
	group := flag.String("group_id", "", "Kafka qroup id.")
	topic := flag.String("topic", "", "Kafka topic.")
	offset := flag.String("offset", "", "Kafka read offset.")
	authorizationKey := flag.String("authorization_key", "", "The authorization key required to UCS access.")

	httpHost := flag.String("http_host", "", "The host the messages are forwarded to.")
	httpEndpoint := flag.String("http_endpoint", "notify", "The endpoint the messages are forwarded to.")
	hostHeader := flag.String("host_header", "cms-notifier", "The host header for the forwarder service.")

	vulcanAuth := flag.String("vulcan-auth", "", "Authentication string by which you access cms-notifier via vulcand.")
	flag.Parse()

	return newBridgeApp(*addrs, *group, *topic, *offset, *authorizationKey, *httpHost, *httpEndpoint, *hostHeader, *vulcanAuth)
}

func main() {
	initLoggers()
	logger.info("Starting Kafka Bridge")

	bridgeApp := initBridgeApp()
	go func() {
		consumer := bridgeApp.startNewConsumer()
		bridgeApp.consumeMessages(consumer)
	}()

	http.HandleFunc("/__health", fthealth.Handler("Dependent services healthcheck", "Services: cms-notifier@aws, kafka-rest-proxy@aws", bridgeApp.ForwardHealthcheck(), bridgeApp.ConsumeHealthcheck()))
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		logger.error(fmt.Sprintf("Couldn't set up HTTP listener for healthcheck: %+v", err))
	}
}
