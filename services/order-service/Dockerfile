FROM golang:1.22-alpine3.19 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/order-service ./cmd/server

FROM alpine:3.19

RUN addgroup -S tradeops && adduser -S tradeops -G tradeops
WORKDIR /app
COPY --from=build /out/order-service /app/order-service
COPY --from=build /src/migrations /app/migrations
USER tradeops

EXPOSE 8080
ENTRYPOINT ["/app/order-service"]
