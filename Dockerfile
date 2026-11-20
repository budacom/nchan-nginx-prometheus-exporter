FROM golang:1.26.11-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG REVISION=unknown
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X github.com/prometheus/common/version.Version=${VERSION} -X github.com/prometheus/common/version.Revision=${REVISION}" \
    -o /nchan-prometheus-exporter .

FROM alpine:3.22
COPY --from=build /nchan-prometheus-exporter /usr/bin/nchan-prometheus-exporter
USER 65534:65534
EXPOSE 9113
ENTRYPOINT ["/usr/bin/nchan-prometheus-exporter"]
