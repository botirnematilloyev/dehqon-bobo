# Dehqon Bobo — ma'lumotlar bazasi (ERD)

PostgreSQL 16+. Migratsiyalar: `migrations/` (golang-migrate, `binary` loyihasi bilan bir xil format).

## Umumiy qoidalar (binary loyihasidan)

- **ID** — `BIGINT GENERATED ALWAYS AS IDENTITY` (ma'lumotnomalarda `INT`).
- **Vaqt** — `BIGINT` unix soniya (`created_at`, `updated_at`, `*_at`). Go tomonida `time.Now().Unix()`.
- **Enum** — `SMALLINT` + `COMMENT ON COLUMN`; qiymatlar Go'dagi `domain` enum'larida takrorlanadi.
- **Ko'p tilli nom** — `JSONB {"uz": lotin, "uz_cyr": kirill, "ru": rus}` (regions, districts, crops, fertilizers, symptoms).
- **OTP kodlar va urinishlar** — jadvalda emas, **Redis**'da (TTL bilan). Binary'dagi `AuthService` shunday ishlaydi.
- **Kimyoviy miqdorlar** — `NUMERIC`, float emas: retsept qayta hisoblanganda bir xil chiqishi kerak.

## Diagramma

```mermaid
erDiagram
    %% ===== 1. Identity (binary loyihasidan) =====
    regions ||--o{ districts : has
    regions ||--o{ users : lives_in
    districts ||--o{ users : lives_in
    users ||--o{ user_linked_accounts : "login via"
    users ||--o{ user_sessions : has

    %% ===== 2. Ma'lumotnomalar =====
    crops ||--o{ crop_profiles : has
    crop_profiles ||--o{ crop_profiles : "fork (parent_id)"
    users ||--o{ crop_profiles : "owns / reviews"
    fertilizers ||--o{ fertilizer_elements : contains

    %% ===== 3. Xo'jalik =====
    users ||--o{ farms : owns
    regions ||--o{ farms : located
    farms ||--o{ plots : has
    farms ||--o{ water_analyses : has
    farms ||--o{ farm_fertilizers : stocks
    fertilizers ||--o{ farm_fertilizers : "brand/price"

    %% ===== 4. Mavsum va jurnal =====
    plots ||--o{ seasons : grows
    crops ||--o{ seasons : crop
    seasons ||--o{ season_issues : had
    symptoms ||--o{ season_issues : kind
    farms ||--o{ solution_batches : mixes
    plots |o--o{ solution_batches : for
    seasons |o--o{ solution_batches : during
    crop_profiles ||--o{ solution_batches : "recipe from"
    water_analyses ||--o{ solution_batches : "water used"
    solution_batches ||--o{ batch_items : contains
    fertilizers ||--o{ batch_items : salt
    solution_batches |o--o{ measurements : "checked by"
    plots |o--o{ measurements : at

    %% ===== 5. Hamjamiyat =====
    users ||--o{ posts : writes
    seasons |o--o{ posts : "result of"
    crop_profiles |o--o{ posts : "shares recipe"
    posts ||--o{ post_media : has
    posts ||--o{ post_symptoms : tagged
    symptoms ||--o{ post_symptoms : tag
    posts ||--o{ post_reactions : gets
    posts ||--o{ post_comments : gets
    post_comments |o--o{ post_comments : reply
    posts ||--o{ post_reports : reported

    users {
        bigint id PK
        varchar username UK
        varchar full_name
        smallint role "1 dehqon, 2 agronom, 3 admin"
        varchar locale "uz | uz_cyr | ru"
        int region_id FK
        int district_id FK
        bigint created_at
    }
    user_linked_accounts {
        bigint id PK
        bigint user_id FK
        smallint provider "1 phone, 2 email, 3 telegram"
        varchar identifier
        boolean is_verified
    }
    user_sessions {
        bigint id PK
        bigint user_id FK
        varchar refresh_token_hash UK
        varchar previous_refresh_token_hash
        varchar fcm_token
        smallint platform
        boolean is_revoked
        bigint expires_at
    }
    crops {
        int id PK
        varchar code UK
        jsonb name
        numeric salinity_ecw_threshold "FAO-29"
    }
    crop_profiles {
        bigint id PK
        int crop_id FK
        smallint stage "1 kochat, 2 osish, 3 hosil"
        bigint owner_user_id FK "NULL = rasmiy"
        bigint parent_id FK
        smallint status "1 draft, 2 approved, 3 archived"
        numeric no3_n "mg/L"
        numeric nh4_n
        numeric p
        numeric k
        numeric ca
        numeric mg
        numeric fe_mn_b_zn_cu_mo "mg/L"
        numeric ec_target
        bigint reviewed_by FK
    }
    fertilizers {
        int id PK
        varchar code UK
        jsonb name
        varchar formula
        smallint kind "1 macro, 2 micro, 3 acid"
        smallint default_tank "1 A, 2 B, 3 C"
        numeric solubility_g_l
    }
    fertilizer_elements {
        int fertilizer_id PK
        varchar element PK "N_NO3, P, Ca ..."
        numeric percent
    }
    farms {
        bigint id PK
        bigint owner_id FK
        int region_id FK
        smallint system_type "tuproq, substrat, gidro, dala"
        smallint mixing_method "1 bak, 2 A/B"
        int injector_ratio
        numeric stock_tank_l
    }
    plots {
        bigint id PK
        bigint farm_id FK
        varchar name "1-issiqxona"
        numeric area_m2
    }
    water_analyses {
        bigint id PK
        bigint farm_id FK
        smallint source "quduq, kanal, vodoprovod, filtr"
        boolean is_estimate
        boolean is_active
        numeric ec
        numeric ca
        numeric mg
        numeric hco3_mmol
    }
    farm_fertilizers {
        bigint id PK
        bigint farm_id FK
        int fertilizer_id FK
        varchar brand
        jsonb label_percents
        bigint price_uzs_per_kg
    }
    seasons {
        bigint id PK
        bigint plot_id FK
        int crop_id FK
        smallint current_stage
        bigint started_at
        bigint ended_at
        numeric yield_kg
        bigint fertilizer_cost_uzs
    }
    solution_batches {
        bigint id PK
        bigint farm_id FK
        bigint plot_id FK
        bigint season_id FK
        bigint crop_profile_id FK
        bigint water_analysis_id FK
        smallint method
        numeric volume_l
        numeric target_ec
        varchar calc_version
        jsonb calc_snapshot
        bigint mixed_at
    }
    batch_items {
        bigint id PK
        bigint batch_id FK
        int fertilizer_id FK
        smallint tank "0 bak, 1 A, 2 B, 3 C"
        numeric amount_g
        numeric amount_ml
    }
    measurements {
        bigint id PK
        bigint farm_id FK
        bigint plot_id FK
        bigint batch_id FK
        numeric ec_in
        numeric ph_in
        numeric ec_drain
        bigint measured_at
    }
    symptoms {
        int id PK
        varchar code UK
        jsonb name
        jsonb hint
    }
    posts {
        bigint id PK
        bigint author_id FK
        smallint type "1 natija, 2 muammo, 3 savol"
        smallint trust_level "1 fikr, 2 data, 3 agronom"
        smallint status
        bigint season_id FK
        bigint crop_profile_id FK
        int crop_id FK
        int region_id FK
        smallint water_source
        jsonb data_snapshot
        int useful_count
        int comment_count
    }
    post_comments {
        bigint id PK
        bigint post_id FK
        bigint author_id FK
        bigint parent_id FK
        boolean is_accepted
    }
    post_reactions {
        bigint post_id PK
        bigint user_id PK
        smallint kind PK
    }
    post_media {
        bigint id PK
        bigint post_id FK
        smallint kind "1 rasm, 2 ovoz"
        varchar url
    }
    post_symptoms {
        bigint post_id PK
        int symptom_id PK
    }
    post_reports {
        bigint id PK
        bigint post_id FK
        bigint reporter_id FK
        smallint reason
        smallint status
    }
```

## Bloklar

### 1. Identity — binary'dan ko'chirildi

| Jadval | Binary'dan farqi |
|---|---|
| `regions`, `districts` | O'zgarishsiz. Seed (`000006_seed_locations`) ham ko'chirildi. |
| `users` | `mentor_id` olib tashlandi. `role` ma'nosi: 1 dehqon, 2 agronom, 3 admin. Yangi: `locale`, `avatar_url`, `onboarded_at`. |
| `user_linked_accounts` | O'zgarishsiz (telefon / email / Telegram). |
| `user_sessions` | O'zgarishsiz. `previous_refresh_token_hash` va `expires_at` indeksi binary'ning 7 va 8-migratsiyalaridan birlashtirildi. |

OTP, urinishlar soni va Telegram login kodlari Redis'da saqlanadi. Ular uchun jadval yo'q.

### 2. Ma'lumotnomalar

- **`crops`**: ekin va FAO-29 bo'yicha suv sho'rlik chegarasi (`salinity_ecw_threshold`, dS/m). Bu qiymat retsept ekranidagi ogohlantirish uchun ishlatiladi.
- **`crop_profiles`**: ekin va bosqich uchun maqsad (mg/L) hamda EC.
  - `owner_user_id IS NULL` — rasmiy profil (agronom tasdiqlagan).
  - `owner_user_id` to'ldirilgan — dehqonning o'z profili. "Retseptni olish" bosilganda `parent_id` orqali nusxa (fork) yaratiladi.
  - `status = 2` va `reviewed_by` — agronom tasdig'i. Shu belgi ilovada "Agronom ko'rdi ✓" bo'lib chiqadi.
- **`fertilizers` + `fertilizer_elements`**: tuz va uning element tarkibi (%). Qiymatlar HPG'dagi standart foizlar. `default_tank` A/B/C bakka taqsimlash uchun, `solubility_g_l` konsentrat ogohlantirishi uchun.

### 3. Xo'jalik

- **`farms`**: birinchi sozlash va Profil ekranlari. Unda viloyat, tizim va aralashtirish usuli saqlanadi.
- **`plots`**: "Issiqxonalarim" ro'yxati.
- **`water_analyses`**: suv tahlili. Har bir xo'jalikda bitta faol tahlil bo'ladi (partial unique index). Eski tahlillar o'chirilmaydi, chunki eski retseptlar ularga bog'langan.
- **`farm_fertilizers`**: dehqon qopdagi yorliqdan kiritgan foizlar va narx (so'm/kg).

### 4. Mavsum va jurnal

- **`seasons`**: bitta issiqxonadagi bitta ekin sikli. Mavsum yakunida hosil (kg), maydon va xarajat to'ldiriladi, natija kartochkasi shulardan tuziladi.
- **`solution_batches`**: tayyorlangan har bir eritma. `calc_snapshot` hisob natijasini o'zgarmas holda saqlaydi: ppm, tuzlar, kislota va EC. `calc_version` esa formulalar versiyasi. Formula keyin tuzatilsa ham, eski jurnal yozuvlari o'zgarmaydi.
- **`batch_items`**: qaysi bakka qaysi tuzdan necha gramm solingani.
- **`measurements`**: EC, pH va drenaj o'lchovlari. Eritmaga yoki issiqxonaga bog'lanadi, bu ikkala holatni ham qamrab oladi: eritma tayyorlangandan keyingi o'lchov va bosh sahifadan qilingan oddiy o'lchov.

### 5. Hamjamiyat

- **`posts`**: postning turi (natija, muammo yoki savol) va ishonch darajasi.
  - `data_snapshot` — kartochkada ko'rinadigan raqamlar: kg/m², o'rtacha EC, o'lchovlar soni.
  - `region_id` va `water_source` post yozilgan paytdagi holatdan nusxa olinadi, shuning uchun dehqon keyin viloyatini o'zgartirsa ham "Mening sharoitim" filtri to'g'ri ishlaydi.
- **`post_reactions`**: "Foydali" reaksiyasi. `useful_count` va `comment_count` lenta tez yuklanishi uchun saqlanadigan hisoblagichlar.
- **`post_comments`**: izoh va javoblar. `is_accepted` — tasdiqlangan javob; bitta postda faqat bittasi bo'lishi partial unique index bilan ta'minlanadi.
- **`symptoms` + `post_symptoms`** (shuningdek `season_issues`): belgilar ro'yxati (Marschner manbasi bo'yicha). Postlarni belgi bo'yicha qidirish uchun.
- **`post_reports`**: moderatsiya — shikoyatlar.

## Keyingi bosqichga qoldirildi

- `notifications` va `reminders`. Hozircha `user_sessions.fcm_token` yetarli.
- `weather_cache`. Ob-havo API natijasi Redis'da saqlanadi.
- Agregat statistika, masalan "viloyatda shu profil bilan mediana hosil". Uni `seasons` asosida materialized view qilib yasaymiz.
