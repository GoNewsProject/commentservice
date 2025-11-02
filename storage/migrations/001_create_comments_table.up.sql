CREATE TABLE IF NOT EXISTS comments(
    id SERIAL PRIMARY KEY,
    news_id SERIAL,
    content TEXT(2000) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- TODO: CREATE INDEX

-- CREATE INDEX IF NOT EXISTS idx_news_published_at ON news(published_at DESC);
-- CREATE INDEX IF NOT EXISTS idx_news_source ON news(source);
-- CREATE INDEX IF NOT EXISTS idx_news_author ON news(author);
-- CREATE INDEX IF NOT EXISTS idx_news_link ON news(link);