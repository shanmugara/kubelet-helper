FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kubelet-helper .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates util-linux procps

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/kubelet-helper .

CMD ["./kubelet-helper"]
