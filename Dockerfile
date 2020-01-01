FROM golang:1.13-alpine AS bob

WORKDIR /go/src/screenshot-as-a-service

COPY . .

RUN go build -o /go/bin/server -ldflags="-s -w -X 'main.Version=$(git describe --tags)'" cmd/screenshot-as-a-service/main.go

FROM alpine:3.9

RUN apk add --no-cache chromium ca-certificates

COPY --from=bob /go/bin/server /usr/bin/server

ENTRYPOINT ["/usr/bin/server", "-c", "/usr/bin/chromium-browser"]
