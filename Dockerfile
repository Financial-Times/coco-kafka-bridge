FROM golang:1

ENV PROJECT=kafka-bridge

COPY . /${PROJECT}/
WORKDIR /${PROJECT}

RUN LDFLAGS="-s -w" \
  && CGO_ENABLED=0 go build -mod=readonly -a -o /artifacts/${PROJECT} -ldflags="${LDFLAGS}" \
  && echo "Build flags: ${LDFLAGS}"


# Multi-stage build - copy certs and the binary into the image
FROM alpine
WORKDIR /
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /artifacts/* /

CMD /kafka-bridge -consumer_proxy_addr=$QUEUE_PROXY_ADDRS \
                  -consumer_group_id=$GROUP_ID \
                  -consumer_offset=latest \
                  -consumer_autocommit_enable=$CONSUMER_AUTOCOMMIT_ENABLE \
                  -consumer_authorization_key="$AUTHORIZATION_KEY" \
                  -topic=$TOPIC \
                  -producer_address=$PRODUCER_ADDRESS \
                  -producer_vulcan_auth="$PRODUCER_VULCAN_AUTH" \
                  -producer_type=$PRODUCER_TYPE \
                  -service_name=$SERVICE_NAME \
                  -region=$REGION
