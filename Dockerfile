FROM golang:1.26.4

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/reference/dockerfile/#copy
COPY *.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /google-translate-telegram

# Run
CMD ["/google-translate-telegram"]
