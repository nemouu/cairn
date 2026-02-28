# === BUILD STAGE ===
# Start from a Go image that has the full toolchain
FROM golang:1.25 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy dependency files first (these change rarely, so Docker caches this layer)
COPY go.mod go.sum ./

# Download dependencies (cached if go.mod/go.sum haven't changed)
RUN go mod download

# Copy the rest of your source code
COPY . .

# Compile the Go app into a static binary
RUN CGO_ENABLED=0 go build -o server ./cmd/server


# === RUN STAGE ===
# Start fresh from a tiny image (just Linux, no Go toolchain)
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy only what we need from the build stage
COPY --from=builder /app/server .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/internal/notes/templates ./internal/notes/templates
COPY --from=builder /app/internal/bookmarks/templates ./internal/bookmarks/templates
COPY --from=builder /app/internal/todos/templates ./internal/todos/templates
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/static ./static

# Tell Docker this container listens on port 8080
EXPOSE 8080

# The command to run when the container starts
CMD ["./server"]
