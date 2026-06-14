FROM golang:1.22-alpine3.19 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/market-data-service ./cmd/server

FROM alpine:3.19

RUN addgroup -S -g 10001 tradeops && adduser -S -D -H -u 10001 -G tradeops tradeops
WORKDIR /app
COPY --from=build /out/market-data-service /app/market-data-service
COPY --from=build /src/migrations /app/migrations
USER 10001:10001

EXPOSE 8080
ENTRYPOINT ["/app/market-data-service"]
