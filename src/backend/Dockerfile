FROM golang:1.24.3 AS builder

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o app main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/app .

ENV PORT=8080
EXPOSE 8080

CMD ["./app"]
