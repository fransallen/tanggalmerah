FROM golang:1.26-trixie AS build
WORKDIR /src
COPY . /src
RUN CGO_ENABLED=0 go build -o app cmd/server/main.go

FROM debian:trixie-slim
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        tzdata \
        curl \
        ca-certificates && \
    rm -rf /var/lib/apt/lists/*
ENV TZ="Asia/Jakarta"
WORKDIR /app
COPY --from=build /src/app /app
COPY . /app
EXPOSE 8080

HEALTHCHECK \
    --interval=5m \
    --timeout=10s \
    --retries=3 \
    --start-period=5s \
    CMD curl --fail "http://localhost:8080/health" || exit 1

CMD ["./app"]
