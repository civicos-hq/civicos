# syntax=docker/dockerfile:1.7
#
# Docusaurus documentation site → static build served by nginx on Cloud Run.
# Build context is the REPO ROOT (pnpm workspace — see apps/web/Dockerfile).

FROM node:20-alpine AS build
WORKDIR /src

ENV HUSKY=0
ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0

COPY . .
RUN corepack enable && corepack pnpm install --frozen-lockfile
RUN corepack pnpm --filter @civicos/docs... build

FROM nginx:1.27-alpine AS runtime
# Docusaurus emits to build/, not dist/.
COPY --from=build /src/apps/docs/build /usr/share/nginx/html
COPY deploy/gcp/nginx-static.conf /etc/nginx/conf.d/default.conf
EXPOSE 8080
