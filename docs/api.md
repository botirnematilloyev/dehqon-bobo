# API umumiy ko'rinishi

To'liq spec: [openapi.yaml](openapi.yaml) (OpenAPI 3.1). `cmd/api/openapi_test.go` spec va
router mosligini tekshiradi — yangi endpoint spec'siz qo'shilsa test yiqiladi.

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
| POST | `/auth/telegram/webapp` | **Mini App ichidan kirish (asosiy):** `{"init_data": Telegram.WebApp.initData}` — kod kerak emas |
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

### Retsept 🔒
| | Yo'l | Izoh |
|---|---|---|
| POST | `/farms/{farmID}/recipes/preview` | ekin + bosqich → A/B tanklar uchun gramm, kislota ml, ogohlantirishlar. Saqlamaydi |
| POST / GET | `/farms/{farmID}/recipes` | retseptni saqlash (jurnal) / ro'yxat |
| GET / DELETE | `/farms/{farmID}/recipes/{id}` | ochish / tayyorlanmaganini o'chirish |
| POST | `/farms/{farmID}/recipes/{id}/mixed` | "tayyorladim" |
| POST | `/farms/{farmID}/recipes/{id}/correction` | aylanma bak: tuzatuvchi eritma (HPG korrektori) |
| POST / GET | `/farms/{farmID}/measurements` | EC/pH/drenaj o'lchovi + baholash va tuzatish / ro'yxat |

### Mavsumlar 🔒
| | Yo'l | Izoh |
|---|---|---|
| POST | `/farms/{farmID}/plots/{plotID}/seasons` | mavsum boshlash |
| GET | `/farms/{farmID}/seasons` | ro'yxat (`open`, `plot_id`) |
| GET / PATCH | `/farms/{farmID}/seasons/{id}` | natija kartochkasi / bosqich |
| POST | `/farms/{farmID}/seasons/{id}/end` | yakun: hosil, xarajat → kartochka |

Batafsil: [api-season.md](api-season.md).

Batafsil: [api-recipe.md](api-recipe.md).

### Katalog (agronom standartlari) 🔒
| | Yo'l | Kim |
|---|---|---|
| GET | `/catalog/fertilizers`, `/catalog/water-sources`, `/catalog/calc-parameters` | hamma |
| PATCH | `/admin/catalog/fertilizers/{code}` | agronom, admin |
| PUT | `/admin/catalog/water-sources/{source}?region_id=` | agronom, admin |
| PATCH | `/admin/catalog/calc-parameters/{key}` | agronom, admin |
| GET | `/catalog/crops` | hamma |
| GET | `/catalog/crops/{code}/profiles` | hamma |
| POST | `/admin/catalog/crops/{code}/profiles` | agronom, admin — yangi profil, doim `draft` |
| PUT | `/admin/catalog/crop-profiles/{id}` | agronom, admin — faqat `draft` |
| POST | `/admin/catalog/crop-profiles/{id}/approve` | agronom, admin |

#### Ekin profillari

Profil — ekin + bosqich (`seedling` ko'chat, `vegetative` o'sish, `fruiting` hosil) uchun
maqsad eritma, **mg/L** (`targets_mg_l`), `ec_target`, `ph_min`/`ph_max`.

- **S kiritilmaydi** — kation/anion ekvivalentligidan (HPG balansi, P = H2PO4⁻) hisoblanadi.
  Anionlar kationlardan ko'p bo'lsa → `422 catalog.ion_imbalance`.
- **Hayot sikli:** `draft` → `approve` → `approved`. Shu ekin+bosqichning avvalgi
  tasdiqlangani `archived` bo'ladi (jurnal yozuvlari unga bog'liq qoladi, ro'yxatda chiqmaydi).
  Tasdiqlangan profil o'zgartirilmaydi (`409 catalog.crop_profile_locked`) — yangi draft yaratiladi.
- `is_reviewed: true` = agronom tasdiqlagan ("Agronom ko'rdi ✓").

**Seed (000012):** 5 ekin — FAO-29 suv sho'rlik chegarasi bilan. Pomidor va bodring
uchun Sonneveld & Voogt (2009) mineral vata standart eritmasi `vegetative` va `fruiting`
bosqichlarida, `draft` holatda — agronom tasdiqlashi kerak. Qalampir, qulupnay, salat
profillari va `seedling` bosqichi agronom tomonidan kiritiladi.

### Admin 🔒
`GET /admin/users`, `GET /admin/users/{id}` — faqat admin.

Batafsil: [api-farm.md](api-farm.md), [calc-parameters.md](calc-parameters.md).
