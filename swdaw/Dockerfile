# Multistage build для оптимизации размера образа
FROM python:3.11-slim as dict-stage

# Установка словарей pymorphy2
RUN pip install --no-cache-dir pymorphy2-dicts-ru pymorphy2 && \
    python -c "import pymorphy2_dicts_ru; print('Словари установлены в:', pymorphy2_dicts_ru.get_path())"

# Go build stage
FROM golang:1.21-alpine AS build-stage

WORKDIR /app

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY *.go ./

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o verb-extractor proper_verb_server_fixed.go

# Production stage
FROM alpine:latest

# Установка необходимых пакетов
RUN apk --no-cache add ca-certificates python3 py3-pip

WORKDIR /root/

# Копируем бинарник из build stage
COPY --from=build-stage /app/verb-extractor .

# Копируем словари из dict-stage
COPY --from=dict-stage /usr/local/lib/python3.11/site-packages/pymorphy2_dicts_ru /opt/pymorphy2_dicts_ru

# Устанавливаем переменную окружения для словарей
ENV PYMORPHY2_DICT_PATH=/opt/pymorphy2_dicts_ru/data

# Создаем пользователя без привилегий
RUN adduser -D -s /bin/sh appuser
USER appuser

# Порт приложения
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

# Запуск приложения
CMD ["./verb-extractor"]
