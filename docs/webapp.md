# Mini App (frontend)

`webapp/` — Vite + React + TypeScript. Telegram ichida ochiladi; kirish
`Telegram.WebApp.initData` bilan (`POST /api/v1/auth/telegram/webapp`), parol/kod yo'q.

## Tuzilishi

| Fayl | Vazifa |
|---|---|
| `src/telegram.ts` | Telegram SDK: `ready`, `expand`, tema, BackButton, haptic |
| `src/api.ts` | API klient: initData bilan kirish, 401 da bir marta qayta kirish, `Accept-Language` |
| `src/i18n.ts` | o'zbek (standart) va rus matnlari; hisob ogohlantirishlari va maslahat kodlari → matn |
| `src/icons.tsx` | Prototipdagi ikonkalar (ekinlar rangli, qolganlari `currentColor`) |
| `src/screens/` | Birinchi sozlash, Bosh sahifa, Retsept, Jurnal, O'lchov |

Tema: Telegram yorug' rejimida — prototip ranglari (yashil, Inter + Manrope); qorong'i
rejimda fon va matn Telegram mavzusidan (`--tg-theme-*`).

## Lokal ishga tushirish

```bash
make up && make migrate-docker   # Postgres + Redis + migratsiyalar
make run                         # API (HTTP_PORT, standart 8087)
cd webapp && npm install && npm run dev   # http://localhost:5173, /api → API
```

API boshqa portda bo'lsa: `API_PROXY=http://localhost:8090 npm run dev`.

Brauzerda to'g'ridan ochilsa ilova "Telegram orqali oching" deydi — initData yo'q.
Haqiqiy botda sinash: tunnel (`cloudflared tunnel --url http://localhost:5173`) →
`WEB_APP_URL=https://…` → API'ni qayta ishga tushiring → botda `/start`.

## Tekshiruvlar

```bash
cd webapp && npm run typecheck && npm test && npm run build
```

## Production

`build/web/Dockerfile` — statik build + Caddy (`build/web/Caddyfile`): `/api/*` → Go API,
qolgani — SPA. Ilova va API bitta domenda (CORS kerak emas). `make up-app` `web`
konteynerini ham ko'taradi; domen uchun `SITE_ADDRESS` (batafsil: [deploy.md](deploy.md)).
