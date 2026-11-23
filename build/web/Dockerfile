# Mini App: statik build (Vite) + Caddy. Caddy /api ni Go API'ga uzatadi —
# ilova va API bitta domenda, shuning uchun CORS kerak emas.
FROM node:22-alpine AS build
WORKDIR /src
COPY webapp/package.json webapp/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY webapp/ ./
RUN npm run build

FROM caddy:2.10-alpine
COPY build/web/Caddyfile /etc/caddy/Caddyfile
COPY --from=build /src/dist /srv
