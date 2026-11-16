# Sozlanadigan qiymatlar: agronom → xo'jalik → hisob

Hisobda ishlatiladigan har bir sozlanadigan qiymat **bazada** turadi.
Kodda faqat o'zgarmas narsalar qoladi:
- molyar massalar
- kislota proton soni (pH 5–6)
- HPG'ning empirik formulalari: EC = 0.095 × (meq + 2), zichlik polinomlari.
  HPG'ning o'zida ham ular o'zgartirilmaydi.
- validatsiya chegaralari (masalan EC ≤ 20)

## Qatlamlar

| Qatlam | Kim | Qayerda | Qachon ishlaydi |
|---|---|---|---|
| 1. Standart | agronom (`role ≥ 2`) | `fertilizers`, `fertilizer_elements`, `calc_parameters`, `water_source_defaults` | har doim |
| 2. Xo'jalik | dehqon (xo'jalik egasi) | `farm_fertilizers`, `farm_calc_parameters` | to'ldirilgan bo'lsa 1-qatlam o'rniga |
| 3. Bitta hisob | so'rov | saqlanmaydi | eng ustun |

Har bir javobda effektiv qiymat bilan birga standart qiymat (`default`) va
`overridden` belgisi qaytadi, shuning uchun ilovada **"Standartga qaytarish"**
tugmasi qilish mumkin.

## Nima qayerda

| Qiymat | Standart (agronom) | Xo'jalik (dehqon) |
|---|---|---|
| Makro tuzlar tarkibi, % | `fertilizer_elements` | `farm_fertilizers.label_percents` (yorliqdan) |
| Mikro tuzlar foizi | `fertilizer_elements` | `farm_fertilizers.label_percents` |
| Monoeritma g/L, zichlik | `fertilizers.stock_g_l` | `stock_g_l`, `stock_density_g_ml` |
| Kislota kuchi %, zichlik | `fertilizers.concentration_pct`, `density_g_ml` | o'z idishi yorlig'idan |
| Narx, brend | — | `price_uzs_per_kg`, `brand` |
| Qoldiq HCO₃ (0.5 mmol/L) | `calc_parameters.residual_hco3_mmol` | ha |
| Suv tahlili balansi chegarasi (10%) | `water_ion_balance_tolerance_pct` | **yo'q** — faqat agronom |
| K₂SO₄ / Mg(NO₃)₂ rejimi | `use_k2so4`, `use_mgno3` | ha |
| Tarozi aniqligi | `weight_decimals` (−1 = HPG qoidasi) | ha |
| Suv manbai taxminiy tarkibi | `water_source_defaults` (mamlakat + viloyat) | — (xo'jalikning o'z tahlili `water_analyses`'da) |
| Sho'rlik chegarasi | `crops.salinity_ecw_threshold` | — |

`calc_parameters.farm_overridable = FALSE` — bunday parametrni dehqon
o'zgartira olmaydi (API 403 qaytaradi).

## Seed

Migratsiya `000011` boshlang'ich qiymatlarni yozadi:
- **makro tuzlar:** HPG `saltCompositions`
- **mikro tuzlar:** HPG wiki
- **monoeritma:** HPG `calcConcentrates` standartlari
- **kislotalar:** CRC Handbook, taxminiy
- **suv manbalari:** taxminiy

Hammasida `is_reviewed = FALSE`. Agronom qiymatni tasdiqlaganda `is_reviewed`
ni yoqadi, o'zgartirgan odam va vaqt `updated_by` va `updated_at`'ga yoziladi.

`TestSeed_ReproducesHPG` (integratsion test) seed qilingan tarkibdan
`inputs.Salts()` bilan yig'ilgan qiymat HPG standartiga **aynan** teng
ekanini tekshiradi.

## API

**Agronom** (`/api/v1/admin/catalog`, `RequireAgronomist`):

```
PATCH /admin/catalog/fertilizers/{code}      {"elements":{...}, "stock_g_l":..., "concentration_pct":..., "density_g_ml":..., "is_reviewed":true}
PUT   /admin/catalog/water-defaults/{source}?region_id=3   {"ec":1.6,"ca":140,"mg":45,"hco3_mmol":5.5,"note":"...","is_reviewed":true}
PATCH /admin/catalog/parameters/{key}         {"value":0.7,"is_reviewed":true}
```

**O'qish** (har bir foydalanuvchi): `GET /catalog/fertilizers`,
`GET /catalog/water-defaults`, `GET /catalog/parameters`.

**Xo'jalik** (egasi, `/api/v1/farms/{farmID}`):

```
GET    /fertilizers                 effektiv + default + overridden
PUT    /fertilizers/{code}          {"brand":"Yara","elements":{"Ca":19,"N_NO3":14.4,"N_NH4":1.1},"price_uzs_per_kg":14000}
DELETE /fertilizers/{code}          standartga qaytarish
GET    /parameters
PUT    /parameters/{key}            {"value":1.0}
DELETE /parameters/{key}
```

`PUT` — o'zgartirishni **to'liq almashtiradi**: yuborilmagan maydon
standartga qaytadi. `elements` berilsa, tarkib to'liq shu bo'ladi.

## Hisobga ulanish

`internal/modules/calc/inputs` effektiv qiymatlarni hisob kirishiga
aylantiradi:
- `Salts()` → `hpg.Salts`
- `MicroPercent()` → `hpg.Micro`
- `Acids(codes)` → `[]water.Acid`
- `ParamsFrom()` → qoldiq HCO₃, balans chegarasi, tuz rejimi, tarozi aniqligi

Kerakli qiymat topilmasa, xato qaytadi. Jimgina standart son qo'yilmaydi.
