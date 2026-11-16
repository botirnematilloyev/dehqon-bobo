DELETE FROM fertilizer_elements WHERE fertilizer_id IN (
    SELECT id FROM fertilizers WHERE code IN (
        'CaNO3','KNO3','NH4NO3','MgNO3','CaCl2','MgSO4','KH2PO4','K2SO4',
        'FeChelate','MnSO4','H3BO3','ZnSO4','CuSO4','NH4Mo','CoSO4','Na2SiO3',
        'HNO3_38','HNO3_60','H3PO4_75','H3PO4_85','H2SO4_96','H2SO4_37'
    )
);
-- Seed o'g'itlari faqat hech qayerda ishlatilmagan bo'lsa o'chadi
-- (batch_items / farm_fertilizers RESTRICT).
DELETE FROM fertilizers WHERE code IN (
    'CaNO3','KNO3','NH4NO3','MgNO3','CaCl2','MgSO4','KH2PO4','K2SO4',
    'FeChelate','MnSO4','H3BO3','ZnSO4','CuSO4','NH4Mo','CoSO4','Na2SiO3',
    'HNO3_38','HNO3_60','H3PO4_75','H3PO4_85','H2SO4_96','H2SO4_37'
);

DROP TABLE IF EXISTS farm_calc_parameters;
DROP TABLE IF EXISTS calc_parameters;
DROP TABLE IF EXISTS water_source_defaults;

ALTER TABLE farm_fertilizers
    DROP COLUMN IF EXISTS stock_density_g_ml,
    DROP COLUMN IF EXISTS stock_g_l,
    DROP COLUMN IF EXISTS density_g_ml,
    DROP COLUMN IF EXISTS concentration_pct;

ALTER TABLE fertilizers
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS is_reviewed,
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS stock_g_l,
    DROP COLUMN IF EXISTS concentration_pct;
