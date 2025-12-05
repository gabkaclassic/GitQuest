DROP TABLE IF EXISTS events;
DROP INDEX IF EXISTS idx_events_timestamp_brin;

DROP TABLE IF EXISTS achievements;
DROP INDEX IF EXISTS idx_achievements_period;
DROP INDEX IF EXISTS idx_achievements_user_rule;

DROP TABLE IF EXISTS users;
DROP INDEX IF EXISTS idx_users_email;

DROP TABLE IF EXISTS rules;
DROP INDEX IF EXISTS idx_rules_version;
DROP INDEX IF EXISTS idx_rules_name;
DROP INDEX IF EXISTS idx_rules_name_version;
