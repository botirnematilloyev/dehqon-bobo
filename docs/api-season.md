# Mavsumlar

Mavsum — bitta issiqxonada bitta ekin davri: boshlanish → bosqichlar → yakun (hosil, xarajat).
Issiqxonada bir vaqtda bitta ochiq mavsum (`uq_seasons_open`).

| | Yo'l | Izoh |
|---|---|---|
| POST | `/farms/{farmID}/plots/{plotID}/seasons` | `{"crop":"tomato","stage":"seedling","started_at":…}` → 201. `stage` yo'q = `seedling`, `started_at` yo'q = hozir |
| GET | `/farms/{farmID}/seasons?plot_id=&open=true\|false&limit=&before_id=` | yangisi birinchi |
| GET | `/farms/{farmID}/seasons/{id}` | natija kartochkasi (ochiq mavsumda — hozirgacha) |
| PATCH | `/farms/{farmID}/seasons/{id}` | `{"stage":"fruiting"}` — faqat ochiq mavsum |
| POST | `/farms/{farmID}/seasons/{id}/end` | `{"yield_kg":2400,"area_m2":120,"fertilizer_cost_uzs":…,"ended_at":…}` → kartochka |

## Retseptlar bilan bog'lanish

`POST /farms/{id}/recipes` da `plot_id` berilsa va issiqxonada **shu ekinning** ochiq mavsumi
bo'lsa, retsept avtomatik unga bog'lanadi (`solution_batches.season_id`). Boshqa ekin — bog'lanmaydi.

## Natija kartochkasi

```json
{
  "season": {"id": 1, "plot_id": 1, "crop": "tomato", "stage": "fruiting", "is_open": false,
             "started_at": 1753000000, "ended_at": 1763400000, "yield_kg": 2400, "area_m2": 120, "fertilizer_cost_uzs": null},
  "days": 120,
  "area_m2": 120,
  "yield_kg_per_m2": 20,
  "batches_mixed": 30,
  "solution_l": 6000,
  "fertilizers": [{"code": "CaNO3", "name": {...}, "grams": 50000, "ml": 0, "cost_uzs": 600000},
                  {"code": "MgSO4", "grams": 10000, "ml": 0, "cost_uzs": null}],
  "cost_uzs": 972300,
  "cost_estimated": true,
  "cost_partial": true,
  "cost_per_kg_uzs": null,
  "measurements": {"count": 12, "avg_ec_in": 2.73, "avg_ph_in": 5.9, "avg_drain_pct": 28}
}
```

(raqamlar misol uchun)

- **Maydon** — yakunda kiritilgani, bo'lmasa issiqxona maydoni (saqlanadi).
- **O'g'it sarfi** — mavsumga bog'langan va **tayyorlangan** (`mixed`) retseptlar tarkibi yig'indisi.
- **Xarajat**: dehqon `fertilizer_cost_uzs` kiritsa — o'sha (`cost_estimated: false`). Aks holda
  sarf × xo'jalik narxi (`/farms/{id}/settings/fertilizers/{code}` → `price_uzs_per_kg`);
  kislota ml → kg zichlik orqali. Hozirgi narx bilan — taxmin. Ba'zi o'g'itda narx yo'q bo'lsa
  `cost_partial: true` va `cost_per_kg_uzs` hisoblanmaydi (noto'g'ri past chiqmasin).
- **O'lchovlar** — issiqxonadagi o'lchovlar mavsum oralig'ida (o'rtachalar).

| Status | `error` |
|---|---|
| 400 | `season.invalid` (hosil yo'q/manfiy, sana boshlanishidan oldin, kelajak), `recipe.invalid_stage` |
| 404 | `season.not_found`, `recipe.plot_not_found`, `recipe.crop_not_found` |
| 409 | `season.open_exists`, `season.ended` |

Keyingi: `season_issues` (muammolar/belgilar) — `symptoms` ma'lumotnomasi to'ldirilgach.
