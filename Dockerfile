# Три стадии: SPA (node) → бинарник (go) → рантайм (alpine).
# Сервис в проде сам отдаёт собранную SPA (WEB_DIST=/web/dist).
#
# Нюанс мульти-репо: go.mod содержит replace на ../linkpulse-contracts,
# поэтому контекст сборки — родительская папка:
#   docker build -f linkpulse-dashboard/Dockerfile .
FROM node:24-alpine AS web
WORKDIR /web
COPY linkpulse-dashboard/web/package.json linkpulse-dashboard/web/package-lock.json ./
RUN npm ci
COPY linkpulse-dashboard/web/ .
RUN npm run build

FROM golang:1.26.5-alpine AS build
WORKDIR /src
COPY linkpulse-contracts/ ../linkpulse-contracts/
COPY linkpulse-dashboard/go.mod linkpulse-dashboard/go.sum ./
RUN go mod download
COPY linkpulse-dashboard/ .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/dashboard ./cmd/dashboard

FROM alpine:3.22
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/dashboard /usr/local/bin/dashboard
COPY --from=web /web/dist /web/dist
ENV WEB_DIST=/web/dist
EXPOSE 8082
ENTRYPOINT ["dashboard"]
