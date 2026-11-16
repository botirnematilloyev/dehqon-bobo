CREATE TABLE IF NOT EXISTS districts (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name JSONB NOT NULL,
    remote_id VARCHAR UNIQUE,
    region_id INT NOT NULL REFERENCES regions(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_districts_region_id ON districts(region_id);
