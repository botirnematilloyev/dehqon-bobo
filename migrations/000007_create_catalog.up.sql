-- Ma'lumotnomalar: ekinlar, ekin profillari, o'g'itlar, belgilar.
-- name / hint JSONB: {"uz": lotin, "uz_cyr": kirill, "ru": rus}

CREATE TABLE IF NOT EXISTS crops (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code VARCHAR(32) UNIQUE NOT NULL,
    name JSONB NOT NULL,
    -- FAO-29 (Ayers & Westcot), Maas–Hoffman: sug'orish suvining EC chegarasi, dS/m.
    -- Undan yuqori bo'lsa hosil kamayishi boshlanadi.
    salinity_ecw_threshold NUMERIC(4,2),
    sort_order SMALLINT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

-- Ekin + bosqich uchun maqsad eritma. Barcha qiymatlar — mg/L (ppm).
-- Mikroelementlar ham mg/L (HPG'dagi µg/L / 1000).
CREATE TABLE IF NOT EXISTS crop_profiles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    crop_id INT NOT NULL REFERENCES crops(id) ON DELETE RESTRICT,
    stage SMALLINT NOT NULL,
    title VARCHAR(255),
    -- NULL = rasmiy profil; aks holda dehqonning shaxsiy nusxasi.
    owner_user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    -- "Retseptni olish" — qaysi profildan nusxa olingan.
    parent_id BIGINT REFERENCES crop_profiles(id) ON DELETE SET NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    source VARCHAR(255),

    nh4_n NUMERIC(8,3) NOT NULL,
    no3_n NUMERIC(8,3) NOT NULL,
    p NUMERIC(8,3) NOT NULL,
    k NUMERIC(8,3) NOT NULL,
    ca NUMERIC(8,3) NOT NULL,
    mg NUMERIC(8,3) NOT NULL,
    -- S va EC ion balansidan hisoblanadi; bu yerda faqat ko'rsatish uchun saqlanadi.
    s NUMERIC(8,3),
    cl NUMERIC(8,3) NOT NULL DEFAULT 0,
    fe NUMERIC(8,4) NOT NULL DEFAULT 0,
    mn NUMERIC(8,4) NOT NULL DEFAULT 0,
    b NUMERIC(8,4) NOT NULL DEFAULT 0,
    zn NUMERIC(8,4) NOT NULL DEFAULT 0,
    cu NUMERIC(8,4) NOT NULL DEFAULT 0,
    mo NUMERIC(8,4) NOT NULL DEFAULT 0,
    ec_target NUMERIC(4,2),
    ph_min NUMERIC(3,1) NOT NULL DEFAULT 5.5,
    ph_max NUMERIC(3,1) NOT NULL DEFAULT 6.5,

    reviewed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at BIGINT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    CONSTRAINT crop_profiles_nonneg_chk CHECK (
        nh4_n >= 0 AND no3_n >= 0 AND p >= 0 AND k >= 0 AND ca >= 0 AND mg >= 0
    )
);

COMMENT ON COLUMN crop_profiles.stage IS '1=kochat, 2=osish, 3=hosil';
COMMENT ON COLUMN crop_profiles.status IS '1=draft, 2=approved (agronom tasdiqlagan), 3=archived';

CREATE INDEX IF NOT EXISTS idx_crop_profiles_crop_stage ON crop_profiles(crop_id, stage);
CREATE INDEX IF NOT EXISTS idx_crop_profiles_owner ON crop_profiles(owner_user_id) WHERE owner_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_crop_profiles_parent ON crop_profiles(parent_id) WHERE parent_id IS NOT NULL;
-- Har bir ekin+bosqich uchun bitta faol rasmiy profil.
CREATE UNIQUE INDEX IF NOT EXISTS uq_crop_profiles_official
    ON crop_profiles(crop_id, stage) WHERE owner_user_id IS NULL AND status = 2;

CREATE TABLE IF NOT EXISTS fertilizers (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code VARCHAR(32) UNIQUE NOT NULL,
    name JSONB NOT NULL,
    formula VARCHAR(64) NOT NULL,
    kind SMALLINT NOT NULL,
    default_tank SMALLINT NOT NULL,
    -- 20 °C dagi eruvchanlik, g/L — A/B konsentrat ogohlantirishi uchun.
    solubility_g_l NUMERIC(7,1),
    -- Kislotalar uchun: zichlik g/mL va konsentratsiya %.
    density_g_ml NUMERIC(5,3),
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

COMMENT ON COLUMN fertilizers.kind IS '1=macro, 2=micro, 3=acid';
COMMENT ON COLUMN fertilizers.default_tank IS '1=A (Ca, nitratlar), 2=B (fosfat, sulfat, mikro), 3=C (kislota)';

CREATE TABLE IF NOT EXISTS fertilizer_elements (
    fertilizer_id INT NOT NULL REFERENCES fertilizers(id) ON DELETE CASCADE,
    element VARCHAR(8) NOT NULL,
    percent NUMERIC(6,3) NOT NULL,
    PRIMARY KEY (fertilizer_id, element),
    CONSTRAINT fertilizer_elements_element_chk CHECK (element IN (
        'N_NO3', 'N_NH4', 'P', 'K', 'Ca', 'Mg', 'S', 'Cl',
        'Fe', 'Mn', 'B', 'Zn', 'Cu', 'Mo', 'Co', 'Si'
    )),
    CONSTRAINT fertilizer_elements_percent_chk CHECK (percent > 0 AND percent <= 100)
);

-- Tanqislik belgilari (Marschner). Postlar va mavsum muammolari uchun teg.
CREATE TABLE IF NOT EXISTS symptoms (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code VARCHAR(32) UNIQUE NOT NULL,
    name JSONB NOT NULL,
    hint JSONB,
    source VARCHAR(255),
    sort_order SMALLINT NOT NULL DEFAULT 0
);
