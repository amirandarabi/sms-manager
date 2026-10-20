# migrations

SQL migrations for PostgreSQL, applied by the goose runner in `pkg/db`. Each migration is a forward/down pair; destructive migrations (drop/rename column or table) require human sign-off per CLAUDE.md safety rules.
