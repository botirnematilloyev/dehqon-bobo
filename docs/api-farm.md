# Farm API (`/api/v1`)

Barcha so'rovlar `Authorization: Bearer <access_token>` talab qiladi.
Begona xo'jalik yoki issiqxona so'ralganda **404** qaytadi (403 emas).

## Enum qiymatlari

| Maydon | Qiymatlar |
|---|---|
| `system_type` | `greenhouse_soil`, `substrate`, `hydroponics`, `open_field` |
| `mixing_method` | `direct` (bakda), `ab` (A/B + injektor) |
| `water_source`, `source` | `well` (quduq), `canal` (ariq/kanal), `tap` (vodoprovod), `filter` (filtr/osmos) |

## Birinchi sozlash

`POST /onboarding`

```json
{ "region_id": 3, "district_id": 41, "water_source": "well", "system_type": "substrate" }
```

- `district_id` va `system_type` ixtiyoriy (`system_type` berilmasa — `greenhouse_soil`).
- Xo'jalik bo'lmasa yaratiladi, bo'lsa yangilanadi. Qayta yuborish xavfsiz.
- Manba bo'yicha **taxminiy** suv tahlili faol qilinadi (`is_estimate: true`).
- `users.onboarded_at` va viloyat yoziladi. Hammasi bitta tranzaksiyada.

Javob — `FarmOverview`:

```json
{
  "farm":  { "id": 1, "name": null, "region_id": 3, "district_id": 41,
             "system_type": "substrate", "has_drain": true,
             "mixing_method": "direct", "injector_ratio": null, "stock_tank_l": null,
             "created_at": 1763270000, "updated_at": 1763270000 },
  "water": { "id": 1, "source": "well", "is_estimate": true, "is_active": true,
             "ec": 1, "ph": null, "ca": 80, "mg": 30, "hco3_mmol": 4, "...": "..." },
  "plots": []
}
```

`has_drain` — o'lchash ekranida drenaj EC qatorini ko'rsatish kerakmi.

## Xo'jaliklar

| Metod | Yo'l | Izoh |
|---|---|---|
| GET | `/farms` | `{ "farms": [Farm] }` |
| POST | `/farms` | `Farm` maydonlari, hammasi ixtiyoriy → 201 |
| GET | `/farms/{farmID}` | `FarmOverview` (faol suv tahlili yo'q bo'lsa `water: null`) |
| PATCH | `/farms/{farmID}` | faqat berilgan maydonlar. `mixing_method: "ab"` uchun `injector_ratio` va `stock_tank_l` shart; `direct` ga qaytilsa ular tozalanadi |
| DELETE | `/farms/{farmID}` | 204. **CASCADE**: issiqxonalar, suv tahlillari, eritmalar ham o'chadi — ilovada tasdiq so'rang |

## Issiqxonalar

| Metod | Yo'l | Izoh |
|---|---|---|
| GET | `/farms/{farmID}/plots?archived=true` | `{ "plots": [Plot] }`, arxivdagilar faqat `archived=true` bilan |
| POST | `/farms/{farmID}/plots` | `{ "name": "1-issiqxona", "area_m2": 120, "default_volume_l": 200 }` → 201 |
| PATCH | `/farms/{farmID}/plots/{plotID}` | `name`, `area_m2`, `default_volume_l`, `is_archived` |
| DELETE | `/farms/{farmID}/plots/{plotID}` | 204, o'chirmaydi — **arxivlaydi** (jurnal yozuvlari saqlanadi) |

## Suv tahlili

| Metod | Yo'l | Izoh |
|---|---|---|
| GET | `/farms/{farmID}/water` | faol tahlil; yo'q bo'lsa 404 |
| GET | `/farms/{farmID}/water/history?limit=20` | `{ "analyses": [...] }`, yangisi birinchi, max 100 |
| POST | `/farms/{farmID}/water` | yangi tahlil faol bo'ladi, eskisi tarixda qoladi → 201 |

POST tanasi:

```json
// "Bilmayman" — taxminiy qiymat
{ "source": "canal", "estimate": true }

// Laboratoriya natijasi: ec, ca, mg, hco3_mmol majburiy
{ "source": "well", "ec": 1.35, "ca": 101.5, "mg": 33, "hco3_mmol": 4.8,
  "ph": 7.6, "na": 40, "lab_name": "Agrolab", "sampled_at": 1763200000 }
```

Birliklar: `ec` — mS/sm, ionlar — mg/L, `hco3_mmol` — mmol/L.

## Xo'jalik sozlamalari

`GET/PUT/DELETE /farms/{farmID}/fertilizers/{code}` va
`GET/PUT/DELETE /farms/{farmID}/parameters/{key}` — agronom standarti ustidan
dehqon kiritadigan qiymatlar. Batafsil: `docs/calc-parameters.md`.

"Bilmayman" (`estimate: true`) uchun suv tarkibi `water_source_defaults`
jadvalidan olinadi: avval viloyatniki, topilmasa mamlakat bo'yicha. Bunday
qiymat umuman bo'lmasa, 422 qaytadi.

## Xatolar

`{ "error": "..." }` — 400 (noto'g'ri qiymat yoki enum), 401, 404, 500.
