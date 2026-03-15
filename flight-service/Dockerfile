FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/flight-service-bin ./flight-service/cmd

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/flight-service-bin ./flight-service

EXPOSE 50051

CMD ["./flight-service"]