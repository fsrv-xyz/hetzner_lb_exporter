FROM golang@sha256:36b4f45d2874905b9e8573b783292629bcb346d0a70d8d7150b6df545234818f AS builder
WORKDIR /build
COPY . /build
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /build/exporter cmd/hetzner_lb_exporter/main.go
RUN ls -la /build

FROM alpine:latest@sha256:51183f2cfa6320055da30872f211093f9ff1d3cf06f39a0bdb212314c5dc7375 as alpine
RUN apk add -U --no-cache ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/exporter /bin/exporter
ENTRYPOINT ["/bin/exporter"]