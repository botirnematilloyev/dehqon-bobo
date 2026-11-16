-- Xo'jalik: birinchi sozlash + Profil ekranlari.

CREATE TABLE IF NOT EXISTS farms (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255),
    region_id INT REFERENCES regions(id),
    district_id INT REFERENCES districts(id),
    system_type SMALLINT NOT NULL DEFAULT 1,
    mixing_method SMALLINT NOT NULL DEFAULT 1,
    injector_ratio INT,
    stock_tank_l NUMERIC(7,1),
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    CONSTRAINT farms_ab_chk CHECK (
        mixing_method = 1 OR (injector_ratio > 0 AND stock_tank_l > 0)
    )
);

COMMENT ON COLUMN farms.system_type IS '1=issiqxona tuproq, 2=substrat/tuvak, 3=gidroponika, 4=ochiq dala';
COMMENT ON COLUMN farms.mixing_method IS '1=bakda, 2=A/B konsentrat + injektor';

CREATE INDEX IF NOT EXISTS idx_farms_owner_id ON farms(owner_id);
CREATE INDEX IF NOT EXISTS idx_farms_region_id ON farms(region_id);

-- "Issiqxonalarim"
CREATE TABLE IF NOT EXISTS plots (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    farm_id BIGINT NOT NULL REFERENCES farms(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    area_m2 NUMERIC(10,1),
    default_volume_l NUMERIC(8,1),
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_plots_farm_id ON plots(farm_id);

-- Suv tahlili. Tarix saqlanadi (eski retseptlar unga bog'langan),
-- lekin xo'jalikda bitta faol tahlil.
CREATE TABLE IF NOT EXISTS water_analyses (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    farm_id BIGINT NOT NULL REFERENCES farms(id) ON DELETE CASCADE,
    source SMALLINT NOT NULL,
    -- TRUE = foydalanuvchi "Bilmayman" dedi, manba bo'yicha taxminiy qiymat.
    is_estimate BOOLEAN NOT NULL DEFAULT TRUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    ec NUMERIC(5,2) NOT NULL,
    ph NUMERIC(3,1),
    ca NUMERIC(7,2) NOT NULL DEFAULT 0,
    mg NUMERIC(7,2) NOT NULL DEFAULT 0,
    na NUMERIC(7,2),
    k NUMERIC(7,2),
    cl NUMERIC(7,2),
    so4 NUMERIC(7,2),
    no3 NUMERIC(7,2),
    hco3_mmol NUMERIC(5,2) NOT NULL DEFAULT 0,
    lab_name VARCHAR(255),
    sampled_at BIGINT,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at BIGINT NOT NULL,
    CONSTRAINT water_analyses_nonneg_chk CHECK (ec >= 0 AND ca >= 0 AND mg >= 0 AND hco3_mmol >= 0)
);

COMMENT ON COLUMN water_analyses.source IS '1=quduq, 2=ariq/kanal, 3=vodoprovod, 4=filtr/osmos';
COMMENT ON COLUMN water_analyses.ca IS 'mg/L; na, k, cl, so4, no3 ham mg/L; hco3 — mmol/L';

CREATE INDEX IF NOT EXISTS idx_water_analyses_farm_id ON water_analyses(farm_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_water_analyses_active ON water_analyses(farm_id) WHERE is_active;

-- Dehqonning o'g'itlari: qopdagi yorliq foizlari va narxi.
CREATE TABLE IF NOT EXISTS farm_fertilizers (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    farm_id BIGINT NOT NULL REFERENCES farms(id) ON DELETE CASCADE,
    fertilizer_id INT NOT NULL REFERENCES fertilizers(id) ON DELETE RESTRICT,
    brand VARCHAR(255),
    -- {"Ca": 16.9, "N_NO3": 11.8} — fertilizer_elements'ni ustidan yozadi; NULL = standart.
    label_percents JSONB,
    price_uzs_per_kg BIGINT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    UNIQUE (farm_id, fertilizer_id)
);
