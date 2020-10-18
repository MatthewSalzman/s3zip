# start from the latest golang base image
FROM golang:alpine


RUN apk update && apk add --no-cache gcc && apk add --no-cache libc-dev && apk add make

# Set the current working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. they will be cached of the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the WORKDIR inisde the container
COPY . .

# Build the Go app
# RUN go build .
RUN go build main.go

# Exporse port 7070
EXPOSE 4005

# Command to run the executable
CMD ["./main"]
