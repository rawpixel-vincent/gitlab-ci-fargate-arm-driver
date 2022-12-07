ARG GO_VERSION
ARG ALPINE_VERSION

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION}

RUN apk add --no-cache make git py-pip bash curl && \
    pip install awscli

