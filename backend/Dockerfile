FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.21

RUN addgroup -S appgroup && adduser -S appuser -G appgroup \
	&& apk add --no-cache ca-certificates tzdata wget

WORKDIR /app
COPY --from=builder /out/api /usr/local/bin/api

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["api"]
