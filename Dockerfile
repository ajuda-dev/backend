FROM golang:1.24-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/purge ./cmd/purge

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata wget \
    && mkdir -p /var/log/ajudadev \
    && chown -R nobody:nobody /var/log/ajudadev
COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/purge /usr/local/bin/purge
USER nobody
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=20s --retries=5 \
    CMD wget -q -O - http://127.0.0.1:8080/health || exit 1
CMD ["api"]
