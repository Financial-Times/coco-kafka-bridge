package main

import (
	fthealth "github.com/Financial-Times/go-fthealth"
	"errors"
	"fmt"
	"net/http"
	"io/ioutil"
	"encoding/json"
)

func (bridge BridgeApp) ForwardHealthcheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Forwarding messages to cms-notifier in coco won't work. Publishing in the containerised stack won't work.",
		Name:             "Forward messages to cms-notifier",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/kafka-bridge-run-book",
		Severity:         1,
		TechnicalSummary: "Forwarding messages is broken. Check networking, aws cluster reachability and/or coco cms-notifier state.",
		Checker:          bridge.checkForwardable,
	}
}

func (bridge BridgeApp) checkForwardable() error {
	req, err := http.NewRequest("GET", "http://" + bridge.httpHost + "/__health", nil)
	if err != nil {
		logger.error(fmt.Sprintf("Error creating new cms-notifier healthcheck request: %v", err.Error()))
		return err
	}
	req.Host = bridge.hostHeader

	resp, err := bridge.httpClient.Do(req)
	if err != nil {
		logger.warn(fmt.Sprintf("Healthcheck: Error executing cms-notifier GET request: %v", err.Error()))
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Healthcheck: Request to cms-notifer /__health endpoint failed. Status: %d.", resp.StatusCode)
		logger.warn(errMsg)
		return errors.New(errMsg)
	}

	return nil
}

func (bridge BridgeApp) ConsumeHealthcheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Consuming messages through kafka-proxy won't work. Publishing in the containerised stack won't work.",
		Name:             "Consume from UCS kafka through the proxy",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/kafka-bridge-run-book",
		Severity:         1,
		TechnicalSummary: "Consuming messages is broken. Check is kafka-proxy is reachable.",
		Checker:          bridge.aggregateConsumableResults,
	}
}

func (bridge BridgeApp) aggregateConsumableResults() error {
	addresses := bridge.consumerConfig.Addrs
	errMsg := ""
	for i := 0; i < len(addresses); i++ {
		error := bridge.checkConsumable(addresses[i])
		if error == nil {
			return nil
		} else {
			errMsg = errMsg + fmt.Sprintf("For %s there is an error %v \n", addresses[i], error.Error())
		}
	}

	return errors.New(errMsg)
}

func (bridge BridgeApp) checkConsumable(address string) error {
	//check if proxy is running and topic is present
	req, err := http.NewRequest("GET", address + "/topics", nil)
	if err != nil {
		logger.error(fmt.Sprintf("Error creating new kafka-proxy healthcheck request: %v", err.Error()))
		return err
	}

	if bridge.consumerAuthorization != "" {
		req.Header.Add("Authorization", bridge.consumerConfig.AuthorizationKey)
	}

	resp, err := bridge.httpClient.Do(req)
	if err != nil {
		logger.error(fmt.Sprintf("Healthcheck: Error executing kafka-proxy GET request: %v", err.Error()))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Connecting to kafka proxy was not successful. Status: %d", resp.StatusCode)
		return errors.New(errMsg)
	}

	body, err := ioutil.ReadAll(resp.Body)
	return checkIfTopicIsPresent(body, bridge.consumerConfig.Topic)

}

func checkIfTopicIsPresent(body []byte, searchedTopic string) error {
	var topics []string

	err := json.Unmarshal(body, &topics)
	if err != nil {
		return errors.New(fmt.Sprintf("Connection could be established to kafka-proxy, but a parsing error occured and topic could not be found. %v", err.Error()))
	}

	for _, topic := range topics {
		if topic == searchedTopic {
			return nil
		}
	}

	return errors.New("Connection could be established to kafka-proxy, but topic was not found")
}