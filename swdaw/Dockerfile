# Stage 1: dict-stage — устанавливаем словари
FROM python:3.11-slim AS dict-stage

RUN pip install --no-cache-dir pymorphy2-dicts-ru pymorphy2 && \
    python -c "import pymorphy2_dicts_ru; print('Словари в:', pymorphy2_dicts_ru.get_path())"


# Stage 2: build-stage — компилируем Go-приложение
FROM golang:1.21-alpine AS build-stage

WORKDIR /app
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY proper_verb_server.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o verb-extractor proper_verb_server.go


# Stage 3: production — на базе Python 3.11-slim
FROM python:3.11-slim AS production

# Устанавливаем системные зависимости и pip
RUN apt-get update && \
    apt-get install -y --no-install-recommends wget ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Устанавливаем Python-пакет pymorphy2 (без словарей)
RUN pip install --no-cache-dir pymorphy2

# Создаём непривилегированного пользователя
RUN useradd --no-create-home --shell /bin/sh appuser

WORKDIR /home/appuser

# Копируем бинарник и делаем его исполняемым
COPY --from=build-stage /app/verb-extractor ./
RUN chmod +x verb-extractor

# Копируем словари и назначаем их владельцем appuser
COPY --from=dict-stage \
     /usr/local/lib/python3.11/site-packages/pymorphy2_dicts_ru \
     /opt/pymorphy2_dicts_ru
RUN chown -R appuser:appuser /opt/pymorphy2_dicts_ru

# Устанавливаем переменные окружения
ENV PYMORPHY2_DICT_PATH=/opt/pymorphy2_dicts_ru/data \
    PYTHONUNBUFFERED=1

# Переключаемся на непривилегированного пользователя
USER appuser

# Открываем порт и задаём проверку здоровья
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

CMD ["./verb-extractor"]
