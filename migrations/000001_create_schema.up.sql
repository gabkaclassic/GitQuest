CREATE TABLE IF NOT EXISTS events (
    "id" varchar(64) primary key, 
    "actor" varchar(64) NOT NULL, 
    "timestamp" TIMESTAMP, 
    "type" varchar(32) NOT NULL
);
CREATE INDEX idx_events_timestamp_brin ON events USING brin ("timestamp");

CREATE TABLE users (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "email" VARCHAR(64) UNIQUE NOT NULL,
    "password" VARCHAR(256) NOT NULL
);
CREATE INDEX idx_users_email ON users(email);

CREATE TABLE rules (
    "name" VARCHAR(256) NOT NULL,
    "version" VARCHAR(64) NOT NULL,
    "description" TEXT,
    "event_type" VARCHAR(100) NOT NULL,
    "window" BIGINT NOT NULL CHECK ("window" >= 0),
    "count" INTEGER NOT NULL CHECK ("count" >= 0),
    "condition" VARCHAR(3) NOT NULL CHECK ("condition" IN ('>', '>=', '==', '<', '<=')),
    "streak" BOOLEAN NOT NULL DEFAULT false,
    "reward" INTEGER NOT NULL,
    
    PRIMARY KEY ("name", "version")
);

CREATE INDEX idx_rules_version ON rules("version");
CREATE INDEX idx_rules_name ON rules("name");
CREATE INDEX idx_rules_name_version ON rules("name", "version");
ALTER TABLE rules ADD CONSTRAINT chk_window_max CHECK ("window" <= 2592000000000000);

CREATE TABLE achievements (
    "user" VARCHAR(64) NOT NULL,
    "rule_name" VARCHAR(256) NOT NULL,
    "rule_version" VARCHAR(64) NOT NULL,
    "reward" INTEGER NOT NULL CHECK ("reward" >= 0),
    "period_start" TIMESTAMP NOT NULL,
    "period_end" TIMESTAMP NOT NULL,
    
    CHECK ("period_end" >= "period_start"),
    
    FOREIGN KEY ("rule_name", "rule_version") REFERENCES rules("name", "version")
);

CREATE INDEX idx_achievements_period ON achievements("period_start", "period_end");
CREATE INDEX idx_achievements_user_rule ON achievements("user", "rule_name", "rule_version");