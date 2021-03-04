FROM golang:1.15-alpine AS builder
WORKDIR /go/src/github.com/halozheng/zhonghong-mqtt
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app .

FROM alpine:latest
ENV GW_HOST="192.168.1.251"
ENV GW_PORT="80"
ENV GW_USERNAME="admin"
ENV GW_PASSWORD=""
ENV MQTT_HOST="127.0.0.1"
ENV MQTT_PORT="1883"
ENV MQTT_USERNAME=""
ENV MQTT_PASSWORD=""
COPY --from=builder /go/src/github.com/halozheng/zhonghong-mqtt/app /
CMD ["/app"]
