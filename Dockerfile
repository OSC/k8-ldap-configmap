FROM golang:1.16-alpine AS builder
RUN apk update && apk add git make
WORKDIR /go/src/app
COPY . ./
ARG VERSION="main"
RUN make build VERSION=${VERSION}

FROM alpine:3.12
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /go/src/app/k8-ldap-configmap .
USER 65534
ENTRYPOINT ["/k8-ldap-configmap"]
