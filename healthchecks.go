package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	ftHealth "github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/service-status-go/gtg"
)

func (bridge BridgeApp) consumeHealthcheck() ftHealth.Check {
	return ftHealth.Check{
		BusinessImpact:   "Consuming messages through kafka-proxy won't work. Publishing in the containerised stack won't work.",
		Name:             "Consume from UCS kafka through the proxy",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/kafka-bridge-run-book",
		Severity:         1,
		TechnicalSummary: "Consuming messages is broken. Check if kafka-proxy in aws is reachable.",
		Checker:          bridge.aggregateConsumableResults,
	}
}

func (bridge BridgeApp) proxyForwarderHealthcheck() ftHealth.Check {
	return ftHealth.Check{
		BusinessImpact:   "Forwarding messages to kafka-proxy in coco won't work. Publishing in the containerised stack won't work.",
		Name:             "Forward messages to kafka-proxy.",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/kafka-bridge-run-book",
		Severity:         1,
		TechnicalSummary: "Forwarding messages is broken. Check if kafka-proxy in coco is reachable.",
		Checker:          bridge.producerInstance.ConnectivityCheck,
	}
}

func (bridge BridgeApp) httpForwarderHealthcheck() ftHealth.Check {
	return ftHealth.Check{
		BusinessImpact:   "Forwarding messages to cms-notifier in coco won't work. Publishing in the containerised stack won't work.",
		Name:             "Forward messages to cms-notifier",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/kafka-bridge-run-book",
		Severity:         1,
		TechnicalSummary: "Forwarding messages is broken. Check networking, aws cluster reachability and/or coco cms-notifier state.",
		Checker:          bridge.producerInstance.ConnectivityCheck,
	}
}

func (bridge BridgeApp) gtgCheck() gtg.Status {
	msg, err := bridge.aggregateConsumableResults()
	if err != nil {
		return gtg.Status{GoodToGo: false, Message: msg}
	}

	msg, err = bridge.producerInstance.ConnectivityCheck()
	if err != nil {
		return gtg.Status{GoodToGo: false, Message: msg}
	}

	return gtg.Status{GoodToGo: true}
}

func (bridge BridgeApp) aggregateConsumableResults() (string, error) {
	addresses := bridge.consumerConfig.Addrs
	errMsg := ""
	for i := 0; i < len(addresses); i++ {
		err := bridge.checkConsumable(addresses[i])
		if err == nil {
			return "", nil
		}
		errMsg = errMsg + fmt.Sprintf("For %s there is an error %v \n", addresses[i], err.Error())
	}

	return "Consuming messages is broken.", errors.New(errMsg)
}

func (bridge BridgeApp) checkConsumable(address string) error {
	err := bridge.checkProxyConnection(address, bridge.consumerConfig.AuthorizationKey)
	if err != nil {
		logger.error(fmt.Sprintf("Healthcheck: Error reading request body: %v", err.Error()))
		return err
	}
	return nil
}

func (bridge BridgeApp) checkProxyConnection(address string, authorizationKey string) error {
	//check if proxy is running
	req, err := http.NewRequest("GET", address+"/topics", nil)
	if err != nil {
		logger.error(fmt.Sprintf("Error creating new kafka-proxy healthcheck request: %v", err.Error()))
		return err
	}

	if authorizationKey != "" {
		req.Header.Add("Authorization", authorizationKey)
	}

	resp, err := bridge.httpClient.Do(req)
	if err != nil {
		logger.error(fmt.Sprintf("Healthcheck: Error executing kafka-proxy GET request: %v", err.Error()))
		return err
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Connecting to kafka proxy was not successful. Status: %d", resp.StatusCode)
		return errors.New(errMsg)
	}
	return nil
}
