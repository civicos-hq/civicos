# syntax=docker/dockerfile:1.7
#
# Citizen web app → static bundle served by nginx on Cloud Run.
# Build context is the REPO ROOT, not apps/web: this is a pnpm workspace and
# the app depends on @civicos/types + @civicos/ui, which turbo builds first.

FROM node:20-alpine AS build
WORKDIR /src

# Husky's prepare script runs on install and needs a git dir, which the build
# context doesn't have. HUSKY=0 skips the hook install.
ENV HUSKY=0
ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0

COPY . .
RUN corepack enable && corepack pnpm install --frozen-lockfile

# Vite inlines env at BUILD time, so the gateway URL has to be present now —
# it cannot be injected when the container starts.
ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL
RUN corepack pnpm --filter @civicos/web... build

FROM nginx:1.27-alpine AS runtime
COPY --from=build /src/apps/web/dist /usr/share/nginx/html
COPY deploy/gcp/nginx-spa.conf /etc/nginx/conf.d/default.conf
EXPOSE 8080
