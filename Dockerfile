FROM golang:1.15-alpine AS builder
WORKDIR /go/src/github.com/halozheng/zhonghong-mqtt
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=aarh64 go build -o app .

FROM alpine:latest
COPY --from=builder /go/src/github.com/halozheng/zhonghong-mqtt/app /
CMD ["/app"]
