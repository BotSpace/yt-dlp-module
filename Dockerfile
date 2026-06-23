# Multi-stage build — Go binary + yt-dlp runtime.

# ---- Builder ----
FROM golang:1.22-alpine AS builder

WORKDIR /build
ENV GOSUMDB=off

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /module .

# ---- Runtime ----
FROM alpine:3.20

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /module ./module

# yt-dlp (community repo) + ffmpeg + ca-certs. yt-dlp python3'ga bog'liq.
RUN apk --no-cache add ca-certificates yt-dlp ffmpeg

USER appuser

ENV PORT=8100
EXPOSE 8100

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8100/health || exit 1

CMD ["/app/module"]
