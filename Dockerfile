FROM golang:1.26-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o go-webhook .

FROM gcr.io/distroless/static-debian12

COPY --from=builder /build/go-webhook /app/go-webhook
COPY --from=builder /build/configs /app/configs

WORKDIR /app
EXPOSE 8080 9090

ENTRYPOINT ["/app/go-webhook"]
CMD ["serve", "--config", "/app/configs/config.yaml"]
