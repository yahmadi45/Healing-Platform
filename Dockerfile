FROM node:22-alpine AS web-build
WORKDIR /app/apps/web
COPY apps/web/package.json apps/web/package-lock.json ./
RUN npm ci
COPY apps/web/ ./
RUN npm run build

FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-build /app/apps/web/dist ./internal/httpapi/web
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/sentinelops ./

FROM alpine:3.22
RUN addgroup -S app && adduser -S -G app app && apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/sentinelops /usr/local/bin/sentinelops
USER app
EXPOSE 8080
ENTRYPOINT ["sentinelops"]
