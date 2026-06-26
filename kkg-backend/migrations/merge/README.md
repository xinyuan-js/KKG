# Backend Merge Database Plan

This directory contains non-destructive migration helpers for merging the Blog
and OJ backends into one backend.

Principles:

- Keep existing tables and data.
- Use `users` as the unified auth/user table.
- Keep OJ's legacy `user` table as migration source and rollback safety.
- Add mapping and shadow user columns before changing application code.
- Back up before every migration run.

Recommended order:

1. Run `scripts/backup_before_merge.sh`.
2. Apply `001_prepare_user_identity_map.sql`.
3. Verify row counts and samples with `002_verify_user_identity_map.sql`.
4. Switch application code to use unified user IDs.
5. Only after a stable release, consider dropping legacy columns/tables.

