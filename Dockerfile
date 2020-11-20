FROM golang:1

ENV PROJECT=coco-kafka-bridge
ENV BUILDINFO_PACKAGE="github.com/Financial-Times/service-status-go/buildinfo."

COPY . ${PROJECT}
WORKDIR ${PROJECT}

RUN VERSION="version=$(git describe --tag --always 2> /dev/null)" \
  && DATETIME="dateTime=$(date -u +%Y%m%d%H%M%S)" \
  && REPOSITORY="repository=$(git config --get remote.origin.url)" \
  && REVISION="revision=$(git rev-parse HEAD)" \
  && BUILDER="builder=$(go version)" \
  && LDFLAGS="-X '"${BUILDINFO_PACKAGE}$VERSION"' -X '"${BUILDINFO_PACKAGE}$DATETIME"' -X '"${BUILDINFO_PACKAGE}$REPOSITORY"' -X '"${BUILDINFO_PACKAGE}$REVISION"' -X '"${BUILDINFO_PACKAGE}$BUILDER"'" \
  && echo "Build flags: $LDFLAGS" \
  && CGO_ENABLED=0 go build -mod=readonly -o /artifacts/${PROJECT} -ldflags="${LDFLAGS}"

# Multi-stage build - copy certs and the binary into the image
FROM scratch
WORKDIR /
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /artifacts/* /

# TL;TR : The service must be adapted to load ENV VAR within the binary instead of being passed due the lack of support and tools from scratch image
#
# The service have been migrated successfuly to Go Modules and Orb config but as it's it cann't be deployed to production
# due the way the golang program loads its variables from the system:
# with the current Dockerfile configuration - our standard- it cannot log in any way the variable that have been set from the system
# due the lack of any sys programs especially /bin/sh in the scratch image, so CMD and ENTRYPOINT commands are useless.
# Therefore the only way to load ENV VAR is from within the provided golang binary, The service must be adapted to load variables and
#  not be passed as parameters, just as any other go service.
# To be deleted
#CMD exec /coco-kafka-bridge -consumer_proxy_addr=$QUEUE_PROXY_ADDRS \
#                            -consumer_group_id=$GROUP_ID \
#                            -consumer_offset=largest \
#                            -consumer_autocommit_enable=$CONSUMER_AUTOCOMMIT_ENABLE \
#                            -consumer_authorization_key="$AUTHORIZATION_KEY" \
#                            -topic=$TOPIC \
#                            -producer_address=$PRODUCER_ADDRESS \
#                            -producer_vulcan_auth="$PRODUCER_VULCAN_AUTH" \
#                            -producer_type=$PRODUCER_TYPE \
#                            -service_name=$SERVICE_NAME
