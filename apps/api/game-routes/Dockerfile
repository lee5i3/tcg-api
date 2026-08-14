# Lambda container image for game-routes. Build context is the REPO ROOT:
#   docker build -f apps/api/game-routes/Dockerfile .
# The AWS base image bundles the Runtime Interface Emulator, so running this
# container locally exposes the Lambda invoke API on :8080 (docker-compose
# uses that through tools/local-gateway).
FROM golang:1.26 AS build
ARG TARGETARCH
WORKDIR /src
COPY libs/card-catalog-store/ libs/card-catalog-store/
COPY libs/httpapi/ libs/httpapi/
COPY apps/api/game-routes/ apps/api/game-routes/
WORKDIR /src/apps/api/game-routes
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -tags lambda.norpc -trimpath -ldflags="-s -w" -o /bootstrap .

FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build /bootstrap /var/runtime/bootstrap
CMD ["bootstrap"]
