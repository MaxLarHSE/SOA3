FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /tmp/flight-app ./flight-service/cmd

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /tmp/flight-app /app/flight-app

EXPOSE 50051

CMD ["/app/flight-app"]