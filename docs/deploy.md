# Ishga tushirish (Docker)

## Lokal dasturlash

```bash
cp .env.example .env    # qiymatlarni to'ldiring
make up                 # faqat Postgres + Redis
make migrate-docker     # yoki: make migrate-up (lokal migrate CLI bilan)
make run                # API: go run ./cmd/api
```

## Hammasi Docker'da (server / staging)

```bash
make up-app             # Postgres + Redis + migratsiyalar + API
make logs               # API loglari
make down
```

`docker compose --profile app up -d --build` tartibi:

1. `postgres`, `redis` — healthcheck'dan o'tguncha kutiladi.
2. `migrate` — `migrate/migrate:v4.18.1`, `./migrations` ni qo'llaydi va chiqadi (0). Qayta
   ishga tushirish xavfsiz: `no change`.
3. `api` — `build/api/Dockerfile` (statik binary, alpine, root emas, ~34 MB). Faqat migratsiya
   muvaffaqiyatli tugasa ishga tushadi. Healthcheck: `/readyz` (Postgres + Redis).

4. `web` — Mini App (statik) + Caddy reverse proxy (`build/web/`). `/api/*` → `api`, qolgani — SPA.
   `SITE_ADDRESS=app.example.uz` bo'lsa Caddy Let's Encrypt sertifikatini o'zi oladi:
   `WEB_BIND=0.0.0.0`, `WEB_PORT=80`, `WEB_TLS_PORT=443` qiling va `TRUST_PROXY=true`.
   Keyin `WEB_APP_URL=https://app.example.uz` — bot menyu tugmasi shu manzilni ochadi.

Portlar faqat `127.0.0.1` ga chiqariladi: tashqariga nginx / reverse proxy orqali
(`TRUST_PROXY=true`). `.env` dagi `DB_PORT`, `REDIS_PORT`, `HTTP_PORT` — host portlari;
konteyner ichidagi manzillarni compose o'zi qo'yadi.

Boshqa muhit fayli: `ENV_FILE=.env.staging docker compose --env-file .env.staging --profile app up -d --build`.

## CI (`.github/workflows/ci.yml`)

Har push (main) va PR'da: `gofmt`, `go vet` (oddiy va `integration` tegi), unit testlar `-race`,
integration testlar (testcontainers), API image build va compose konfiguratsiyasi tekshiruvi.
Image registry'ga hali yuklanmaydi.

## Serverda nginx allaqachon bo'lsa (masalan binary.uz VPS)

Caddy (`web`) o'rniga mavjud nginx: u Mini App statik fayllarini beradi va `/api` ni
API'ga uzatadi, HTTPS'ni `certbot` qo'yadi.

```bash
git clone https://github.com/botirnematilloyev/dehqon-bobo.git /opt/dehqon-bobo && cd /opt/dehqon-bobo
# .env: TRUST_PROXY=true, WEB_APP_URL=https://<domen>, bo'sh portlar (DB_PORT, REDIS_PORT, HTTP_PORT)
docker compose --profile app up -d --build api   # postgres, redis, migrate, api (web emas)
make webapp-build                                 # webapp/dist
cp build/nginx/dehqonbobo.binary.uz.conf /etc/nginx/sites-available/
ln -s /etc/nginx/sites-available/dehqonbobo.binary.uz.conf /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx
certbot --nginx -d dehqonbobo.binary.uz
```

Yangilash: `git pull && docker compose --profile app up -d --build api && make webapp-build`.
