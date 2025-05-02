# Use an official Golang runtime as a parent image
FROM golang:1.24-alpine

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the rest of the application
COPY . .

# Build the Go app
RUN go build -o gobouncer .

# Expose port 8080
EXPOSE 8080

# Run the application
CMD ["./gobouncer"]
