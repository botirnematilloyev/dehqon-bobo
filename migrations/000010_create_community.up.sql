-- Hamjamiyat: natija / muammo / savol postlari.

CREATE TABLE IF NOT EXISTS posts (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type SMALLINT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    trust_level SMALLINT NOT NULL DEFAULT 1,
    title VARCHAR(255) NOT NULL,
    body TEXT,
    crop_id INT REFERENCES crops(id) ON DELETE SET NULL,
    season_id BIGINT REFERENCES seasons(id) ON DELETE SET NULL,
    -- "Retseptni olish" tugmasi shu profilni nusxalaydi.
    crop_profile_id BIGINT REFERENCES crop_profiles(id) ON DELETE SET NULL,
    -- Post yozilgan paytdagi sharoit (keyin profil o'zgarsa ham filtr to'g'ri ishlaydi).
    region_id INT REFERENCES regions(id),
    water_source SMALLINT,
    -- Kartochkadagi raqamlar: {"yield_kg_m2": 17, "avg_ec": 2.3, "measurements": 46}
    data_snapshot JSONB,
    useful_count INT NOT NULL DEFAULT 0,
    comment_count INT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

COMMENT ON COLUMN posts.type IS '1=natija, 2=muammo, 3=savol';
COMMENT ON COLUMN posts.status IS '1=published, 2=hidden (moderatsiya), 3=deleted';
COMMENT ON COLUMN posts.trust_level IS '1=shaxsiy fikr, 2=maʼlumot bilan (jurnal), 3=agronom koʻrdi';

-- Lenta: yangilari tepada, faqat chop etilganlar.
CREATE INDEX IF NOT EXISTS idx_posts_feed ON posts(created_at DESC) WHERE status = 1;
CREATE INDEX IF NOT EXISTS idx_posts_type_feed ON posts(type, created_at DESC) WHERE status = 1;
-- "Mening sharoitim" filtri.
CREATE INDEX IF NOT EXISTS idx_posts_condition ON posts(region_id, water_source, created_at DESC) WHERE status = 1;
CREATE INDEX IF NOT EXISTS idx_posts_author_id ON posts(author_id);

CREATE TABLE IF NOT EXISTS post_media (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    kind SMALLINT NOT NULL,
    url VARCHAR(1024) NOT NULL,
    duration_sec INT,
    width INT,
    height INT,
    position SMALLINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL
);

COMMENT ON COLUMN post_media.kind IS '1=rasm, 2=ovozli xabar';

CREATE INDEX IF NOT EXISTS idx_post_media_post_id ON post_media(post_id);

CREATE TABLE IF NOT EXISTS post_symptoms (
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    symptom_id INT NOT NULL REFERENCES symptoms(id) ON DELETE RESTRICT,
    PRIMARY KEY (post_id, symptom_id)
);

CREATE INDEX IF NOT EXISTS idx_post_symptoms_symptom_id ON post_symptoms(symptom_id);

CREATE TABLE IF NOT EXISTS post_reactions (
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind SMALLINT NOT NULL DEFAULT 1,
    created_at BIGINT NOT NULL,
    PRIMARY KEY (post_id, user_id, kind)
);

COMMENT ON COLUMN post_reactions.kind IS '1=foydali';

CREATE INDEX IF NOT EXISTS idx_post_reactions_user_id ON post_reactions(user_id);

CREATE TABLE IF NOT EXISTS post_comments (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id BIGINT REFERENCES post_comments(id) ON DELETE CASCADE,
    body TEXT,
    voice_url VARCHAR(1024),
    is_accepted BOOLEAN NOT NULL DEFAULT FALSE,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    CONSTRAINT post_comments_content_chk CHECK (body IS NOT NULL OR voice_url IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_post_comments_post_created ON post_comments(post_id, created_at);
-- Bitta postda bitta tasdiqlangan javob.
CREATE UNIQUE INDEX IF NOT EXISTS uq_post_comments_accepted ON post_comments(post_id) WHERE is_accepted;

CREATE TABLE IF NOT EXISTS post_reports (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    comment_id BIGINT REFERENCES post_comments(id) ON DELETE CASCADE,
    reporter_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason SMALLINT NOT NULL,
    note TEXT,
    status SMALLINT NOT NULL DEFAULT 1,
    resolved_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    resolved_at BIGINT,
    created_at BIGINT NOT NULL
);

COMMENT ON COLUMN post_reports.reason IS '1=xavfli maslahat, 2=yashirin reklama, 3=haqorat, 4=boshqa';
COMMENT ON COLUMN post_reports.status IS '1=yangi, 2=koʻrib chiqildi, 3=rad etildi';

CREATE INDEX IF NOT EXISTS idx_post_reports_open ON post_reports(created_at) WHERE status = 1;
