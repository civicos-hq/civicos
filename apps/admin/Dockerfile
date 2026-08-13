# syntax=docker/dockerfile:1.7
#
# Admin console → static bundle served by nginx on Cloud Run.
# Build context is the REPO ROOT (pnpm workspace — see apps/web/Dockerfile).

FROM node:20-alpine AS build
WORKDIR /src

ENV HUSKY=0
ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0

COPY . .
RUN corepack enable && corepack pnpm install --frozen-lockfile

ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL
RUN corepack pnpm --filter @civicos/admin... build

FROM nginx:1.27-alpine AS runtime
COPY --from=build /src/apps/admin/dist /usr/share/nginx/html
COPY deploy/gcp/nginx-spa.conf /etc/nginx/conf.d/default.conf
EXPOSE 8080
