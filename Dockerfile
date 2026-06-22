FROM golang:1.26.4-alpine3.24 AS builder
RUN apk update && apk add git make
WORKDIR /go/src/app
COPY . .
ARG VERSION="main"
RUN make build VERSION=${VERSION}

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /go/src/app/k8-ldap-configmap .
USER 65534
ENTRYPOINT ["/k8-ldap-configmap"]
