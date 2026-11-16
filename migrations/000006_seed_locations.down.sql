-- Faqat shu migratsiya kiritgan qatorlarni o'chiradi.
--
-- Agar biror foydalanuvchi region/district ga bog'langan bo'lsa,
-- users.region_id / users.district_id FK'lari o'chirishni to'sadi.
-- Bu ATAYLAB shunday: ma'lumotnomani jimgina yo'q qilib, foydalanuvchi
-- profilini buzib qo'ymaslik kerak.
DELETE FROM districts WHERE remote_id IS NOT NULL;
DELETE FROM regions WHERE remote_id IS NOT NULL;
