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
	"github.com/dchest/uniuri"
	metrics "github.com/rcrowley/go-metrics"
	kafkaClient "github.com/stealthly/go_kafka_client"
	_ "log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

type bridge struct {
	consumerConfig *kafkaClient.ConsumerConfig
	httpEndpoint   string
}

func resolveConfig(conf string) (*bridge, string, int) {
	rawConfig, err := kafkaClient.LoadConfiguration(conf)
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

	bridgeConfig := &bridge{}
	bridgeConfig.consumerConfig = consumerConfig
	bridgeConfig.httpEndpoint = rawConfig["http_endpoint"]

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

func main() {
	if len(os.Args) < 2 {
		panic("Conf file path must be provided")
	}
	conf := os.Args[1]

	config, topic, numConsumers := resolveConfig(conf)

	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt)

	consumers := make([]*kafkaClient.Consumer, numConsumers)
	for i := 0; i < numConsumers; i++ {
		consumers[i] = startNewConsumer(*config, topic)
		time.Sleep(10 * time.Second)
	}

	<-ctrlc
	fmt.Println("Shutdown triggered, closing all alive consumers")
	for _, consumer := range consumers {
		<-consumer.Close()
	}
	fmt.Println("Successfully shut down all consumers")
}

func startNewConsumer(bridge bridge, topic string) *kafkaClient.Consumer {
	consumerConfig := bridge.consumerConfig
	consumerConfig.Strategy = getStrategy(consumerConfig.Consumerid, bridge.httpEndpoint)
	consumerConfig.WorkerFailureCallback = failedCallback
	consumerConfig.WorkerFailedAttemptCallback = failedAttemptCallback
	consumer := kafkaClient.NewConsumer(consumerConfig)
	topics := map[string]int{topic: consumerConfig.NumConsumerFetchers}
	go func() {
		consumer.StartStatic(topics)
	}()
	return consumer
}

func getStrategy(consumerID, httpEndpoint string) func(*kafkaClient.Worker, *kafkaClient.Message, kafkaClient.TaskId) kafkaClient.WorkerResult {
	consumeRate := metrics.NewRegisteredMeter(fmt.Sprintf("%s-ConsumeRate", consumerID), metrics.DefaultRegistry)
	return func(_ *kafkaClient.Worker, rawMsg *kafkaClient.Message, id kafkaClient.TaskId) kafkaClient.WorkerResult {
		msg := string(rawMsg.Value)
		kafkaClient.Infof("main", "Got a message: %s", msg)
		consumeRate.Mark(1)

		go func(kafkaMsg string) {
			jsonContent, err := extractJSON(kafkaMsg)
			if err != nil {
				fmt.Printf("Extracting JSON content failed. Skip forwarding message. Reason: %s\n", err.Error())
				return
			}
			client := &http.Client{}
			req, err := http.NewRequest("POST", httpEndpoint, strings.NewReader(jsonContent))

			if err != nil {
				fmt.Printf("Error creating new request: %v\n", err.Error())
				return
			}

			req.Header.Add("X-Origin-System-Id", "methode-web-pub") //TODO: parse this from msg
			req.Header.Add("X-Request-Id", "tid_kafka_bridge_" + uniuri.NewLen(8))
			req.Host = "cms-notifier"
			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("Error: %v\n", err.Error())
				return
			}
			fmt.Printf("\nResponse: %+v\n", resp)
		}(msg)

		return kafkaClient.NewSuccessfulResult(id)
	}
}

func failedCallback(wm *kafkaClient.WorkerManager) kafkaClient.FailedDecision {
	kafkaClient.Info("main", "Failed callback")

	return kafkaClient.DoNotCommitOffsetAndStop
}

func failedAttemptCallback(task *kafkaClient.Task, result kafkaClient.WorkerResult) kafkaClient.FailedDecision {
	kafkaClient.Info("main", "Failed attempt")

	return kafkaClient.CommitOffsetAndContinue
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
