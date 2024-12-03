FROM golang@sha256:2b01164887cecfa796f6bff6787c8c597d0e0e09e0694428fdde4a343303eb60 AS builder
WORKDIR /build
COPY . /build
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /build/exporter cmd/hetzner_lb_exporter/main.go
RUN ls -la /build

FROM alpine:latest@sha256:1e42bbe2508154c9126d48c2b8a75420c3544343bf86fd041fb7527e017a4b4a as alpine
RUN apk add -U --no-cache ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/exporter /bin/exporter
ENTRYPOINT ["/bin/exporter"]