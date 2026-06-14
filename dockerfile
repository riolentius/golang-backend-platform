# build
FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/api ./cmd/api

# run
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
  addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=build /bin/api /app/api
USER app
EXPOSE 8080
CMD ["/app/api"]