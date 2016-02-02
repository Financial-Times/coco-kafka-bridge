###Kafka consumer listening to kafka-proxy and forwarding messages to another KAFKA-PROXY or a simple HTTP endpoint.

Run: `./start.sh`

* Change parameters in start.sh, or set the following environment variables:
    * $QUEUE_PROXY_ADDRS
    * $GROUP_ID
    * $AUTHORIZATION_KEY
    * $TOPIC
    * $PRODUCER_HOST
    * $PRODUCER_HOST_HEADER
    * $PRODUCER_VULCAN_AUTH
    * $PRODUCER_TYPE (possible values: `proxy` or `plainHTTP`)
