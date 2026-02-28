-- Seed the first admin user. Idempotent: no-op if the email already exists.
-- Run via: psql ... -v email="..." -v passhash="..." -f seed_admin.sql
-- Or use scripts/seed_admin.sh which reads FIRST_ADMIN_EMAIL and FIRST_ADMIN_PASSWORD from env.
INSERT INTO users (email, role, password_hash)
VALUES (:'email', 'admin', :'passhash')
ON CONFLICT (email) DO NOTHING;
