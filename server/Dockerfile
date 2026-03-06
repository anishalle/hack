FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /hack-server ./server/cmd/server

FROM gcr.io/distroless/static-debian12

COPY --from=builder /hack-server /hack-server

EXPOSE 8080

ENTRYPOINT ["/hack-server"]
