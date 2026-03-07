FROM golang:1.24.4 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o gin-app ./cmd

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/gin-app .
RUN apk add coreutils
EXPOSE 8080
CMD ["./gin-app"]