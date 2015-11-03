package main

import (
	queueConsumer "github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/jimlawless/cfg"
	"strconv"
	"io/ioutil"
	"strings"
)

func ResolveConfig(propertyConfPath string, authorizationKeyPath string) (queueConsumer.QueueConfig, string, string, string, int) {

	rawConfig := make(map[string]string)
	err := cfg.Load(propertyConfPath, rawConfig)
	if err != nil {
		panic("Failed to load configuration file")
	}

	consumerConfig := queueConsumer.QueueConfig{}
	consumerConfig.Addrs = strings.Split(rawConfig["queue_proxy_addr"],",")
	consumerConfig.Group, _ = rawConfig["group_id"]
	consumerConfig.Queue, _ = rawConfig["queue"]
	consumerConfig.Topic, _ = rawConfig["topic"]

	numConsumers, _ := strconv.Atoi(rawConfig["num_consumers"])

	authorizationKey := ""
	key, err := ioutil.ReadFile(authorizationKeyPath)
	if err != nil {
		logger.warn("Failed to load authorization file. Header will not be set.")
	} else {
		authorizationKey = string(key[0:len(key)])
	}
    consumerConfig.AuthorizationKey = authorizationKey

	return consumerConfig, rawConfig["http_host"], rawConfig["http_endpoint"], rawConfig["host_header"], numConsumers
}

