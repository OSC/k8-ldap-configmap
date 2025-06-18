FROM golang:1.24.4-alpine3.22 AS builder
RUN apk update && apk add git make
WORKDIR /go/src/app
COPY . .
ARG VERSION="main"
RUN make build VERSION=${VERSION}

FROM alpine:3.22
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /go/src/app/k8-ldap-configmap .
USER 65534
ENTRYPOINT ["/k8-ldap-configmap"]
