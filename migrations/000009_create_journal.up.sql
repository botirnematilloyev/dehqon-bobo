-- Mavsum, eritmalar (jurnal) va o'lchovlar.

CREATE TABLE IF NOT EXISTS seasons (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    plot_id BIGINT NOT NULL REFERENCES plots(id) ON DELETE CASCADE,
    crop_id INT NOT NULL REFERENCES crops(id) ON DELETE RESTRICT,
    current_stage SMALLINT NOT NULL DEFAULT 1,
    started_at BIGINT NOT NULL,
    ended_at BIGINT,
    -- Mavsum yakunida to'ldiriladi:
    yield_kg NUMERIC(12,1),
    area_m2 NUMERIC(10,1),
    fertilizer_cost_uzs BIGINT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    CONSTRAINT seasons_dates_chk CHECK (ended_at IS NULL OR ended_at >= started_at)
);

COMMENT ON COLUMN seasons.current_stage IS '1=kochat, 2=osish, 3=hosil';

CREATE INDEX IF NOT EXISTS idx_seasons_plot_id ON seasons(plot_id);
-- Issiqxonada bir vaqtda bitta ochiq mavsum.
CREATE UNIQUE INDEX IF NOT EXISTS uq_seasons_open ON seasons(plot_id) WHERE ended_at IS NULL;

CREATE TABLE IF NOT EXISTS season_issues (
    season_id BIGINT NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    symptom_id INT NOT NULL REFERENCES symptoms(id) ON DELETE RESTRICT,
    PRIMARY KEY (season_id, symptom_id)
);

-- Har bir tayyorlangan eritma = jurnal yozuvi.
CREATE TABLE IF NOT EXISTS solution_batches (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    farm_id BIGINT NOT NULL REFERENCES farms(id) ON DELETE CASCADE,
    plot_id BIGINT REFERENCES plots(id) ON DELETE SET NULL,
    season_id BIGINT REFERENCES seasons(id) ON DELETE SET NULL,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    crop_profile_id BIGINT NOT NULL REFERENCES crop_profiles(id) ON DELETE RESTRICT,
    water_analysis_id BIGINT REFERENCES water_analyses(id) ON DELETE SET NULL,
    method SMALLINT NOT NULL,
    volume_l NUMERIC(10,1),
    stock_tank_l NUMERIC(7,1),
    injector_ratio INT,
    target_ec NUMERIC(4,2) NOT NULL,
    -- Hisob formulalari versiyasi va natija (ppm, tuzlar, kislota, ogohlantirishlar).
    -- O'zgarmas: formula keyin tuzatilsa ham eski jurnal o'zgarmaydi.
    calc_version VARCHAR(32) NOT NULL,
    calc_snapshot JSONB NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    mixed_at BIGINT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    CONSTRAINT solution_batches_volume_chk CHECK (
        (method = 1 AND volume_l > 0) OR (method = 2 AND stock_tank_l > 0 AND injector_ratio > 0)
    )
);

COMMENT ON COLUMN solution_batches.method IS '1=bakda, 2=A/B konsentrat';
COMMENT ON COLUMN solution_batches.status IS '1=retsept (hali tayyorlanmagan), 2=tayyorlandi';

CREATE INDEX IF NOT EXISTS idx_solution_batches_farm_created ON solution_batches(farm_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_solution_batches_season_id ON solution_batches(season_id);
CREATE INDEX IF NOT EXISTS idx_solution_batches_profile_id ON solution_batches(crop_profile_id);

CREATE TABLE IF NOT EXISTS batch_items (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    batch_id BIGINT NOT NULL REFERENCES solution_batches(id) ON DELETE CASCADE,
    fertilizer_id INT NOT NULL REFERENCES fertilizers(id) ON DELETE RESTRICT,
    tank SMALLINT NOT NULL,
    amount_g NUMERIC(12,3),
    amount_ml NUMERIC(12,1),
    position SMALLINT NOT NULL DEFAULT 0,
    CONSTRAINT batch_items_amount_chk CHECK (amount_g IS NOT NULL OR amount_ml IS NOT NULL)
);

COMMENT ON COLUMN batch_items.tank IS '0=bak (bitta), 1=A, 2=B, 3=C (kislota)';

CREATE INDEX IF NOT EXISTS idx_batch_items_batch_id ON batch_items(batch_id);

-- EC/pH/drenaj o'lchovlari: eritmadan keyin yoki bosh sahifadan alohida.
CREATE TABLE IF NOT EXISTS measurements (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    farm_id BIGINT NOT NULL REFERENCES farms(id) ON DELETE CASCADE,
    plot_id BIGINT REFERENCES plots(id) ON DELETE SET NULL,
    batch_id BIGINT REFERENCES solution_batches(id) ON DELETE SET NULL,
    measured_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ec_in NUMERIC(4,2),
    ph_in NUMERIC(3,1),
    ec_drain NUMERIC(4,2),
    ph_drain NUMERIC(3,1),
    drain_pct NUMERIC(4,1),
    solution_temp_c NUMERIC(4,1),
    note TEXT,
    measured_at BIGINT NOT NULL,
    created_at BIGINT NOT NULL,
    CONSTRAINT measurements_any_chk CHECK (ec_in IS NOT NULL OR ph_in IS NOT NULL OR ec_drain IS NOT NULL),
    CONSTRAINT measurements_ph_chk CHECK ((ph_in IS NULL OR ph_in BETWEEN 0 AND 14) AND (ph_drain IS NULL OR ph_drain BETWEEN 0 AND 14))
);

CREATE INDEX IF NOT EXISTS idx_measurements_farm_measured ON measurements(farm_id, measured_at DESC);
CREATE INDEX IF NOT EXISTS idx_measurements_plot_measured ON measurements(plot_id, measured_at DESC);
CREATE INDEX IF NOT EXISTS idx_measurements_batch_id ON measurements(batch_id);
