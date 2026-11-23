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
