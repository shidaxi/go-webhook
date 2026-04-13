FROM gcr.io/distroless/static-debian12

COPY go-webhook /app/go-webhook
COPY config.yaml /app/configs/config.yaml
COPY rules.yaml /app/configs/rules.yaml

WORKDIR /app
EXPOSE 8080 9090

ENTRYPOINT ["/app/go-webhook"]
CMD ["serve", "--config", "/app/configs/config.yaml"]
