FROM golang:1.26.1-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o web-gin-service .

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata gettext
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /app/web-gin-service .
EXPOSE 8080
CMD ["./web-gin-service"]
