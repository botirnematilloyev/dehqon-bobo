# API umumiy ko'rinishi

Hamma yo'llar `/api/v1` ostida. 🔒 = `Authorization: Bearer <access_token>` kerak.

## Til (Accept-Language)

Server `Accept-Language` sarlavhasini o'qiydi va xato/xabar matnini shu tilda qaytaradi.

| So'ralgan | Javob tili (`Content-Language`) |
|---|---|
| yo'q, `uz`, `uz-Latn`, qo'llanmaydigan til (`de`, …) | `uz` (lotin, standart) |
| `uz-Cyrl` | `uz-Cyrl` (lotindan avtomatik o'giriladi; EC, pH, NO3 kabi texnik so'zlar lotinda qoladi) |
| `ru`, `ru-RU` | `ru` |
| `en`, `en-US` | `en` |

`q=` vaznlari hisobga olinadi. Har javobda `Content-Language` va `Vary: Accept-Language` bor.

## Xato formati

```json
{ "error": "farm.water_analysis_not_found", "message": "Анализ воды ещё не добавлен" }
```

- `error` — **barqaror kod**, tilga bog'liq emas. Ilova mantiqni shunga quradi.
- `message` — foydalanuvchiga ko'rsatiladigan, tarjima qilingan matn.

Kodlar `<modul>.<sabab>` ko'rinishida: `common.*`, `auth.*`, `user.*`, `location.*`,
`farm.*`, `catalog.*`. Ro'yxat: `internal/platform/i18n/common.go` va har modulning
`transport/http/messages.go` fayli. Yangi kalit qo'shilsa, uz/ru/en matni bo'lmasa
`TestMessagesRegistered` yiqiladi.

## Endpointlar

### Kirish (`/auth`)
| | Yo'l | Izoh |
|---|---|---|
| POST | `/auth/telegram/verify` | Telegram botdan olingan kod bilan kirish |
| POST | `/auth/otp/request`, `/auth/otp/verify` | email OTP (zaxira usul) |
| POST | `/auth/refresh` | tokenni yangilash |
| POST 🔒 | `/auth/logout` | joriy qurilmadan chiqish |
| POST 🔒 | `/auth/logout-all` | barcha qurilmalardan chiqish |

### Men (`/me`)
| | Yo'l | Izoh |
|---|---|---|
| GET/PATCH 🔒 | `/me` | profil |
| POST 🔒 | `/me/onboarding` | birinchi sozlash: xo'jalik + suv manbai |
| GET 🔒 | `/me/sessions` | faol qurilmalar |
| DELETE 🔒 | `/me/sessions/{id}` | bitta qurilmani chiqarish |

### Joylashuv
`GET /regions`, `GET /districts?region_id=` — ochiq.

### Xo'jaliklar (`/farms`) 🔒
| | Yo'l | Izoh |
|---|---|---|
| GET/POST | `/farms` | ro'yxat / yangi |
| GET/PATCH/DELETE | `/farms/{farmID}` | |
| GET/POST | `/farms/{farmID}/plots` | maydonlar (issiqxona, qator) |
| PATCH | `/farms/{farmID}/plots/{plotID}` | |
| POST | `/farms/{farmID}/plots/{plotID}/archive` | o'chirmaydi, arxivlaydi |
| GET/POST | `/farms/{farmID}/water-analyses` | suv tahlillari tarixi / yangisini kiritish |
| GET | `/farms/{farmID}/water-analyses/active` | hozirgi faol tahlil |
| GET | `/farms/{farmID}/settings/fertilizers` | o'g'itlar: standart + xo'jalik qiymati |
| PUT/DELETE | `/farms/{farmID}/settings/fertilizers/{code}` | o'z qiymatini qo'yish / standartga qaytarish |
| GET | `/farms/{farmID}/settings/calc-parameters` | hisob parametrlari |
| PUT/DELETE | `/farms/{farmID}/settings/calc-parameters/{key}` | o'z qiymati / standartga qaytarish |

### Katalog (agronom standartlari) 🔒
| | Yo'l | Kim |
|---|---|---|
| GET | `/catalog/fertilizers`, `/catalog/water-sources`, `/catalog/calc-parameters` | hamma |
| PATCH | `/admin/catalog/fertilizers/{code}` | agronom, admin |
| PUT | `/admin/catalog/water-sources/{source}?region_id=` | agronom, admin |
| PATCH | `/admin/catalog/calc-parameters/{key}` | agronom, admin |

### Admin 🔒
`GET /admin/users`, `GET /admin/users/{id}` — faqat admin.

Batafsil: [api-farm.md](api-farm.md), [calc-parameters.md](calc-parameters.md).
