FROM alpine

ADD *.go /coco-kafka-bridge/
ADD kafka-bridge.properties /coco-kafka-bridge/kafka-bridge.properties
ADD authorization.yml /coco-kafka-bridge/authorization.yml
ADD start.sh /

RUN apk add --update bash \
  && apk --update add git bzr \
  && echo "http://dl-4.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories \
  && apk --update add go \
  && export GOPATH=/gopath \
  && REPO_PATH="github.com/Financial-Times/coco-kafka-bridge" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && mv /coco-kafka-bridge/* $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get \
  && go test \
  && go build \
  && mv coco-kafka-bridge /app \
  && mv kafka-bridge.properties /kafka-bridge.properties  \
  && mv authorization.yml /authorization.yml  \
  && apk del go git bzr \
  && rm -rf $GOPATH /var/cache/apk/*

ENTRYPOINT [ "/bin/sh", "-c" ]
CMD [ "/start.sh" ]