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
	"encoding/json"
	"errors"
	"fmt"
	fthealth "github.com/Financial-Times/go-fthealth"
	"github.com/dchest/uniuri"
	kafkaClient "github.com/stealthly/go_kafka_client"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"
)

// BridgeApp wraps the config and represents the API for the bridge
type BridgeApp struct {
	consumerConfig *kafkaClient.ConsumerConfig
	httpHost       string
	topic          string
}

func newBridgeApp(confPath string) (*BridgeApp, int) {
	consumerConfig, host, topic, numConsumers := ResolveConfig(confPath)
	bridgeApp := &BridgeApp{
		consumerConfig: consumerConfig,
		httpHost:       strings.Trim(host, "/"),
		topic:          topic,
	}
	return bridgeApp, numConsumers
}

func (bridge BridgeApp) startNewConsumer() *kafkaClient.Consumer {
	consumerConfig := bridge.consumerConfig
	consumerConfig.Strategy = bridge.kafkaBridgeStrategy
	consumerConfig.WorkerFailureCallback = failedCallback
	consumerConfig.WorkerFailedAttemptCallback = failedAttemptCallback
	consumer := kafkaClient.NewConsumer(consumerConfig)
	topics := map[string]int{bridge.topic: consumerConfig.NumConsumerFetchers}
	go func() {
		consumer.StartStatic(topics)
	}()
	return consumer
}

func (bridge BridgeApp) kafkaBridgeStrategy(_ *kafkaClient.Worker, rawMsg *kafkaClient.Message, id kafkaClient.TaskId) kafkaClient.WorkerResult {
	msg := string(rawMsg.Value)
	kafkaClient.Infof("main", "Got a message: %s", msg)

	go bridge.forwardMsg(msg)

	return kafkaClient.NewSuccessfulResult(id)
}

func (bridge BridgeApp) forwardMsg(kafkaMsg string) error {
	jsonContent, err := extractJSON(kafkaMsg)
	if err != nil {
		log.Printf("Extracting JSON content failed. Skip forwarding message. Reason: %s\n", err.Error())
		return err
	}
	client := &http.Client{}
	req, err := http.NewRequest("POST", "http://"+bridge.httpHost+"/notify", strings.NewReader(jsonContent))

	if err != nil {
		log.Printf("Error creating new request: %v\n", err.Error())
		return err
	}

	originSystem, err := extractOriginSystem(kafkaMsg)
	if err != nil {
		log.Printf("Error parsing origin system id. Skip forwarding message. Reason: %s", err.Error())
		return err
	}
	tid, err := extractTID(kafkaMsg)

	if err != nil {
		log.Printf("Error parsing transaction id: %v\n", err.Error())
		tid = "tid_" + uniuri.NewLen(10) + "_kafka_bridge"
		log.Printf("Generating tid: " + tid)
	}

	req.Header.Add("X-Origin-System-Id", originSystem)
	req.Header.Add("X-Request-Id", tid)
	req.Host = "cms-notifier"
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error: %v\n", err.Error())
		return err
	}
	log.Printf("\nResponse: %+v\n", resp)
	if resp.StatusCode != http.StatusOK {
		return errors.New("Forwarding message is not successful. Status: " + string(resp.StatusCode))
	}
	return nil
}

func extractJSON(msg string) (jsonContent string, err error) {
	startIndex := strings.Index(msg, "{")
	endIndex := strings.LastIndex(msg, "}")

	if startIndex == -1 || endIndex == -1 {
		return jsonContent, errors.New("Unparseable message.")
	}

	jsonContent = msg[startIndex : endIndex+1]

	var temp map[string]interface{}
	if err = json.Unmarshal([]byte(jsonContent), &temp); err != nil {
		log.Printf("Error: Not valid JSON: %s\n", err.Error())
	}

	return jsonContent, err
}

func extractTID(msg string) (tid string, err error) {
	if !strings.Contains(msg, "X-Request-Id") {
		return tid, errors.New("X-Request-Id header could not be found.")
	}
	if !strings.Contains(msg, "X-Request-Id: tid_") {
		return tid, errors.New("Transaction id is not in expected format.")
	}
	startIndex := strings.Index(msg, "X-Request-Id: tid_") + len("X-Request-Id: ")
	tid = msg[startIndex : startIndex+len("tid_")+10]
	return tid, nil
}

var origSysHeaderRegexp = regexp.MustCompile(`Origin-System-Id:\s[a-zA-Z0-9:/.-]*`)
var systemIDRegexp = regexp.MustCompile(`[a-zA-Z-]*$`)

func extractOriginSystem(msg string) (string, error) {
	origSysHeader := origSysHeaderRegexp.FindString(msg)
	systemID := systemIDRegexp.FindString(origSysHeader)
	if systemID == "" {
		return "", errors.New("Origin system id is not set.")
	}
	return systemID, nil
}

func failedCallback(wm *kafkaClient.WorkerManager) kafkaClient.FailedDecision {
	kafkaClient.Info("main", "Failed callback")

	return kafkaClient.DoNotCommitOffsetAndStop
}

func failedAttemptCallback(task *kafkaClient.Task, result kafkaClient.WorkerResult) kafkaClient.FailedDecision {
	kafkaClient.Info("main", "Failed attempt")

	return kafkaClient.CommitOffsetAndContinue
}

func (bridge BridgeApp) forwardHealthcheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Forwarding messages to coco cluster won't work. Publishing in the containerised stack won't work.",
		Name:             "Forward to aws co-co cluster",
		PanicGuide:       "none",
		Severity:         1,
		TechnicalSummary: "Forwarding messages is broken. Check networking, aws cluster reachability and/or coco cms-notifier state.",
		Checker:          bridge.checkForwardable,
	}
}

func (bridge BridgeApp) checkForwardable() error {
	resp, err := http.Get("http://" + bridge.httpHost + "/health/cms-notifier-1/__health")
	if err != nil {
		log.Printf("Error executing GET request: %v\n", err.Error())
		return err
	}
	log.Printf("\nResponse: %+v\n", resp)
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Request to cms-notifer /__health endpoint failed. Status: %d.", resp.StatusCode)
		return errors.New(errMsg)
	}

	return nil
}

func (bridge BridgeApp) consumeHealthcheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Consuming messages from kafka won't work. Publishing in the containerised stack won't work.",
		Name:             "Consume from kafka",
		PanicGuide:       "none",
		Severity:         1,
		TechnicalSummary: "Consuming messages is broken. Check kafka/zookeeper is reachable.",
		Checker:          bridge.checkConsumable,
	}
}

func (bridge BridgeApp) checkConsumable() error {
	//TODO
	return nil
}

func main() {
	if len(os.Args) < 2 {
		panic("Conf file path must be provided")
	}
	conf := os.Args[1]

	bridgeApp, numConsumers := newBridgeApp(conf)

	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt)

	consumers := make([]*kafkaClient.Consumer, numConsumers)
	for i := 0; i < numConsumers; i++ {
		consumers[i] = bridgeApp.startNewConsumer()
		time.Sleep(10 * time.Second)
	}

	go func() {
		http.HandleFunc("/__health", fthealth.Handler("Dependent services healthcheck", "Services: cms-notifier@aws, kafka-prod@ucs", bridgeApp.forwardHealthcheck(), bridgeApp.consumeHealthcheck()))
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Printf("Couldn't set up HTTP listener: %+v\n", err)
			close(ctrlc)
		}
	}()

	<-ctrlc
	log.Println("Shutdown triggered, closing all alive consumers")
	for _, consumer := range consumers {
		<-consumer.Close()
	}
	log.Println("Successfully shut down all consumers")
}
