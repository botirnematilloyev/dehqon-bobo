-- Ekinlar va maqsad eritma profillari (agronom standarti).
--
-- Manbalar:
--   * salinity_ecw_threshold — FAO-29 (Ayers & Westcot, 1985), 4-jadval:
--     hosil 100% saqlanadigan sug'orish suvi EC chegarasi (ECw, dS/m).
--   * Profillar — Sonneveld & Voogt (2009) "Plant Nutrition of Greenhouse
--     Crops" / Sonneveld & Straver (1994): mineral vata (ochiq tizim) uchun
--     standart oziq eritma. mmol/L va µmol/L qiymatlari mg/L ga o'girilgan
--     (HPG'dagi molyar massalar bilan): ion × molyar massa.
--     Kation va anionlar ekvivalentligi: pomidor 26.3 = 26.3, bodring
--     20.0 = 20.0 meq/L (P — H2PO4⁻, bir valentli).
--
-- Standart eritma butun mavsum uchun berilgan, shuning uchun 2 (o'sish) va
-- 3 (hosil) bosqichlarga bir xil yoziladi. 1 (ko'chat) bosqichi va boshqa
-- ekinlar profillari agronom tomonidan API orqali kiritiladi.
--
-- status = 1 (draft): agronom hali tasdiqlamagan. Tasdiqlash —
-- POST /admin/catalog/crop-profiles/{id}/approve.

INSERT INTO crops (code, name, salinity_ecw_threshold, sort_order) VALUES
    ('tomato',     '{"uz":"Pomidor","uz_cyr":"Помидор","ru":"Томат","en":"Tomato"}', 1.7, 1),
    ('cucumber',   '{"uz":"Bodring","uz_cyr":"Бодринг","ru":"Огурец","en":"Cucumber"}', 1.7, 2),
    ('pepper',     '{"uz":"Bulg‘or qalampiri","uz_cyr":"Булғор қалампири","ru":"Сладкий перец","en":"Sweet pepper"}', 1.0, 3),
    ('strawberry', '{"uz":"Qulupnay","uz_cyr":"Қулупнай","ru":"Клубника","en":"Strawberry"}', 0.7, 4),
    ('lettuce',    '{"uz":"Salat bargi","uz_cyr":"Салат барги","ru":"Салат","en":"Lettuce"}', 0.9, 5)
ON CONFLICT (code) DO NOTHING;

INSERT INTO crop_profiles (
    crop_id, stage, title, status, source,
    nh4_n, no3_n, p, k, ca, mg, s, cl,
    fe, mn, b, zn, cu, mo,
    ec_target, ph_min, ph_max, created_at, updated_at
)
SELECT c.id, st.stage, v.title, 1, v.source,
       v.nh4_n, v.no3_n, v.p, v.k, v.ca, v.mg, v.s, 0,
       v.fe, v.mn, v.b, v.zn, v.cu, v.mo,
       v.ec_target, 5.5, 6.5, 1763251200, 1763251200
FROM (VALUES
    -- NH4 1.2, K 9.5, Ca 5.4, Mg 2.4, NO3 16.0, SO4 4.4, H2PO4 1.5 mmol/L;
    -- Fe 15, Mn 10, Zn 5, B 30, Cu 0.75, Mo 0.5 µmol/L.
    ('tomato', 'Standart eritma (mineral vata)', 'Sonneveld & Voogt 2009; Sonneveld & Straver 1994',
     16.808, 224.107, 46.461, 371.434, 216.421, 58.332, 141.086,
     0.8377, 0.5494, 0.3243, 0.3269, 0.0477, 0.0480, 2.6),
    -- NH4 1.25, K 8.0, Ca 4.0, Mg 1.375, NO3 16.0, SO4 1.375, H2PO4 1.25 mmol/L;
    -- Fe 15, Mn 10, Zn 5, B 25, Cu 0.75, Mo 0.5 µmol/L.
    ('cucumber', 'Standart eritma (mineral vata)', 'Sonneveld & Voogt 2009; Sonneveld & Straver 1994',
     17.508, 224.107, 38.717, 312.786, 160.312, 33.419, 44.089,
     0.8377, 0.5494, 0.2702, 0.3269, 0.0477, 0.0480, 2.2)
) AS v(crop, title, source, nh4_n, no3_n, p, k, ca, mg, s, fe, mn, b, zn, cu, mo, ec_target)
JOIN crops c ON c.code = v.crop
CROSS JOIN (VALUES (2::smallint), (3::smallint)) AS st(stage)
WHERE NOT EXISTS (
    SELECT 1 FROM crop_profiles p
    WHERE p.crop_id = c.id AND p.stage = st.stage AND p.owner_user_id IS NULL
);
