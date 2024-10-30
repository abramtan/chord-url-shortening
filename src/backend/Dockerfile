# syntax=docker/dockerfile:1

FROM golang:1.23.2

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
# COPY *.go ./
ADD . ./
# Perhaps optimize the build by adding a .dockerignore
# Consider multi-stage builds to keep the image lightweight, just keep the binary

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /chord-url-shortening ./cmd

EXPOSE 3000 8080

# # Run without build
# ENTRYPOINT go run main.go

# Run
CMD ["/chord-url-shortening"]