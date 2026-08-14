# Lambda container image for auth-routes. Build context is the REPO ROOT:
#   docker build -f apps/api/auth-routes/Dockerfile .
# The AWS base image bundles the Runtime Interface Emulator, so running this
# container locally exposes the Lambda invoke API on :8080 (docker-compose
# uses that through tools/local-gateway).
FROM golang:1.26 AS build
ARG TARGETARCH
WORKDIR /src
COPY libs/card-catalog-store/ libs/card-catalog-store/
COPY libs/user-accounts/ libs/user-accounts/
COPY libs/httpapi/ libs/httpapi/
COPY apps/api/auth-routes/ apps/api/auth-routes/
WORKDIR /src/apps/api/auth-routes
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -tags lambda.norpc -trimpath -ldflags="-s -w" -o /bootstrap .

FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build /bootstrap /var/runtime/bootstrap
CMD ["bootstrap"]
