FROM golang@sha256:25b05d58b6c4b4e86f5e197dc9e867c4b9e0b4d211dfe741282ed88307a892dc AS builder
WORKDIR /build
COPY . /build
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /build/exporter cmd/hetzner_lb_exporter/main.go
RUN ls -la /build

FROM alpine:latest@sha256:51b67269f354137895d43f3b3d810bfacd3945438e94dc5ac55fdac340352f48 as alpine
RUN apk add -U --no-cache ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/exporter /bin/exporter
ENTRYPOINT ["/bin/exporter"]