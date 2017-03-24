FROM alpine:3.5

ADD *.go /kafka-bridge/

RUN apk update \
  && apk add bash \
  && apk add git bzr \
  && apk add go libc-dev \
  && export GOPATH=/gopath \
  && REPO_PATH="github.com/Financial-Times/coco-kafka-bridge" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && mv /kafka-bridge/* $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get -t \
  && go get -u github.com/kardianos/govendor \
  && $GOPATH/bin/govendor sync \
  && go test \
  && go build \
  && mv coco-kafka-bridge /coco-kafka-bridge \
  && apk del go git bzr libc-dev \
  && rm -rf $GOPATH /var/cache/apk/*

CMD exec ./coco-kafka-bridge -consumer_proxy_addr=$QUEUE_PROXY_ADDRS \
                             -consumer_group_id=$GROUP_ID \
                             -consumer_offset=largest \
                             -consumer_autocommit_enable=$CONSUMER_AUTOCOMMIT_ENABLE \
                             -consumer_authorization_key="$AUTHORIZATION_KEY" \
                             -topic=$TOPIC \
                             -producer_host=$PRODUCER_HOST \
                             -producer_host_header=$PRODUCER_HOST_HEADER \
                             -producer_vulcan_auth="$PRODUCER_VULCAN_AUTH" \
                             -producer_type=$PRODUCER_TYPE
