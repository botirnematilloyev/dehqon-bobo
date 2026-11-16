CREATE TABLE IF NOT EXISTS user_linked_accounts (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider SMALLINT NOT NULL, --M: 1=phone, 2=email, 3=telegram
    identifier VARCHAR(255) NOT NULL,-- phone, email, telegram_id
    is_verified BOOLEAN NOT NULL DEFAULT false,
    metadata JSONB,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    UNIQUE (provider, identifier)
);

CREATE INDEX IF NOT EXISTS idx_user_linked_accounts_user_id ON user_linked_accounts(user_id);
