# Retsept preview

`POST /api/v1/farms/{farmID}/recipes/preview` 🔒 — ekin + bosqich uchun qaysi tuzdan
necha gramm. **Hech narsa saqlamaydi** (saqlash — keyingi qadam: `solution_batches`).

## So'rov

```json
{
  "crop": "tomato",
  "stage": "fruiting",
  "method": "direct",
  "volume_l": 200,
  "acids": ["H3PO4_85", "HNO3_38"]
}
```

| Maydon | Izoh |
|---|---|
| `crop` | `GET /catalog/crops` dagi kod |
| `stage` | `seedling` \| `vegetative` \| `fruiting` |
| `method` | `direct` (bakda) \| `ab` (A/B konsentrat). Yo'q bo'lsa — xo'jalik sozlamasi |
| `volume_l` | `direct` uchun majburiy: ish eritmasi hajmi |
| `stock_tank_l`, `injector_ratio` | `ab` uchun; yo'q bo'lsa xo'jalik sozlamasi. Ish eritmasi = idish × nisbat |
| `acids` | ishlatish tartibida. Tavsiya: avval `H3PO4` (P baribir kerak), keyin `HNO3`. Bo'sh bo'lsa bikarbonat neytrallanmaydi |

## Hisob tartibi

1. **Profil** — katalogdan: tasdiqlangani, bo'lmasa eng yangi draft (`profile_not_reviewed` ogohlantirishi bilan).
2. **Suv** — xo'jalikning faol tahlili (taxminiy bo'lsa `water_is_estimate`).
3. **O'g'itlar va parametrlar** — agronom standarti + xo'jalik o'zgartirishlari
   (`/farms/{id}/settings/...`). Kerakli tuz yoki parametr yo'q bo'lsa — xato, jimgina standart qo'yilmaydi.
4. **Suv + kislota qatlami** ([calc-water.md](calc-water.md)): suvdagi elementlar ayiriladi,
   HCO3 `residual_hco3_mmol` gacha neytrallanadi.
5. **HPG** — 10 L ga gramm → hajmga ko'paytiriladi, tarozi aniqligi `weight_decimals` bo'yicha.
6. **Mikro** — har element uchun katalogdagi birinchi mikro o'g'it: `mg/L × 1000 / % × V / 10000` g (HPG formulasi), 0.001 g gacha.

Tanklar: **A** — Ca va nitratlar (CaNO3, KNO3, NH4NO3, MgNO3, CaCl2); **B** — sulfat, fosfat, mikro.
Kislotalar alohida, `tank` katalogdagi `default_tank` (odatda `C`).

## Javob (qisqartirilgan)

Haqiqiy natija (e2e test): pomidor, `fruiting`, 200 L, kanal suvi (taxminiy: Ca 70, Mg 25,
HCO3 3.5 mmol/L), `H3PO4_85` + `HNO3_38`. `name` maydonlari qisqartirilgan.

```json
{
  "profile": {"id": 2, "crop": "tomato", "stage": "fruiting", "title": "Standart eritma (mineral vata)",
              "source": "Sonneveld & Voogt 2009; Sonneveld & Straver 1994", "is_reviewed": false},
  "method": "direct", "volume_l": 200, "stock_tank_l": null, "injector_ratio": null,
  "tanks": [
    {"tank": "A", "items": [{"code": "CaNO3", "grams": 172.5}, {"code": "KNO3", "grams": 69.6},
                            {"code": "NH4NO3", "grams": 19.2}], "total_grams": 261.3},
    {"tank": "B", "items": [{"code": "MgSO4", "grams": 67.6}, {"code": "K2SO4", "grams": 105.5},
                            {"code": "FeChelate", "grams": 1.289}, {"code": "MnSO4", "grams": 0.338},
                            {"code": "H3BO3", "grams": 0.371}, {"code": "ZnSO4", "grams": 0.288},
                            {"code": "CuSO4", "grams": 0.037}, {"code": "NH4Mo", "grams": 0.018}], "total_grams": 175.441}
  ],
  "acids": [
    {"code": "H3PO4_85", "tank": "C", "ml": 20.5, "element": "P",   "nutrient_mg_l": 46.461, "neutralized_hco3_mmol": 1.5},
    {"code": "HNO3_38",  "tank": "C", "ml": 40.4, "element": "NO3", "nutrient_mg_l": 21.01,  "neutralized_hco3_mmol": 1.5}
  ],
  "target_mg_l":    {"no3_n": 224.107, "nh4_n": 16.808, "p": 46.461, "k": 371.434, "ca": 216.421, "mg": 58.332, "s": 141.086, "cl": 0},
  "achieved_mg_l":  {"no3_n": 188.388, "nh4_n": 16.808, "p": 46.461, "k": 371.434, "ca": 216.421, "mg": 58.332, "s": 141.086, "cl": 0},
  "deviation_mg_l": {"no3_n": -35.719, "nh4_n": 0, "p": 0, "k": 0, "ca": 0, "mg": 0, "s": 0, "cl": 0},
  "ec": {"target": 2.6, "estimated": 2.688},
  "ph": {"min": 5.5, "max": 6.5},
  "hco3_left_mmol": 0.5,
  "water": {"analysis_id": 1, "ec": 0.9},
  "warnings": [{"code": "poor_ion_balance", "value": 22.655}, {"code": "profile_not_reviewed"}, {"code": "water_is_estimate"}],
  "calc_version": "hpg@0a8eb86+water/1"
}
```

Bu misolda NO3 −35.7 mg/L: taxminiy kanal suvida faqat Ca, Mg va HCO3 bor (SO4, Cl yo'q),
shuning uchun tahlil 23% muvozanatsiz (`poor_ion_balance`). Suvdagi Ca kalsiy selitraning
o'rnini egallaydi va u bilan keladigan NO3 kamayadi. Haqiqiy laboratoriya tahlili bilan bu
farq aniq hisoblanadi.

`deviation_mg_l` = erishilgan − maqsad. Kationlar, P va S aniq maqsadda bo'ladi; **NO3 — "erkin"
anion** (HPG modeli): elektroneytrallik bo'yicha suvdagi Na, Cl va qoldiq HCO3 ga qarab siljiydi.

### Ogohlantirishlar

| Kod | Ma'nosi |
|---|---|
| `profile_not_reviewed` | profil agronom tomonidan hali tasdiqlanmagan |
| `water_is_estimate` | suv tahlili o'lchanmagan, manba bo'yicha taxminiy |
| `water_salinity_above_crop_threshold` | suv EC ekinning FAO-29 chegarasidan yuqori (`value` = EC) |
| `water_exceeds_target` | suvda element maqsaddan ko'p (`element`, `value` mg/L ortiqcha) |
| `poor_ion_balance` | suv tahlilida kation/anion farqi > `water_ion_balance_tolerance_pct` |
| `no_acid_selected` | kislota tanlanmagan, HCO3 qoldiqdan yuqori |
| `bicarbonate_left` | kislota yetmadi (`value` = qolgan HCO3 mmol/L) |
| `acid_over_target` | oxirgi kislota ozuqa maqsadidan oshdi (pH ustuvor) |
| `infeasible_salts` | manfiy gramm: tanlangan tuzlar bilan profilni yig'ib bo'lmaydi |
| `micro_fertilizer_missing` | katalogda shu mikroelement uchun o'g'it yo'q |
| `solubility_exceeded` | `ab`: tuz idishdagi 20 °C eruvchanlikdan oshdi (`element` = tuz, `value` = g/L) |

### Xatolar

| Status | `error` |
|---|---|
| 400 | `recipe.invalid_stage`, `recipe.invalid_method` (hajm yo'q / A/B sozlanmagan), `recipe.invalid_acid` |
| 404 | `recipe.farm_not_found`, `recipe.crop_not_found`, `recipe.profile_not_found` |
| 409 | `recipe.water_analysis_required` |
| 422 | `recipe.catalog_incomplete` — katalogda tuz/parametr yo'q; agronom tuzatadi |

# Jurnal (saqlangan retseptlar)

| | Yo'l | Izoh |
|---|---|---|
| POST | `/farms/{farmID}/recipes` | preview bilan bir xil so'rov + ixtiyoriy `plot_id` → 201 |
| GET | `/farms/{farmID}/recipes?limit=20&before_id=&plot_id=` | yangisi birinchi, `limit` ≤ 100 |
| GET | `/farms/{farmID}/recipes/{id}` | to'liq: tarkib + saqlangan hisob |
| POST | `/farms/{farmID}/recipes/{id}/mixed` | "tayyorladim" — idempotent (qayta bosish vaqtni o'zgartirmaydi) |
| DELETE | `/farms/{farmID}/recipes/{id}` | faqat tayyorlanmagan → 204; tayyorlangani 409 |

- **Hisob serverda qayta bajariladi** — mijoz yuborgan grammlarga ishonilmaydi.
- **`recipe` = o'sha paytdagi preview javobi** (`calc_snapshot`, o'zgarmas). Katalog,
  xo'jalik sozlamalari yoki formula keyin o'zgarsa ham jurnal yozuvi o'zgarmaydi;
  `calc_version` qaysi formula bilan hisoblanganini aytadi.
- `items` — `batch_items`: `tank` = `single` (to'g'ridan usul, hammasi bitta bakda) yoki
  `A` / `B` / `C`; tuzlar `grams`, kislotalar `ml`.
- `target_ec` — kutilgan EC (suv + kislota + tuzlar bahosi). Keyingi qadamda o'lchov shu bilan solishtiriladi.
- Manfiy gramm (tuzlar bilan yig'ib bo'lmaydigan profil) preview'da ko'rinadi, lekin
  saqlanmaydi: `422 recipe.infeasible`.
- Ro'yxat sahifalari: javobdagi `next_before_id` ni keyingi so'rovda `before_id` qiling; `null` — oxiri.

```json
{
  "id": 1, "crop": "tomato", "stage": "fruiting", "plot_id": 1, "method": "direct",
  "volume_l": 200, "stock_tank_l": null, "injector_ratio": null, "target_ec": 2.69,
  "status": "planned", "created_at": 1763400000, "mixed_at": null, "calc_version": "hpg@0a8eb86+water/1",
  "items": [{"code": "CaNO3", "tank": "single", "grams": 172.5}, {"code": "HNO3_38", "tank": "single", "ml": 40.4}],
  "recipe": { "...": "preview javobi" }
}
```

| Status | `error` |
|---|---|
| 404 | `recipe.not_found`, `recipe.plot_not_found` (issiqxona shu xo'jalikda emas yoki arxivda) |
| 409 | `recipe.already_mixed` |
| 422 | `recipe.infeasible` |

# O'lchovlar va korrektor

| | Yo'l | Izoh |
|---|---|---|
| POST | `/farms/{farmID}/measurements` | o'lchov + baholash → 201 |
| GET | `/farms/{farmID}/measurements?plot_id=&batch_id=&limit=&before_id=` | yangisi birinchi |
| POST | `/farms/{farmID}/recipes/{id}/correction` | aylanma bak uchun tuzatuvchi eritma (HPG korrektori), saqlamaydi |

## O'lchov

```json
{"batch_id": 1, "ec_in": 3.2, "ph_in": 7.2, "ec_drain": 4.7, "drain_pct": 30, "tank_volume_l": 150, "note": "ertalab"}
```

`ec_in`, `ph_in`, `ec_drain` dan kamida bittasi kerak. `measured_at` yo'q bo'lsa — hozir.
`batch_id` berilsa issiqxona retseptdan olinadi va o'lchov retsept bilan solishtiriladi:

| Tekshiruv | Asos | Parametr (calc_parameters, 000013) |
|---|---|---|
| EC kirish ↔ retseptning `target_ec` | `|farq| / kutilgan > chegara` | `ec_tolerance_pct` (10%) |
| pH kirish ↔ profil `ph_min…ph_max` | oraliq | — (profil) |
| EC drenaj − EC kirish | tuz to'planishi | `drain_ec_excess_max` (1.0 mS/sm) |
| drenaj ulushi | oraliq | `drain_pct_min` / `drain_pct_max` (20 / 50%) |

Parametrlar agronom standarti (tasdiq kutilmoqda), xo'jalik o'zgartira oladi
(`/farms/{id}/settings/calc-parameters`). Yo'q bo'lsa — `422 recipe.catalog_incomplete`.

**EC tuzatish** — aralashtirish balansi: `(EC − EC_suv)` erigan tuzlarga proporsional
(kichik o'zgarishlar uchun chiziqli; HPG korrektori ham shu farazda):

| Holat | `correction.action` | Hisob |
|---|---|---|
| bak, EC yuqori | `add_water` | `water_l = V·(EC − EC_kut)/(EC_kut − EC_suv)` |
| bak, EC past | `add_salts` | `fraction = (EC_kut − EC)/(EC − EC_suv)`; `items` — retsept tuzlari × ulush × V/V_retsept. **Kislota qo'shilmaydi** (bikarbonat allaqachon neytrallangan) |
| A/B | `adjust_injector` | `injector_ratio = N·(EC − EC_suv)/(EC_kut − EC_suv)` |

`V` — `tank_volume_l` (yo'q = retsept hajmi). pH uchun ml hisoblanmaydi: buning uchun
eritmaning titrlash egri chizig'i kerak — faqat yo'nalish (`ph_high_add_acid`, `ph_low_check_acid`).

Javob shakli (raqamlar misol uchun):

```json
{
  "measurement": {"id": 3, "batch_id": 1, "plot_id": 1, "ec_in": 3.2, "ph_in": 7.2, "measured_at": 1763400000, "...": "..."},
  "assessment": {
    "ec": {"value": 3.2, "expected": 2.69, "deviation_pct": 19, "tolerance_pct": 10, "status": "high"},
    "ph": {"value": 7.2, "min": 5.5, "max": 6.5, "status": "high"},
    "drain": {"ec_excess": 1.5, "ec_excess_max": 1, "ec_status": "high", "pct": 30, "pct_min": 20, "pct_max": 50, "pct_status": "ok"},
    "correction": {"action": "add_water", "water_l": 40.9},
    "advice": ["ec_high_add_water", "ph_high_add_acid", "drain_ec_high"]
  }
}
```

Maslahat kodlari: `ec_low_add_salts`, `ec_high_add_water`, `ec_adjust_injector`,
`ph_high_add_acid`, `ph_low_check_acid`, `drain_ec_high`, `drain_pct_low`, `drain_pct_high`.

## Korrektor (aylanma bak)

```json
{"ec_now": 2.96, "volume_now_l": 100, "volume_final_l": 200, "acids": ["HNO3_38"]}
```

HPG modeli (`hpg.Correct` bilan bir xil):

```
joriy   = retseptda erishilgan tarkib × (EC_hozir / EC_kutilgan)
V_tuz   = V_yakuniy − V_hozir
C_tuz   = (C_yakuniy·V_yakuniy − C_joriy·V_hozir) / V_tuz      (C_yakuniy = erishilgan tarkib)
```

Tuzatuvchi eritma o'sha suv va kislotalar bilan tayyorlanadi, shuning uchun uning
retsepti oddiy hisob bilan chiqadi (`recipe` — preview javobi shaklida, `V_tuz` litr uchun).
`acids` yo'q bo'lsa — retseptdagilar. Qaysidir ion `C_tuz` da manfiy bo'lsa (bak juda quyuq)
— `dilute_only: true`, `water_to_add_l = V·(EC_hozir − EC_suv)/(EC_kut − EC_suv) − V`, tuz yo'q.

| Status | `error` |
|---|---|
| 400 | `recipe.invalid_measurement`, `recipe.invalid_correction` |
| 404 | `recipe.not_found` (batch), `recipe.plot_not_found` |
| 422 | `recipe.catalog_incomplete` (o'lchov parametri yo'q) |
