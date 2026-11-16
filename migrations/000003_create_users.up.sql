-- binary/000003 + 000009 birlashtirilgan, dehqon uchun moslangan.
--
-- mentor_id olib tashlandi (bu yerda kerak emas).
-- role ma'nosi o'zgardi: 1=dehqon, 2=agronom (profil tasdiqlaydi,
-- hamjamiyatda "Agronom koʻrdi" belgisi), 3=admin.
CREATE TABLE IF NOT EXISTS users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    full_name VARCHAR(255),
    avatar_url VARCHAR(512),
    role SMALLINT NOT NULL DEFAULT 1,
    locale VARCHAR(8) NOT NULL DEFAULT 'uz',
    region_id INT REFERENCES regions(id),
    district_id INT REFERENCES districts(id),
    onboarded_at BIGINT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    CONSTRAINT users_locale_chk CHECK (locale IN ('uz', 'uz_cyr', 'ru'))
);

COMMENT ON COLUMN users.role IS '1=dehqon, 2=agronom, 3=admin';
COMMENT ON COLUMN users.onboarded_at IS 'Birinchi sozlash ekrani tugatilgan vaqt; NULL = hali sozlanmagan';

CREATE INDEX IF NOT EXISTS idx_users_region_id ON users(region_id);
CREATE INDEX IF NOT EXISTS idx_users_district_id ON users(district_id);
-- Agronomlar ro'yxati uchun: qatorlarning aksariyati role=1, shuning uchun partial.
CREATE INDEX IF NOT EXISTS idx_users_staff ON users(role) WHERE role > 1;
