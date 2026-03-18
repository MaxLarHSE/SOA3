FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /tmp/booking-app ./booking-service/cmd

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /tmp/booking-app /app/booking-app

EXPOSE 8080

CMD ["/app/booking-app"]