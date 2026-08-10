# Changelog

All notable changes to this project are documented in this file.

## 1.2.0 - 2026-08-10

### Added

- Add a shared SQLite/MySQL/PostgreSQL database contract fixture and CI matrix using the three production migrations and equivalent domain-model assertions.
- Freeze the 11 aggregate kinds as Aggregate Contract v1 and persist the contract version across daily, monthly, lifetime, state, and retention records.
- Add `doctor --deep` read-only data-quality checks for schema and contract versions, lifecycle links, mode/side ownership, PvE totals, boot heartbeats, and aggregate freshness.
- Add verified ZIP backup and restore commands with online SQLite snapshots, SHA-256 manifests, integrity checks, safe extraction, rollback copies, and external-database guidance.
- Add redacted diagnostics export containing doctor reports, aggregate status, storage usage, server directory, configuration, and bounded recent logs.

### Changed

- Upgrade Dashboard schema from 8 to 9; Stats schema and `stats_version` remain at 1.
- Reject unknown aggregate contract versions during reads, writes, status checks, rebuilds, and retention cleanup validation.
- Include aggregate contract version and source watermark in retention previews and require both to match before deletion.
- Split the Dashboard frontend API, administration pages, player pages, shared player presentation code, and PvE collector statistics into smaller modules without changing public routes or statistics semantics.
- Refresh embedded frontend assets and align collector, Dashboard, frontend, build, and packaging versions at `1.2.0`.

## 1.1.0 - 2026-08-08

### Added

- Add an online-player preview card to the homepage A2S player list, backed by aggregated SteamID statistics.
- Expose each collector's stable `server_key` through A2S rules so Dashboard servers can be matched to their Stats DB source.
- Add offline Chinese installation and upgrade guides covering collector setup, all three database drivers, Dashboard initialization, systemd, Nginx, verification, backup, and rollback.
- Add ready-to-copy Dashboard examples for SQLite, MySQL, and PostgreSQL plus a SourceMod database and Nginx example.

### Changed

- Use release-ready `1.1.0` versions and neutral production defaults throughout the collector and Dashboard.
- Default Dashboard examples to `0.0.0.0:18848` so first-time administrators can initialize the site from another computer using the server IP.
- Package Dashboard configuration beside each Windows/Linux binary and keep development-only `docs/` and `contracts/` out of release archives.
- Clarify the distinction between normal read-only Stats DB access and the write permission required by optional raw-data cleanup.

## 1.0.0 - 2026-08-05

### Collector

- Record SteamID64-backed identities, Sessions, Runs, Rounds, Segments, chapter results, connection time, and active play time for `coop`, `realism`, and `versus`.
- Collect bounded PvE and Versus combat, survival, rescue, healing, equipment, objective, technique, class, score, half, and match-result statistics without recording Bot player profiles.
- Support SQLite, MySQL, and PostgreSQL through equivalent schema migrations and an asynchronous snapshot-based persistence pipeline.
- Add retry, bounded queues, periodic and lifecycle flushes, migration checks, diagnostics, and privacy-safe status commands.

### Dashboard

- Add a Go/Fiber single-binary service with an embedded React interface, strict YAML configuration, rotating logs, Cobra commands, and optional systemd installation.
- Add Steam OpenID/manual SteamID lookup, personal statistics, PvE/Versus leaderboards, multi-server A2S status, announcements, Markdown-managed server information, themes, SEO settings, and a protected runtime monitor.
- Add a single-administrator JWT/CSRF-protected management interface for site, server, security, announcement, and data-operation settings.
- Add bounded public caches, request coalescing, site-wide rate limiting, incremental daily aggregation, monthly/lifetime rollups, growth monitoring, and guarded batched raw-data cleanup.

### Release engineering

- Add Go and React CI workflows, reproducible SourcePawn/Go/frontend builds, cross-platform release packaging, migration validation, database inspection tooling, and an MIT license.

## 0.4.0–0.6.6

- Expand the collector from lifecycle records into the complete PvE and Versus v1 statistics contract.
- Add equipment and infected-class breakdowns, special techniques, objective interactions, duration metrics, campaign/half scores, result snapshots, and three-database validation.

## 0.3.0

- Add campaign Run, Round, Segment, map-transition, retry, finale, side-change, idle, spectator, and abnormal-restart lifecycle handling.

## 0.2.0

- Add SteamID64 player identity, Session continuity, connected time, active play time, and bounded asynchronous persistence.

## 0.1.0

- Establish the SourcePawn monorepo, portable SQLite/MySQL/PostgreSQL migrations, connection management, migration execution, heartbeat, retry, and diagnostics foundations.
