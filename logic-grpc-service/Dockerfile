FROM golang:1.22-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o logic-grpc-service .

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata gettext
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /app/logic-grpc-service .
COPY --from=builder /app/sql/init.sql ./sql/
EXPOSE 9090
CMD ["./logic-grpc-service"]
