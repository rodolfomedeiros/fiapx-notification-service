FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o notification ./cmd/notification
FROM alpine:3.21
RUN adduser -D -H fiapx
USER fiapx
COPY --from=build /app/notification /usr/local/bin/notification
ENTRYPOINT ["notification"]
