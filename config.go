package main

import (
	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/jimlawless/cfg"
	"strconv"
)

func ResolveConfig(confPath string) (queueConsumer.QueueConfig, string, string, string, int) {

	rawConfig := make(map[string]string)
	err := cfg.Load(confPath, rawConfig)
	if err != nil {
		panic("Failed to load configuration file")
	}

	consumerConfig := queueConsumer.QueueConfig{}
	consumerConfig.Addr, _ = rawConfig["queue_proxy_addr"]
	consumerConfig.Group, _ = rawConfig["group_id"]
	consumerConfig.Queue, _ = rawConfig["queue"]
	consumerConfig.Topic, _ = rawConfig["topic"]

	numConsumers, _ := strconv.Atoi(rawConfig["num_consumers"])

	return consumerConfig, rawConfig["http_host"], rawConfig["http_endpoint"], rawConfig["host_header"], numConsumers
}

