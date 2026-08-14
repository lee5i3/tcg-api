# Marketing site (@tcg/www), fully prerendered SvelteKit static build.
# Build from the repo root: docker build -f apps/web/www/Dockerfile .

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

# Where the public app lives. The site is fully prerendered, so this is a
# BUILD-time value baked into every CTA link — pass it per environment:
#   docker build --build-arg PUBLIC_APP_URL=https://<app-domain> ...
ARG PUBLIC_APP_URL=https://app.example.com
ENV PUBLIC_APP_URL=${PUBLIC_APP_URL}

# Build the site (adapter-static output lands in apps/web/www/dist).
COPY libs/design-system/ libs/design-system/
COPY apps/web/www/ apps/web/www/
RUN npm run build --workspace @tcg/www

FROM nginx:alpine
COPY apps/web/www/nginx.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /repo/apps/web/www/dist /usr/share/nginx/html
EXPOSE 80
