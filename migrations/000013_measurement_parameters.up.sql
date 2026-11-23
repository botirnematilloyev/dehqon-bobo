-- O'lchovni baholash chegaralari. Kodda son yo'q: hammasi shu yerda,
-- agronom standartini, dehqon esa o'zinikini o'zgartira oladi.
-- is_reviewed = FALSE: agronom tasdiqlashi kerak.
INSERT INTO calc_parameters (key, value, min_value, max_value, unit, description, source, farm_overridable, updated_at) VALUES
    ('ec_tolerance_pct', 10, 2, 50, '%',
     '{"uz":"O‘lchangan EC kutilgandan shuncha foiz farq qilsa tuzatish taklif qilinadi","ru":"Допустимое отклонение измеренной EC от расчётной"}',
     'amaliy chegara (tasdiq kutilmoqda)', TRUE, 1763251200),
    ('drain_ec_excess_max', 1.0, 0, 5, 'mS/cm',
     '{"uz":"Drenaj EC kiruvchi eritmadan shuncha ko‘p bo‘lsa — tuz to‘planmoqda","ru":"Максимальное превышение EC дренажа над EC подачи"}',
     'Sonneveld & Voogt 2009, mineral vata (tasdiq kutilmoqda)', TRUE, 1763251200),
    ('drain_pct_min', 20, 0, 100, '%',
     '{"uz":"Drenaj ulushi pastki chegarasi","ru":"Минимальная доля дренажа"}',
     'Sonneveld & Voogt 2009, mineral vata (tasdiq kutilmoqda)', TRUE, 1763251200),
    ('drain_pct_max', 50, 0, 100, '%',
     '{"uz":"Drenaj ulushi yuqori chegarasi","ru":"Максимальная доля дренажа"}',
     'Sonneveld & Voogt 2009, mineral vata (tasdiq kutilmoqda)', TRUE, 1763251200)
ON CONFLICT (key) DO NOTHING;
