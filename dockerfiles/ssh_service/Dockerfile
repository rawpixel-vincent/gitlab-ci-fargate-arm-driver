ARG GO_VERSION
ARG ALPINE_VERSION

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

WORKDIR /go/src/ssh_service

COPY main.go ./main.go

ENV CGO_ENABLED 0
RUN go build -o /usr/local/bin/ssh_service ./

FROM alpine:${ALPINE_VERSION}

RUN apk add -U bash openssh-server
RUN sed -r 's/root:!/root:*/' -i /etc/shadow

COPY --from=builder /usr/local/bin/ssh_service /usr/local/bin/ssh_service

EXPOSE 8888

CMD ["ssh_service"]
