FROM golang:1.24-alpine

WORKDIR /app
COPY . .

RUN go mod download
RUN go build -a -installsuffix cgo -o runner .
#RUN go build -o /runner ./cmd/runner

EXPOSE 8081
CMD ["./runner"]