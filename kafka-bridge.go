/**
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Taken from: https://github.com/stealthly/go_kafka_client/blob/master/consumers/consumers.go
 *
 */

package main

import (
	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
	fthealth "github.com/Financial-Times/go-fthealth"
	"errors"
	"fmt"
	"github.com/dchest/uniuri"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// BridgeApp wraps the config and represents the API for the bridge
type BridgeApp struct {
	consumerConfig *queueConsumer.QueueConfig
	httpClient     *http.Client
	httpHost       string
	httpEndpoint   string
	hostHeader     string
}

const tidValidRegexp = "(tid|SYNTHETIC-REQ-MON)[a-zA-Z0-9_-]*$"
const systemIDValidRegexp = `[a-zA-Z-]*$`

func newBridgeApp(confPath string) (*BridgeApp, int) {
	consumerConfig, host, endpoint, header, numConsumers := ResolveConfig(confPath)
	bridgeApp := &BridgeApp{
		consumerConfig: &consumerConfig,
		httpClient:     &http.Client{},
		httpHost:       host,
		httpEndpoint:   endpoint,
		hostHeader:     header,
	}
	return bridgeApp, numConsumers
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
			go bridge.forwardMsg(m)
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

	req, err := http.NewRequest("POST", "http://" + bridge.httpHost + "/" + bridge.httpEndpoint, strings.NewReader(msg.Body))
	if err != nil {
		logger.error(fmt.Sprintf("Error creating new request: %v", err.Error()))
		return err
	}
	req.Header.Add("X-Origin-System-Id", originSystem)
	req.Header.Add("X-Request-Id", tid)
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
	validRegexp := regexp.MustCompile(systemIDValidRegexp);
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
	validRegexp := regexp.MustCompile(tidValidRegexp);
	tid := validRegexp.FindString(header)
	if tid == "" {
		return "", fmt.Errorf("Transaction ID is in unknown format: %s.", header)
	}
	return tid, nil
}

func main() {
	initLoggers()
	logger.info("Starting Kafka Bridge")
	if len(os.Args) < 2 {
		panic("Conf file path must be provided")
	}
	conf := os.Args[1]
	bridgeApp, numConsumers := newBridgeApp(conf)

	consumers := make([]queueConsumer.MessageIterator, numConsumers)

	go func() {for i := 0; i < numConsumers; i++ {
		consumers[i] = bridgeApp.startNewConsumer()
		bridgeApp.consumeMessages(consumers[i])
		time.Sleep(10 * time.Second)
	}}()

	http.HandleFunc("/__health", fthealth.Handler("Dependent services healthcheck", "Services: cms-notifier@aws, kafka-rest-proxy@aws", bridgeApp.ForwardHealthcheck(), bridgeApp.ConsumeHealthcheck()))
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		logger.error(fmt.Sprintf("Couldn't set up HTTP listener: %+v", err))
	}
}