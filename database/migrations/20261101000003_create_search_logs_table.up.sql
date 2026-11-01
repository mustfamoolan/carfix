CREATE TABLE IF NOT EXISTS search_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    vin VARCHAR(17) NOT NULL,
    vehicle_title VARCHAR(255),
    result_status VARCHAR(50),
    fetch_source VARCHAR(50),
    duration_ms BIGINT DEFAULT 0,
    ip_address VARCHAR(50),
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_search_logs_user_id ON search_logs (user_id);
CREATE INDEX IF NOT EXISTS idx_search_logs_vin ON search_logs (vin);

