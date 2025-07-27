FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY subscription-service/go.mod subscription-service/go.sum ./subscription-service/
COPY pkg/observability/go.mod pkg/observability/go.sum ./pkg/observability/

COPY pkg/ ./pkg/

WORKDIR /app/pkg/observability
RUN go mod download

WORKDIR /app/subscription-service
RUN go mod download

COPY subscription-service/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/subscription-service/main .

EXPOSE 8081

CMD ["./main"]
