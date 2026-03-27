# Multi-stage Dockerfile for Cairn: a self-hosted journaling app.
# Builds a static Go binary and packages it in a lightweight Alpine image.

# === BUILD STAGE ===
# Start from a Go image with the full toolchain for compilation.
FROM golang:1.25 AS builder

# Set the working directory inside the container.
WORKDIR /app

# Copy dependency files first (cached if unchanged).
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code.
COPY . .

# Compile the Go app into a static binary (no C dependencies).
RUN CGO_ENABLED=0 go build -o server ./cmd/server

# === RUN STAGE ===
# Use a tiny Alpine image for the final container (no Go toolchain).
FROM alpine:latest

# Set the working directory.
WORKDIR /app

# Copy only the compiled binary, templates, migrations, and static assets.
COPY --from=builder /app/server .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/internal/notes/templates ./internal/notes/templates
COPY --from=builder /app/internal/bookmarks/templates ./internal/bookmarks/templates
COPY --from=builder /app/internal/todos/templates ./internal/todos/templates
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/static ./static

# Declare the port the app listens on.
EXPOSE 8080

# Command to run the server when the container starts.
CMD ["./server"]
