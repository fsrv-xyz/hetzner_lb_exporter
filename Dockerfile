FROM golang@sha256:cb00c135e0e60590614a68f1361fcafdf0f4d2bec2efa7d11b4eb634d1cbe4ed AS builder
WORKDIR /build
COPY . /build
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /build/exporter cmd/hetzner_lb_exporter/main.go
RUN ls -la /build

FROM alpine:latest@sha256:21dc6063fd678b478f57c0e13f47560d0ea4eeba26dfc947b2a4f81f686b9f45 as alpine
RUN apk add -U --no-cache ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/exporter /bin/exporter
ENTRYPOINT ["/bin/exporter"]