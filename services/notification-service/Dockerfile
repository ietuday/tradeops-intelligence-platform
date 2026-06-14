FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM alpine:3.19
WORKDIR /app
RUN addgroup -S -g 10001 tradeops && adduser -S -D -H -u 10001 -G tradeops tradeops
COPY --from=build /out/server /app/server
COPY migrations /app/migrations
USER 10001:10001
EXPOSE 8091
CMD ["/app/server"]
