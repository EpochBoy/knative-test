# Build stage
FROM docker.io/library/golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download || true

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o knative-test .

# Runtime stage
FROM docker.io/library/alpine:3.21

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/knative-test .

RUN adduser -D -u 65534 appuser
USER appuser

EXPOSE 8080

ENTRYPOINT ["./knative-test"]
