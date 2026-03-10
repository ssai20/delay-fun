# Билдер с полным набором инструментов
FROM golang:1.23-bullseye AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Собираем статически скомпилированный бинарник
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o app ./cmd/api

# Финальный образ - минимальный
FROM debian:bullseye-slim

# Устанавливаем только необходимые TeX пакеты без лишних зависимостей
RUN apt-get update && apt-get install -y --no-install-recommends \
    texlive-latex-base \
    texlive-latex-extra \
    texlive-fonts-recommended \
    texlive-lang-cyrillic \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

WORKDIR /app

# Копируем только бинарник и статику
COPY --from=builder /app/app .
COPY --from=builder /app/static ./static

# Создаем директорию для результатов с правильными правами
RUN mkdir -p /tmp/results && \
    chown -R nobody:nogroup /tmp/results && \
    chmod +x ./app

# Переключаемся на непривилегированного пользователя
USER nobody

EXPOSE 8084

# Healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app/app", "-health"]

CMD ["./app"]