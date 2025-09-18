# Dict stage
FROM python:3.11-slim as dict-stage
RUN pip install --no-cache-dir pymorphy2-dicts-ru pymorphy2

# Build stage
FROM golang:1.21-alpine AS build-stage
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY proper_verb_server.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o verb-extractor proper_verb_server.go

# Production stage  
FROM alpine:latest
RUN apk --no-cache add ca-certificates wget
WORKDIR /root/
COPY --from=build-stage /app/verb-extractor .
COPY --from=dict-stage /usr/local/lib/python3.11/site-packages/pymorphy2_dicts_ru /opt/pymorphy2_dicts_ru
ENV PYMORPHY2_DICT_PATH=/opt/pymorphy2_dicts_ru/data
RUN adduser -D -s /bin/sh appuser
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1
CMD ["./verb-extractor"]
