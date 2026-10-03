-- Indexes for the per-URL analytics and per-user listing queries, which
-- previously scanned the whole clicks/urls tables.
CREATE INDEX IF NOT EXISTS idx_clicks_url_id_created_at ON clicks (url_id, created_at);
CREATE INDEX IF NOT EXISTS idx_clicks_user_id ON clicks (user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_urls_user_id_created_at ON urls (user_id, created_at DESC);
