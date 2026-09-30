# Build Stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build static binary.
# The main package lives in ./cmd, not at the repository root.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -v -o /app/bin/app ./cmd

# Final Lightweight Stage
FROM alpine:latest

# Add ca-certificates in case your app makes HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/bin/app /usr/local/bin/app

# The app reads PORT and falls back to 9080.
EXPOSE 9080

CMD ["/usr/local/bin/app"]