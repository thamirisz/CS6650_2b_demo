# Stage 1: build
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o product-service .

# Stage 2: small run image
FROM --platform=linux/amd64 alpine:3.20
WORKDIR /app
COPY --from=build /app/product-service .
ENV GIN_MODE=release
EXPOSE 8080
CMD ["./product-service"]