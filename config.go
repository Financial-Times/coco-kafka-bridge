package main

import (
	kafkaClient "github.com/stealthly/go_kafka_client"
	"strconv"
	"strings"
	"time"
)

func ResolveConfig(confPath string) (*kafkaClient.ConsumerConfig, string, string, int) {
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

	return consumerConfig, rawConfig["http_host"], rawConfig["topic"], numConsumers
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
