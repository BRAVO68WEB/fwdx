FROM golang:1.24.0-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 go build -o app .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/app /usr/local/bin/app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/app"]
