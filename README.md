# COCO Kafka Bridge
[![CircleCI](https://circleci.com/gh/Financial-Times/coco-kafka-bridge.svg?style=shield)](https://circleci.com/gh/Financial-Times/coco-kafka-bridge)[![Coverage Status](https://coveralls.io/repos/github/Financial-Times/coco-kafka-bridge/badge.svg)](https://coveralls.io/github/Financial-Times/coco-kafka-bridge)

## Installation

Download the source code, the dependencies and build the binary

```shell
go get github.com/Financial-Times/coco-kafka-bridge
cd $GOPATH/src/github.com/Financial-Times/coco-kafka-bridge
go install
```

To run the unit tests:

```shell
go test ./... -race
```

Run the binary (using the `help` flag to see the available optional arguments):

```shell
coco-kafka-bridge [--help]
```

Options:

```sh
  -consumer_authorization_key string
        The authorization key required to UCS access.
  -consumer_autocommit_enable
        Enable autocommit for small messages.
  -consumer_group_id string
        Kafka qroup id used for message consuming.
  -consumer_offset string
        Kafka read offset. Possible values: "earliest" and "latest"
  -consumer_proxy_addr string
        Comma separated kafka proxy hosts for message consuming.
  -producer_address string
        The address the messages are forwarded to.
  -producer_type string
        Two possible values are accepted: proxy - if the requests are going through the kafka-proxy; or plainHTTP if a normal http request is required. (default "proxy")
  -producer_vulcan_auth string
        Authentication string by which you access cms-notifier via vulcand.
  -service_name cms-kafka-bridge-pub-xp
        The full name for the bridge app, like: cms-kafka-bridge-pub-xp (default "kafka-bridge")
  -topic string
        Kafka topic
```

Environment variables if running by Docker container:
* $QUEUE_PROXY_ADDRS
* $GROUP_ID
* $CONSUMER_AUTOCOMMIT_ENABLE (enable autocommit when consuming from kafka proxy - use `true` for smaller, `false` for larger messages)
* $AUTHORIZATION_KEY
* $TOPIC
* $PRODUCER_ADDRESS
* $PRODUCER_VULCAN_AUTH
* $PRODUCER_TYPE (possible values: `proxy` or `plainHTTP`)
* $SERVICE_NAME
