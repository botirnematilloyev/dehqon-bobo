# Telegram bot va Mini App

Ilova Telegram ichida Mini App sifatida ochiladi. Kirish — `Telegram.WebApp.initData`
orqali (`POST /api/v1/auth/telegram/webapp`), kod/OTP kerak emas.

## Sozlash

1. **`.env`**
   - `TELEGRAM_BOT_TOKEN` — @BotFather bergan token (repoga qo'shilmaydi, chatga yozilmaydi).
   - `WEB_APP_URL` — Mini App manzili, **faqat `https://`**.
   - `TELEGRAM_WEBAPP_MAX_AGE` — initData amal qilish muddati (standart `24h`).
2. **API'ni ishga tushiring** (`make run` yoki `make up-app`). Bot ishga tushganda o'zi:
   - "/" buyruqlarini qo'yadi: `/start`, `/login` — o'zbek (standart), rus, ingliz tillarida;
   - chat menyu tugmasini Mini App'ga o'rnatadi (`setChatMenuButton`, barcha chatlar uchun).
     Shuning uchun BotFather'da menyu tugmasini qo'lda sozlash shart emas.
3. **Ixtiyoriy — to'g'ridan havola** (`t.me/<bot>/<app>`): @BotFather → `/newapp` → botni
   tanlang → nom, tavsif, rasm → URL = `WEB_APP_URL` → qisqa nom (masalan `app`).

## Bot xatti-harakati

| Buyruq | Mini App URL https bo'lsa | aks holda |
|---|---|---|
| `/start` | "Ilovani ochish" tugmasi (`web_app`) | `/login` haqida matn |
| `/login` | brauzer uchun 6 xonali kod (eski usul) | xuddi shunday |

Matnlar foydalanuvchining Telegram tilida (`language_code`): `ru`, `en`, qolganlari — o'zbekcha.

## Lokal sinov

Telegram Mini App'ni faqat HTTPS orqali ochadi:

```bash
cloudflared tunnel --url http://localhost:5173
```

Chiqqan `https://…trycloudflare.com` manzilini `WEB_APP_URL` ga qo'ying va API'ni qayta
ishga tushiring — menyu tugmasi yangi manzilga o'tadi.
