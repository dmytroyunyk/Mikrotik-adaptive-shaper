FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /adaptive-shaper .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /adaptive-shaper /app/adaptive-shaper
COPY configs/config.yaml /app/configs/config.yaml

EXPOSE 9090

ENTRYPOINT ["/app/adaptive-shaper"]