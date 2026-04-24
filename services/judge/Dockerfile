# syntax=docker/dockerfile:1

FROM golang:1.25.9-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/judge ./cmd/judge

FROM alpine:3.22
RUN apk add --no-cache ca-certificates docker-cli
ENV TMPDIR=/tmp/cabugi-judge-workspaces
EXPOSE 8082
COPY --from=build /out/judge /usr/local/bin/judge
ENTRYPOINT ["/usr/local/bin/judge"]
