FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /src/main_server .

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /src/main_server ./main
COPY public ./public

RUN mkdir -p storage/logs && chmod +x ./main

EXPOSE 8080

CMD ["./main"]
