FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o orchestrator .
#RUN CGO_ENABLED=0 GOOS=linux go build -o /orchestrator ./cmd/orchestrator

FROM alpine:3.17
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/orchestrator .

EXPOSE 8000
CMD ["./orchestrator"]