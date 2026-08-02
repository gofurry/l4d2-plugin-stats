# Changelog

All notable changes to this project will be documented in this file.

## Unreleased

## 0.3.0

- Add PvE campaign Run continuity, chapter attempts, failure retries, transitions, and finale completion.
- Add separate Versus half Rounds with half and attempt numbering.
- Add human survivor/infected Segments that close on spectating, idle takeover, side changes, or Round end.
- Persist Run, Round, and Segment absolute snapshots in the shared asynchronous flush transaction.
- Add a bounded lifecycle closure queue, lifecycle diagnostics, SQLite checks, and a local validation checklist.
- Keep `boot_id` stable when SourceMod re-executes plugin configuration on map changes.

## 0.2.0

- Record SteamID64-backed human player identities in supported game modes.
- Add Session creation, cross-map continuity, disconnect and unsupported-mode closure.
- Track connected time separately from active play time, excluding spectators and idle takeover.
- Persist active and closed Session snapshots through the shared asynchronous flush transaction.
- Add a bounded in-memory closed Session queue for temporary database outages.
- Add privacy-safe Session diagnostics and SQLite inspection tooling.

## 0.1.0

- Establish the SourcePawn collector monorepo layout.
- Add portable schema migrations for SQLite, MySQL, and PostgreSQL.
- Add asynchronous database connection, migration, retry, heartbeat, and diagnostics foundations.
