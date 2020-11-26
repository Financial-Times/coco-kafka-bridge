# COCO Kafka Bridge

[![CircleCI](https://circleci.com/gh/Financial-Times/coco-kafka-bridge.svg?style=shield)](https://circleci.com/gh/Financial-Times/coco-kafka-bridge)
[![Coverage Status](https://coveralls.io/repos/github/Financial-Times/coco-kafka-bridge/badge.svg)](https://coveralls.io/github/Financial-Times/coco-kafka-bridge)

## Kafka consumer listening to kafka-proxy and forwarding messages to another KAFKA-PROXY or a simple HTTP endpoint

Set the following environment variables:

- $QUEUE_PROXY_ADDRS
- $GROUP_ID
- $CONSUMER_OFFSET (default `largest`)
- $CONSUMER_AUTOCOMMIT_ENABLE (enable autocommit when consuming from kafka proxy - use `true` for smaller, `false` for larger messages)
- $AUTHORIZATION_KEY
- $TOPIC
- $PRODUCER_ADDRESS
- $PRODUCER_AUTH
- $PRODUCER_TYPE (possible values: `proxy` or `plainHTTP`)
- $SERVICE_NAME
