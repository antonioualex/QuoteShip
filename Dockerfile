# Use the official Golang image to build the application
FROM golang:1.23 as builder

# Set the working directory inside the container
WORKDIR /app

# Copy the go.mod files
COPY go.mod ./

# Copy the entire project into the container
COPY . ./

# Run tests
RUN go test ./... -v

# Build the Go application binary
RUN go build -o quoteship ./cmd

# Use a distroless base image for the final image
FROM gcr.io/distroless/base-debian12:nonroot

# Set the working directory
WORKDIR /

# Copy the binary from the builder stage
COPY --from=builder /app/quoteship /quoteship

# Expose the application port
EXPOSE 3142

# Set the default command to run the service
CMD ["/quoteship"]