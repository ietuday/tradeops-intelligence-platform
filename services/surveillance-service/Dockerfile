FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM alpine:3.19
WORKDIR /app
RUN adduser -D -g '' appuser
COPY --from=build /out/server /app/server
COPY migrations /app/migrations
USER appuser
EXPOSE 8090
CMD ["/app/server"]
