FROM golang@sha256:205c7636d74e0ffbe3d007162df7d74f859f5a8c8322dfb9f907977686c126cf AS builder
WORKDIR /build
COPY . /build
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /build/exporter cmd/hetzner_lb_exporter/main.go
RUN ls -la /build

FROM alpine:latest@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf as alpine
RUN apk add -U --no-cache ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/exporter /bin/exporter
ENTRYPOINT ["/bin/exporter"]