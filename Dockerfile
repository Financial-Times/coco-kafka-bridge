FROM alpine

ADD *.go /kafka-bridge/
ADD start.sh /

RUN apk update \ 
  && apk add bash \
  && apk add git bzr \
  && apk add go \
  && export GOPATH=/gopath \
  && REPO_PATH="github.com/Financial-Times/coco-kafka-bridge" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && mv /kafka-bridge/* $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get -t \
  && go test \
  && go build \
  && mv coco-kafka-bridge /coco-kafka-bridge \
  && apk del go git bzr \
  && rm -rf $GOPATH /var/cache/apk/* 

ENTRYPOINT [ "/bin/sh", "-c" ]
CMD [ "/start.sh" ]