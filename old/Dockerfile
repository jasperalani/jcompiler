FROM golang:1.23-alpine

RUN apk add --no-cache docker

WORKDIR /app

# Install air for hot reloading
RUN go install github.com/air-verse/air@latest

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Expose the port
EXPOSE 8080

# Use air for hot reloading
CMD ["air", "-c", ".air.toml"]