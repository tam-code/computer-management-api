################################################################################
# Build sample-company binary
################################################################################

FROM golang:1.23-alpine AS builder

RUN apk update && apk add --no-cache gcc build-base make

ARG APP_NAME=sample-company
ARG APP_VERSION=dev
ARG BUILD_TIME

COPY ./ /app

WORKDIR /app/cmd/api

ENV GO111MODULE=on
ENV GOSUMDB=off

RUN CGO_ENABLED=1 GOOS=linux go build -mod=readonly \
    -ldflags "-X main.version=${APP_VERSION} -X main.buildTime=${BUILD_TIME}" -a -v -o /app/${APP_NAME}

################################################################################
# Build Docker Image
################################################################################
FROM alpine:3.18

ARG APP_NAME=sample-company
ARG APP_VERSION=dev
ARG BUILD_TIME

LABEL name="${APP_NAME}" version="${APP_VERSION}" buildTime="${BUILD_TIME}"

RUN apk update && apk add --no-cache tar gzip libstdc++

# Copy and rename binary to a fixed name for easier execution
COPY --from=builder /app/${APP_NAME} /app/api

# Copy environment configuration
COPY ./.env /app

RUN chmod +x /app/api && \
    chown 65534:65534 -R /app

USER 65534

WORKDIR /app

ENTRYPOINT ["/app/api"]
