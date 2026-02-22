FROM golang:1.26.0-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go install go.etcd.io/bbolt/cmd/bbolt@latest

FROM alpine

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /server /server
COPY --from=builder /go/bin/bbolt /usr/local/bin/bbolt

EXPOSE 8080

ENTRYPOINT ["/server"]
