# 多服务共用 Dockerfile：build-arg SERVICE=webhttp|history|push|mqtt
ARG GO_VERSION=1.22
FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG SERVICE=webhttp
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags '-s -w' -o /out/service ./backend/cmd/${SERVICE}

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/service /app/service
COPY backend/configs/config.example.yaml /app/configs/config.yaml
ENV WATCHER_ENV=prod
EXPOSE 8080 8082 8083
ENTRYPOINT ["/app/service"]
