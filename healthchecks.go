package main

import (
	fthealth "github.com/Financial-Times/go-fthealth"
	"errors"
	"fmt"
	"net/http"
)

func (bridge BridgeApp) ForwardHealthcheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Forwarding messages to coco cluster won't work. Publishing in the containerised stack won't work.",
		Name:             "Forward to aws co-co cluster",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/kafka-bridge-run-book",
		Severity:         1,
		TechnicalSummary: "Forwarding messages is broken. Check networking, aws cluster reachability and/or coco cms-notifier state.",
		Checker:          bridge.checkForwardable,
	}
}

func (bridge BridgeApp) checkForwardable() error {
	//TODO not really a nice way to check just one of the cms notifier nodes...
	resp, err := bridge.httpClient.Get("http://" + bridge.httpHost + "/health/cms-notifier-1/__health")
	if err != nil {
		logger.warn(fmt.Sprintf("Healthcheck: Error executing GET request: %v", err.Error()))
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
		BusinessImpact:   "Consuming messages from kafka won't work. Publishing in the containerised stack won't work.",
		Name:             "Consume from kafka",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/kafka-bridge-run-book",
		Severity:         1,
		TechnicalSummary: "Consuming messages is broken. Check kafka/zookeeper is reachable.",
		Checker:          bridge.checkConsumable,
	}
}

func (bridge BridgeApp) checkConsumable() error {
	//TODO
	return nil
}
