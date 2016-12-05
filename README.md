###Kafka consumer listening to kafka-proxy and forwarding messages to another KAFKA-PROXY or a simple HTTP endpoint.

* Set the following environment variables:
    * $QUEUE_PROXY_ADDRS
    * $GROUP_ID
    $ CONSUMER_AUTOCOMMIT_ENABLE (enable autocommit when consuming from kafka proxy - use `true` for smaller, `false` for larger messages)
    * $AUTHORIZATION_KEY
    * $TOPIC
    * $PRODUCER_HOST
    * $PRODUCER_HOST_HEADER
    * $PRODUCER_VULCAN_AUTH
    * $PRODUCER_TYPE (possible values: `proxy` or `plainHTTP`)
