FROM golang:1.24 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /playstore-api ./cmd/playstore-api


FROM alpine:3.18 AS runtime
RUN apk add --no-cache ca-certificates
COPY --from=builder /playstore-api /usr/local/bin/playstore-api

ENTRYPOINT ["/usr/local/bin/playstore-api"]