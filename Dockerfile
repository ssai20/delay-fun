FROM golang:1.23-bullseye AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o app ./cmd/api

FROM debian:bullseye-slim
RUN apt-get update && apt-get install -y \
    texlive-latex-base \
    texlive-latex-extra \
    texlive-fonts-recommended \
    texlive-lang-cyrillic \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/app .
# 👇 ВАЖНО: скопировать папку static
COPY --from=builder /app/static ./static

RUN mkdir -p /tmp/results && \
    chmod +x ./app

EXPOSE 10000
CMD ["./app"]