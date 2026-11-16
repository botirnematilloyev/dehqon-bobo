-- binary/000005 + 000007 + 000008 birlashtirilgan.
CREATE TABLE IF NOT EXISTS user_sessions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) UNIQUE NOT NULL,
    -- Refresh rotatsiyasida eski hash shu yerga ko'chadi: qayta taqdim
    -- etilsa — token o'g'irlangan yoki takror, sessiya bekor qilinadi.
    previous_refresh_token_hash VARCHAR(255),
    fcm_token VARCHAR(255),
    device_id VARCHAR(255),
    device_name VARCHAR(255),
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    ip_address VARCHAR(255),
    platform SMALLINT,
    os VARCHAR(255),
    last_active_at BIGINT,
    expires_at BIGINT NOT NULL,
    revoked_at BIGINT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

COMMENT ON COLUMN user_sessions.platform IS '1=web, 2=ios, 3=android';

CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
-- Muddati o'tganlarni soatlik tozalash: WHERE expires_at < $1
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_user_sessions_previous_refresh_hash ON user_sessions(previous_refresh_token_hash);
