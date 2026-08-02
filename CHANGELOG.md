# Changelog

All notable changes to this project will be documented in this file.

## Unreleased

- Add successful medkit self/other use counts and actual permanent health restored.
- Add pills, adrenaline, and actual temporary health received statistics.
- Add per-Segment chapter participation, alive/dead completion, and campaign completion results.
- Capture temporary-health gain from a pre-command health snapshot without assuming default item ConVars.
- Extend SQLite validation and inspection output for all v0.5 PvE fields.
- Add `coop` and `realism` common, special, Tank, and Witch last-hit kill statistics.
- Record effective health loss dealt to special infected, Tanks, and Witches without overkill inflation.
- Record infected damage taken and split human-target, bot-target, and received friendly fire.
- Add incapacitation, death, incap revive, ledge rescue, defibrillator revive, and rescue-received statistics.
- Persist PvE Segment statistics as absolute snapshots in the shared asynchronous flush transaction.
- Extend SQLite integration and inspection tools with PvE statistics checks.

## 0.3.0

- Add PvE campaign Run continuity, chapter attempts, failure retries, transitions, and finale completion.
- Add separate Versus half Rounds with half and attempt numbering.
- Add human survivor/infected Segments that close on spectating, idle takeover, side changes, or Round end.
- Persist Run, Round, and Segment absolute snapshots in the shared asynchronous flush transaction.
- Add a bounded lifecycle closure queue, lifecycle diagnostics, SQLite checks, and a local validation checklist.
- Keep `boot_id` stable when SourceMod re-executes plugin configuration on map changes.
- Preserve one SteamID-backed Session across listen-server map reconnects, with a bounded 120-second transfer window.
- Abandon a Versus Run when the map actually ends after only one half instead of preserving a stale half continuation.
- Extend SQLite inspection and integration checks for corruption, orphan records, invalid times, and abandoned-startup recovery.

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
