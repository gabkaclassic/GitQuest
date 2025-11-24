CREATE TABLE IF NOT EXISTS events (
    "id" varchar(64) primary key, 
    "actor" varchar(64) NOT NULL, 
    "timestamp" TIMESTAMP, 
    "type" varchar(32) NOT NULL
);
CREATE INDEX idx_events_timestamp_brin ON events USING brin ("timestamp");

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(64) UNIQUE NOT NULL,
    password VARCHAR(256) NOT NULL
);
CREATE INDEX idx_users_email ON users(email);