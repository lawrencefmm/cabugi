# syntax=docker/dockerfile:1

FROM golang:1.25.9-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed-starter-problems ./cmd/seed-starter-problems

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && addgroup -S cabugi && adduser -S -G cabugi cabugi
COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/seed-starter-problems /usr/local/bin/seed-starter-problems
USER cabugi
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
