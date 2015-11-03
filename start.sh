[ ! -z "$QUEUE_PROXY_ADDRS" ] && sed -i "s QUEUE_PROXY_ADDRS $QUEUE_PROXY_ADDRS " kafka-bridge.properties
[ ! -z "$GROUP_ID" ] && sed -i "s GROUP_ID $GROUP_ID " kafka-bridge.properties
[ ! -z "$HTTP_HOST" ] && sed -i "s HTTP_HOST $HTTP_HOST " kafka-bridge.properties
sed -i "s AUTHORIZATION_KEY $AUTHORIZATION_KEY " kafka-bridge.properties

./coco-kafka-bridge kafka-bridge.properties
