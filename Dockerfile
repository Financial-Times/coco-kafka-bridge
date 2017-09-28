FROM golang:1.7-alpine3.5

ADD *.go /kafka-bridge/
ADD vendor /kafka-bridge/vendor

RUN apk update \
  && apk add git bzr \
  && REPO_PATH="github.com/Financial-Times/coco-kafka-bridge" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && mv /kafka-bridge/* $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get -u github.com/kardianos/govendor \
  && $GOPATH/bin/govendor sync \
  && go build \
  && mv coco-kafka-bridge /coco-kafka-bridge \
  && apk del go git bzr libc-dev \
  && rm -rf $GOPATH /var/cache/apk/* /kafka-bridge

CMD exec /coco-kafka-bridge -consumer_proxy_addr=$QUEUE_PROXY_ADDRS \
                            -consumer_group_id=$GROUP_ID \
                            -consumer_offset=largest \
                            -consumer_autocommit_enable=$CONSUMER_AUTOCOMMIT_ENABLE \
                            -consumer_authorization_key="$AUTHORIZATION_KEY" \
                            -topic=$TOPIC \
                            -producer_address=$PRODUCER_ADDRESS \
                            -producer_vulcan_auth="$PRODUCER_VULCAN_AUTH" \
                            -producer_type=$PRODUCER_TYPE \
                            -service_name=$SERVICE_NAME
