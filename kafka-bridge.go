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
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// BridgeApp wraps the config and represents the API for the bridge
type BridgeApp struct {
	consumerConfig *kafkaClient.ConsumerConfig
	httpEndpoint   string
}

func resolveConfig(confPath string) (*BridgeApp, string, int) {
	rawConfig, err := kafkaClient.LoadConfiguration(confPath)
	if err != nil {
		panic("Failed to load configuration file")
	}
	logLevel := rawConfig["log_level"]
	setLogLevel(logLevel)
	numConsumers, _ := strconv.Atoi(rawConfig["num_consumers"])
	zkTimeout, _ := time.ParseDuration(rawConfig["zookeeper_timeout"])

	numWorkers, _ := strconv.Atoi(rawConfig["num_workers"])
	maxWorkerRetries, _ := strconv.Atoi(rawConfig["max_worker_retries"])
	workerBackoff, _ := time.ParseDuration(rawConfig["worker_backoff"])
	workerRetryThreshold, _ := strconv.Atoi(rawConfig["worker_retry_threshold"])
	workerConsideredFailedTimeWindow, _ := time.ParseDuration(rawConfig["worker_considered_failed_time_window"])
	workerTaskTimeout, _ := time.ParseDuration(rawConfig["worker_task_timeout"])
	workerManagersStopTimeout, _ := time.ParseDuration(rawConfig["worker_managers_stop_timeout"])

	rebalanceBarrierTimeout, _ := time.ParseDuration(rawConfig["rebalance_barrier_timeout"])
	rebalanceMaxRetries, _ := strconv.Atoi(rawConfig["rebalance_max_retries"])
	rebalanceBackoff, _ := time.ParseDuration(rawConfig["rebalance_backoff"])
	partitionAssignmentStrategy, _ := rawConfig["partition_assignment_strategy"]
	excludeInternalTopics, _ := strconv.ParseBool(rawConfig["exclude_internal_topics"])

	numConsumerFetchers, _ := strconv.Atoi(rawConfig["num_consumer_fetchers"])
	fetchBatchSize, _ := strconv.Atoi(rawConfig["fetch_batch_size"])
	fetchMessageMaxBytes, _ := strconv.Atoi(rawConfig["fetch_message_max_bytes"])
	fetchMinBytes, _ := strconv.Atoi(rawConfig["fetch_min_bytes"])
	fetchBatchTimeout, _ := time.ParseDuration(rawConfig["fetch_batch_timeout"])
	requeueAskNextBackoff, _ := time.ParseDuration(rawConfig["requeue_ask_next_backoff"])
	fetchWaitMaxMs, _ := strconv.Atoi(rawConfig["fetch_wait_max_ms"])
	socketTimeout, _ := time.ParseDuration(rawConfig["socket_timeout"])
	queuedMaxMessages, _ := strconv.Atoi(rawConfig["queued_max_messages"])
	refreshLeaderBackoff, _ := time.ParseDuration(rawConfig["refresh_leader_backoff"])
	fetchMetadataRetries, _ := strconv.Atoi(rawConfig["fetch_metadata_retries"])
	fetchMetadataBackoff, _ := time.ParseDuration(rawConfig["fetch_metadata_backoff"])

	time.ParseDuration(rawConfig["fetch_metadata_backoff"])

	offsetsCommitMaxRetries, _ := strconv.Atoi(rawConfig["offsets_commit_max_retries"])

	deploymentTimeout, _ := time.ParseDuration(rawConfig["deployment_timeout"])

	zkConfig := kafkaClient.NewZookeeperConfig()
	zkConfig.ZookeeperConnect = strings.Split(rawConfig["zookeeper_connect"], ",")
	zkConfig.ZookeeperTimeout = zkTimeout

	consumerConfig := kafkaClient.DefaultConsumerConfig()
	consumerConfig.Groupid = rawConfig["group_id"]
	consumerConfig.NumWorkers = numWorkers
	consumerConfig.MaxWorkerRetries = maxWorkerRetries
	consumerConfig.WorkerBackoff = workerBackoff
	consumerConfig.WorkerRetryThreshold = int32(workerRetryThreshold)
	consumerConfig.WorkerThresholdTimeWindow = workerConsideredFailedTimeWindow
	consumerConfig.WorkerTaskTimeout = workerTaskTimeout
	consumerConfig.WorkerManagersStopTimeout = workerManagersStopTimeout
	consumerConfig.BarrierTimeout = rebalanceBarrierTimeout
	consumerConfig.RebalanceMaxRetries = int32(rebalanceMaxRetries)
	consumerConfig.RebalanceBackoff = rebalanceBackoff
	consumerConfig.PartitionAssignmentStrategy = partitionAssignmentStrategy
	consumerConfig.ExcludeInternalTopics = excludeInternalTopics
	consumerConfig.NumConsumerFetchers = numConsumerFetchers
	consumerConfig.FetchBatchSize = fetchBatchSize
	consumerConfig.FetchMessageMaxBytes = int32(fetchMessageMaxBytes)
	consumerConfig.FetchMinBytes = int32(fetchMinBytes)
	consumerConfig.FetchBatchTimeout = fetchBatchTimeout
	consumerConfig.FetchTopicMetadataRetries = fetchMetadataRetries
	consumerConfig.FetchTopicMetadataBackoff = fetchMetadataBackoff
	consumerConfig.RequeueAskNextBackoff = requeueAskNextBackoff
	consumerConfig.FetchWaitMaxMs = int32(fetchWaitMaxMs)
	consumerConfig.SocketTimeout = socketTimeout
	consumerConfig.QueuedMaxMessages = int32(queuedMaxMessages)
	consumerConfig.RefreshLeaderBackoff = refreshLeaderBackoff
	consumerConfig.Coordinator = kafkaClient.NewZookeeperCoordinator(zkConfig)
	consumerConfig.AutoOffsetReset = rawConfig["auto_offset_reset"]
	consumerConfig.OffsetsCommitMaxRetries = offsetsCommitMaxRetries
	consumerConfig.DeploymentTimeout = deploymentTimeout
	consumerConfig.OffsetCommitInterval = 10 * time.Second

	bridgeConfig := &BridgeApp{}
	bridgeConfig.consumerConfig = consumerConfig
	bridgeConfig.httpEndpoint = buildHTTPEndpoint(rawConfig["http_host"])

	return bridgeConfig, rawConfig["topic"], numConsumers
}

func setLogLevel(logLevel string) {
	var level kafkaClient.LogLevel
	switch strings.ToLower(logLevel) {
	case "trace":
		level = kafkaClient.TraceLevel
	case "debug":
		level = kafkaClient.DebugLevel
	case "info":
		level = kafkaClient.InfoLevel
	case "warn":
		level = kafkaClient.WarnLevel
	case "error":
		level = kafkaClient.ErrorLevel
	case "critical":
		level = kafkaClient.CriticalLevel
	default:
		level = kafkaClient.InfoLevel
	}
	kafkaClient.Logger = kafkaClient.NewDefaultLogger(level)
}

func buildHTTPEndpoint(host string) string {
	return "http://" + strings.Trim(host, "/") + "/notify"
}

func (bridge BridgeApp) startNewConsumer(topic string) *kafkaClient.Consumer {
	consumerConfig := bridge.consumerConfig
	consumerConfig.Strategy = bridge.kafkaBridgeStrategy
	consumerConfig.WorkerFailureCallback = failedCallback
	consumerConfig.WorkerFailedAttemptCallback = failedAttemptCallback
	consumer := kafkaClient.NewConsumer(consumerConfig)
	topics := map[string]int{topic: consumerConfig.NumConsumerFetchers}
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
		fmt.Printf("Extracting JSON content failed. Skip forwarding message. Reason: %s\n", err.Error())
		return err
	}
	client := &http.Client{}
	req, err := http.NewRequest("POST", bridge.httpEndpoint, strings.NewReader(jsonContent))

	if err != nil {
		fmt.Printf("Error creating new request: %v\n", err.Error())
		return err
	}

	originSystem, err := extractOriginSystem(kafkaMsg)
	if err != nil {
		fmt.Printf("Error parsing origin system id. Skip forwarding message. Reason: %s", err.Error())
		return err
	}
	tid, err := extractTID(kafkaMsg)

	if err != nil {
		fmt.Printf("Error parsing transaction id: %v\n", err.Error())
		tid = "tid_" + uniuri.NewLen(10) + "_kafka_bridge"
		fmt.Printf("Generating tid: " + tid)
	}

	req.Header.Add("X-Origin-System-Id", originSystem)
	req.Header.Add("X-Request-Id", tid)
	req.Host = "cms-notifier"
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err.Error())
		return err
	}
	fmt.Printf("\nResponse: %+v\n", resp)
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
		fmt.Printf("Error: Not valid JSON: %s\n", err.Error())
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

func extractOriginSystem(msg string) (string, error) {
	origSysHeaderRegexp := regexp.MustCompile(`Origin-System-Id:\s[a-zA-Z0-9:/.-]*`)
	origSysHeader := origSysHeaderRegexp.FindString(msg)
	systemIDRegexp := regexp.MustCompile(`[a-zA-Z-]*$`)
	systemID := systemIDRegexp.FindString(origSysHeader)
    if (systemID == "") {
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
	return bridge.forwardMsg("{}")
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

	bridgeApp, topic, numConsumers := resolveConfig(conf)

	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt)

	consumers := make([]*kafkaClient.Consumer, numConsumers)
	for i := 0; i < numConsumers; i++ {
		consumers[i] = bridgeApp.startNewConsumer(topic)
		time.Sleep(10 * time.Second)
	}

	go func() {
		http.HandleFunc("/__health", fthealth.Handler("Dependent services healthcheck", "Services: cms-notifier@aws, kafka-prod@ucs", bridgeApp.forwardHealthcheck(), bridgeApp.consumeHealthcheck()))
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			fmt.Printf("Couldn't set up HTTP listener: %+v\n", err)
			close(ctrlc)
		}
	}()

	<-ctrlc
	fmt.Println("Shutdown triggered, closing all alive consumers")
	for _, consumer := range consumers {
		<-consumer.Close()
	}
	fmt.Println("Successfully shut down all consumers")
}
