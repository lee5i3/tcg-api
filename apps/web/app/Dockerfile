# Public catalog browser (@tcg/app), SvelteKit static build.
# Build from the repo root: docker build -f apps/web/app/Dockerfile .

FROM node:24-alpine AS build
WORKDIR /repo

# Install workspace dependencies first for better layer caching.
# npm ci needs every workspace's package.json to be present.
COPY package.json package-lock.json ./
COPY libs/api-client/package.json libs/api-client/
COPY libs/design-system/package.json libs/design-system/
COPY apps/web/app/package.json apps/web/app/
COPY apps/web/admin/package.json apps/web/admin/
COPY apps/web/www/package.json apps/web/www/
RUN npm ci

# Build the app (adapter-static output lands in apps/web/app/dist).
COPY libs/api-client/ libs/api-client/
COPY libs/design-system/ libs/design-system/
COPY apps/web/app/ apps/web/app/
RUN npm run build --workspace @tcg/app

FROM nginx:alpine
ENV GATEWAY_URL=http://gateway:8080
COPY apps/web/app/nginx.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /repo/apps/web/app/dist /usr/share/nginx/html
EXPOSE 80
