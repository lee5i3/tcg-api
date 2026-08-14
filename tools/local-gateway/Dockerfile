# Local-dev API gateway (see main.go). Build context is the repo root:
#   docker build -f tools/local-gateway/Dockerfile .
FROM golang:1.26 AS build
ARG TARGETARCH
WORKDIR /src
COPY tools/local-gateway/ tools/local-gateway/
WORKDIR /src/tools/local-gateway
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -trimpath -o /gateway .

FROM gcr.io/distroless/static-debian12
COPY --from=build /gateway /gateway
EXPOSE 8080
ENTRYPOINT ["/gateway"]
