-- Hisobda ishlatiladigan, sozlanadigan barcha qiymatlar bazada.
--
-- Uch qatlam (HPG web'dagi kabi har bir qiymatni o'zgartirish mumkin):
--   1. standart — agronom belgilaydi (fertilizers, calc_parameters,
--      water_source_defaults);
--   2. xo'jalik — dehqon o'zi uchun o'zgartiradi (farm_fertilizers,
--      farm_calc_parameters); yo'q bo'lsa standart ishlatiladi;
--   3. bitta hisob — so'rovda berilgan qiymat (saqlanmaydi).
--
-- Kodda faqat o'zgarmas narsalar qoladi: molyar massalar, kislota proton
-- soni, HPG'ning empirik formulalari (EC koeffitsienti, zichlik polinomi).
--
-- Seed'dagi qiymatlar manbasi `source` ustunida; is_reviewed = FALSE —
-- agronom hali tasdiqlamagan.

-- ---------- o'g'itlar: monoeritma, kislota kuchi, tasdiq ----------
ALTER TABLE fertilizers
    ADD COLUMN IF NOT EXISTS concentration_pct NUMERIC(5,2),
    ADD COLUMN IF NOT EXISTS stock_g_l NUMERIC(7,1),
    ADD COLUMN IF NOT EXISTS source VARCHAR(255),
    ADD COLUMN IF NOT EXISTS is_reviewed BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS updated_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS updated_at BIGINT;

COMMENT ON COLUMN fertilizers.concentration_pct IS 'Kislotalar: massa ulushi, % (yorliqdan)';
COMMENT ON COLUMN fertilizers.density_g_ml IS 'Kislotalar: zichlik g/mL, 20 °C (yorliqdan)';
COMMENT ON COLUMN fertilizers.stock_g_l IS 'Monoeritma (konsentrat) standart konsentratsiyasi, g/L';

ALTER TABLE farm_fertilizers
    ADD COLUMN IF NOT EXISTS concentration_pct NUMERIC(5,2),
    ADD COLUMN IF NOT EXISTS density_g_ml NUMERIC(5,3),
    ADD COLUMN IF NOT EXISTS stock_g_l NUMERIC(7,1),
    ADD COLUMN IF NOT EXISTS stock_density_g_ml NUMERIC(5,4);

COMMENT ON COLUMN farm_fertilizers.stock_density_g_ml IS
    'Monoeritma zichligi; NULL = HPG formulasi bilan stock_g_l dan hisoblanadi';

-- ---------- suv manbalari: taxminiy tarkib ----------
CREATE TABLE IF NOT EXISTS water_source_defaults (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source SMALLINT NOT NULL,
    -- NULL = butun mamlakat; to'ldirilgan = shu viloyat uchun aniqroq.
    region_id INT REFERENCES regions(id) ON DELETE CASCADE,
    ec NUMERIC(5,2) NOT NULL,
    ph NUMERIC(3,1),
    ca NUMERIC(7,2) NOT NULL DEFAULT 0,
    mg NUMERIC(7,2) NOT NULL DEFAULT 0,
    na NUMERIC(7,2) NOT NULL DEFAULT 0,
    k NUMERIC(7,2) NOT NULL DEFAULT 0,
    cl NUMERIC(7,2) NOT NULL DEFAULT 0,
    so4 NUMERIC(7,2) NOT NULL DEFAULT 0,
    no3 NUMERIC(7,2) NOT NULL DEFAULT 0,
    hco3_mmol NUMERIC(5,2) NOT NULL DEFAULT 0,
    source_note VARCHAR(255),
    is_reviewed BOOLEAN NOT NULL DEFAULT FALSE,
    updated_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    updated_at BIGINT NOT NULL,
    CONSTRAINT water_source_defaults_nonneg_chk CHECK (
        ec >= 0 AND ca >= 0 AND mg >= 0 AND na >= 0 AND k >= 0 AND cl >= 0 AND so4 >= 0 AND no3 >= 0 AND hco3_mmol >= 0
    )
);

COMMENT ON COLUMN water_source_defaults.source IS '1=quduq, 2=ariq/kanal, 3=vodoprovod, 4=filtr/osmos';

CREATE UNIQUE INDEX IF NOT EXISTS uq_water_source_defaults_global
    ON water_source_defaults(source) WHERE region_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_water_source_defaults_region
    ON water_source_defaults(source, region_id) WHERE region_id IS NOT NULL;

-- ---------- son parametrlar ----------
CREATE TABLE IF NOT EXISTS calc_parameters (
    key VARCHAR(64) PRIMARY KEY,
    value NUMERIC(12,4) NOT NULL,
    min_value NUMERIC(12,4) NOT NULL,
    max_value NUMERIC(12,4) NOT NULL,
    unit VARCHAR(16),
    description JSONB NOT NULL,
    source VARCHAR(255),
    -- FALSE = faqat agronom o'zgartiradi (xo'jalik darajasida emas).
    farm_overridable BOOLEAN NOT NULL DEFAULT TRUE,
    is_reviewed BOOLEAN NOT NULL DEFAULT FALSE,
    updated_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    updated_at BIGINT NOT NULL,
    CONSTRAINT calc_parameters_range_chk CHECK (min_value <= value AND value <= max_value)
);

CREATE TABLE IF NOT EXISTS farm_calc_parameters (
    farm_id BIGINT NOT NULL REFERENCES farms(id) ON DELETE CASCADE,
    key VARCHAR(64) NOT NULL REFERENCES calc_parameters(key) ON DELETE CASCADE,
    value NUMERIC(12,4) NOT NULL,
    updated_at BIGINT NOT NULL,
    PRIMARY KEY (farm_id, key)
);

-- ======================= SEED =======================
-- updated_at: 2026-11-16 00:00 UTC

INSERT INTO calc_parameters (key, value, min_value, max_value, unit, description, source, farm_overridable, updated_at) VALUES
    ('residual_hco3_mmol', 0.5, 0, 3, 'mmol/L',
     '{"uz":"Kislotadan keyin qoldiriladigan bikarbonat (pH bufer)","ru":"Остаточный бикарбонат после подкисления"}',
     'Sonneveld & Voogt 2009', TRUE, 1763251200),
    ('water_ion_balance_tolerance_pct', 10, 1, 50, '%',
     '{"uz":"Suv tahlilida kation/anion farqi chegarasi","ru":"Допустимый дисбаланс ионов в анализе воды"}',
     'laboratoriya amaliyoti', FALSE, 1763251200),
    ('use_k2so4', 1, 0, 1, 'bool',
     '{"uz":"Kaliy sulfat ishlatiladi (HPG chK2SO4)","ru":"Использовать K2SO4"}',
     'WEGA-HPG', TRUE, 1763251200),
    ('use_mgno3', 0, 0, 1, 'bool',
     '{"uz":"Magniy selitra ishlatiladi (HPG chMgNO3)","ru":"Использовать Mg(NO3)2"}',
     'WEGA-HPG', TRUE, 1763251200),
    ('weight_decimals', -1, -1, 4, 'digits',
     '{"uz":"Tarozi aniqligi (kasr xonasi); -1 = hajmga qarab (HPG)","ru":"Точность весов; -1 = авто (HPG)"}',
     'WEGA-HPG calcWeight', TRUE, 1763251200)
ON CONFLICT (key) DO NOTHING;

-- Suv manbalari: TAXMIN, mamlakat bo'yicha (region_id NULL). Agronom
-- viloyatlar bo'yicha aniq qiymat qo'shadi.
INSERT INTO water_source_defaults (source, region_id, ec, ca, mg, hco3_mmol, source_note, updated_at) VALUES
    (1, NULL, 1.00, 80, 30, 4.0, 'taxminiy, agronom tasdig''i kutilmoqda', 1763251200),
    (2, NULL, 0.90, 70, 25, 3.5, 'taxminiy, agronom tasdig''i kutilmoqda', 1763251200),
    (3, NULL, 0.50, 50, 15, 3.0, 'taxminiy, agronom tasdig''i kutilmoqda', 1763251200),
    (4, NULL, 0.05,  2,  1, 0.3, 'taxminiy, agronom tasdig''i kutilmoqda', 1763251200);

-- O'g'itlar. kind: 1=makro, 2=mikro, 3=kislota. default_tank: 1=A, 2=B, 3=C.
-- Makro tarkibi — HPG saltCompositions (toza tuzlar uchun stexiometrik).
-- Mikro foizlari — HPG wiki "Micro". Monoeritma g/L — HPG calcConcentrates
-- standartlari. Eruvchanlik — HPG wiki/FAQ (20–25 °C).
-- Kislota zichligi — CRC Handbook (20 °C), taxminiy.
INSERT INTO fertilizers (code, name, formula, kind, default_tank, solubility_g_l, stock_g_l, concentration_pct, density_g_ml, source, updated_at) VALUES
    ('CaNO3',  '{"uz":"Kalsiy selitra","uz_cyr":"Кальций селитра","ru":"Кальциевая селитра"}', 'Ca(NO3)2·4H2O', 1, 1, 1290, 600, NULL, NULL, 'WEGA-HPG', 1763251200),
    ('KNO3',   '{"uz":"Kaliy selitra","uz_cyr":"Калий селитра","ru":"Калиевая селитра"}', 'KNO3', 1, 1, 316, 250, NULL, NULL, 'WEGA-HPG', 1763251200),
    ('NH4NO3', '{"uz":"Ammiakli selitra","uz_cyr":"Аммиакли селитра","ru":"Аммиачная селитра"}', 'NH4NO3', 1, 1, 1920, 100, NULL, NULL, 'WEGA-HPG', 1763251200),
    ('MgNO3',  '{"uz":"Magniy selitra","uz_cyr":"Магний селитра","ru":"Магниевая селитра"}', 'Mg(NO3)2·6H2O', 1, 1, NULL, 500, NULL, NULL, 'WEGA-HPG', 1763251200),
    ('CaCl2',  '{"uz":"Kalsiy xlorid","uz_cyr":"Кальций хлорид","ru":"Хлорид кальция"}', 'CaCl2·6H2O', 1, 1, NULL, 100, NULL, NULL, 'WEGA-HPG', 1763251200),
    ('MgSO4',  '{"uz":"Magniy sulfat","uz_cyr":"Магний сульфат","ru":"Сульфат магния"}', 'MgSO4·7H2O', 1, 2, 710, 600, NULL, NULL, 'WEGA-HPG', 1763251200),
    ('KH2PO4', '{"uz":"Monokaliy fosfat","uz_cyr":"Монокалий фосфат","ru":"Монофосфат калия"}', 'KH2PO4', 1, 2, 330, 150, NULL, NULL, 'WEGA-HPG', 1763251200),
    ('K2SO4',  '{"uz":"Kaliy sulfat","uz_cyr":"Калий сульфат","ru":"Сульфат калия"}', 'K2SO4', 1, 2, 100, 100, NULL, NULL, 'WEGA-HPG', 1763251200),

    ('FeChelate', '{"uz":"Temir xelati","uz_cyr":"Темир хелати","ru":"Хелат железа"}', 'Fe-DTPA', 2, 2, NULL, 10, NULL, NULL, 'WEGA-HPG wiki', 1763251200),
    ('MnSO4',  '{"uz":"Marganets sulfat","uz_cyr":"Марганец сульфат","ru":"Сульфат марганца"}', 'MnSO4·H2O', 2, 2, NULL, 10, NULL, NULL, 'WEGA-HPG wiki', 1763251200),
    ('H3BO3',  '{"uz":"Bor kislotasi","uz_cyr":"Бор кислотаси","ru":"Борная кислота"}', 'H3BO3', 2, 2, NULL, 10, NULL, NULL, 'WEGA-HPG wiki', 1763251200),
    ('ZnSO4',  '{"uz":"Rux sulfat","uz_cyr":"Рух сульфат","ru":"Сульфат цинка"}', 'ZnSO4·7H2O', 2, 2, NULL, 10, NULL, NULL, 'WEGA-HPG wiki', 1763251200),
    ('CuSO4',  '{"uz":"Mis kuporosi","uz_cyr":"Мис купороси","ru":"Медный купорос"}', 'CuSO4·5H2O', 2, 2, NULL, 10, NULL, NULL, 'WEGA-HPG wiki', 1763251200),
    ('NH4Mo',  '{"uz":"Ammoniy molibdat","uz_cyr":"Аммоний молибдат","ru":"Молибдат аммония"}', '(NH4)6Mo7O24·4H2O', 2, 2, NULL, 10, NULL, NULL, 'WEGA-HPG wiki', 1763251200),
    ('CoSO4',  '{"uz":"Kobalt sulfat","uz_cyr":"Кобальт сульфат","ru":"Сульфат кобальта"}', 'CoSO4·7H2O', 2, 2, NULL, 10, NULL, NULL, 'WEGA-HPG wiki', 1763251200),
    ('Na2SiO3', '{"uz":"Natriy silikat","uz_cyr":"Натрий силикат","ru":"Силикат натрия"}', 'Na2SiO3·5H2O', 2, 2, NULL, 10, NULL, NULL, 'WEGA-HPG wiki', 1763251200),

    ('HNO3_38',  '{"uz":"Azot kislotasi 38%","ru":"Азотная кислота 38%"}', 'HNO3', 3, 3, NULL, NULL, 38, 1.230, 'CRC Handbook, taxminiy', 1763251200),
    ('HNO3_60',  '{"uz":"Azot kislotasi 60%","ru":"Азотная кислота 60%"}', 'HNO3', 3, 3, NULL, NULL, 60, 1.370, 'CRC Handbook, taxminiy', 1763251200),
    ('H3PO4_75', '{"uz":"Ortofosfat kislota 75%","ru":"Ортофосфорная кислота 75%"}', 'H3PO4', 3, 3, NULL, NULL, 75, 1.570, 'CRC Handbook, taxminiy', 1763251200),
    ('H3PO4_85', '{"uz":"Ortofosfat kislota 85%","ru":"Ортофосфорная кислота 85%"}', 'H3PO4', 3, 3, NULL, NULL, 85, 1.690, 'CRC Handbook, taxminiy', 1763251200),
    ('H2SO4_96', '{"uz":"Sulfat kislota 96%","ru":"Серная кислота 96%"}', 'H2SO4', 3, 3, NULL, NULL, 96, 1.840, 'CRC Handbook, taxminiy', 1763251200),
    ('H2SO4_37', '{"uz":"Sulfat kislota 37% (akkumulyator)","ru":"Серная кислота 37%"}', 'H2SO4', 3, 3, NULL, NULL, 37, 1.280, 'CRC Handbook, taxminiy', 1763251200)
ON CONFLICT (code) DO NOTHING;

INSERT INTO fertilizer_elements (fertilizer_id, element, percent)
SELECT f.id, e.element, e.percent
FROM (VALUES
    ('CaNO3', 'Ca', 16.972), ('CaNO3', 'N_NO3', 11.863),
    ('KNO3', 'K', 38.672), ('KNO3', 'N_NO3', 13.854),
    ('NH4NO3', 'N_NH4', 17.499), ('NH4NO3', 'N_NO3', 17.499),
    ('MgNO3', 'Mg', 9.479), ('MgNO3', 'N_NO3', 10.925),
    ('CaCl2', 'Ca', 18.294), ('CaCl2', 'Cl', 32.366),
    ('MgSO4', 'Mg', 9.861), ('MgSO4', 'S', 13.010),
    ('KH2PO4', 'K', 28.731), ('KH2PO4', 'P', 22.761),
    ('K2SO4', 'K', 44.874), ('K2SO4', 'S', 18.401),
    ('FeChelate', 'Fe', 13.0),
    ('MnSO4', 'Mn', 32.5),
    ('H3BO3', 'B', 17.5),
    ('ZnSO4', 'Zn', 22.7),
    ('CuSO4', 'Cu', 25.5),
    ('NH4Mo', 'Mo', 54.3),
    ('CoSO4', 'Co', 21.0),
    ('Na2SiO3', 'Si', 13.2)
) AS e(code, element, percent)
JOIN fertilizers f ON f.code = e.code
ON CONFLICT (fertilizer_id, element) DO NOTHING;
