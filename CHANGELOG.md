# Changelog

All notable changes to this project are documented in this file.

## Unreleased

## 1.3.4 - 2026-08-17

### Added

- Add a dedicated server-rendered L4D2 in-game portal for the native MOTD browser, including current-server status, online players, compact public player profiles, Achievement badges, lifetime highlights, rankings, announcements, and server documents.
- Add administrator-managed in-game defaults and per-server-group overrides for title, description, HTTP/HTTPS banner, external full-site link, homepage modules, highlight metrics, and bounded cache presets.
- Add per-server-group inherit, override, and hide behavior for introduction, command, and resource documents.
- Add a no-JavaScript legacy presentation and a dedicated PNG Achievement atlas generated from the same artwork sources as the modern WebP atlas.
- Add MOTD deployment guidance and generated `motd.txt` HTML redirection to `/ingame?server=<server_key>`.
- Add a global portal Background URL plus per-server-group inherit, override, and hidden modes without server-side fetching or proxying.
- Add independent server-introduction and per-instance status module switches, validated per-instance Steam join links, and group-wide status/player summaries.

### Changed

- Upgrade Collector and Dashboard package versions to `1.3.4` and Dashboard schema from 16 to 18 while keeping Stats schema 6, `stats_version=1`, and all gameplay/statistical contracts unchanged.
- Treat `server_key` as a logical server-group identity: multiple configured IP:PORT instances may intentionally share one key, portal configuration, document set, selection entry, and ranking scope.
- Reuse persisted A2S snapshots and existing Player, Achievement, Relationship, and Ranking services through bounded server-side views rather than browser-side API fan-out.
- Cache bounded in-game view models with approved TTL presets, request coalescing, stale-value fallback, and targeted invalidation after settings, content, server, and profile-visibility changes.
- Redesign the legacy-safe portal with a normal-flow proportional Home Banner, centered introduction panel, fixed cover background layer, compact group summary header, per-instance status table, denser Player and Ranking views, translucent panels, and old-WebKit scrollbar styling.
- Fingerprint embedded CSS and Achievement atlas URLs from their content while retaining immutable asset caching and no-cache HTML responses.

### Security

- Restrict in-game Banner and full-site destinations to credential-free absolute HTTP/HTTPS URLs, never server-fetch configured external URLs, and render all `/ingame` player data using anonymous public profile visibility.
- Sanitize the supported Markdown subset, reject executable or embedded content, and keep all in-game pages free of client-side scripts and API calls.
- Route validated HTTP/HTTPS links from server Markdown through the controlled Steam external-browser helper instead of navigating the native MOTD WebView.

### Fixed

- Complete Background columns automatically for pre-release Dashboard schema 17 databases before migrating group-scoped settings and documents to schema 18.
- Keep the last known valid `server_key` through instance outages, avoid copying group-wide Stats presence into each instance, and only attach SteamID64 when A2S and Stats display-name matching is unique.
- Keep the In-Game administrator form populated with approved highlight metrics and cache defaults while settings load or when an older partial response omits those values.

## 1.3.3 - 2026-08-15

### Added

- Add 12 public tiered Achievement series with 42 underlying tiers for throwables, objective interactions, temporary-health items, upgrade-pack deployment, and eight retention-safe PvE weapon families.
- Add the `weapon` Achievement category and 12 replaceable 256×256 placeholder artwork sources, expanding the deterministic sprite atlas from 26 to 38 tiles.
- Add frozen equipment-family mapping, cross-mode behavior aggregation, threshold boundaries, family disjointness, and Atlas coverage tests for the expanded Achievement Contract v1 catalog.

### Changed

- Upgrade the Collector and Dashboard to `1.3.3` while keeping Stats schema 6, Dashboard schema 16, `stats_version=1`, and all existing contract versions unchanged.
- Expand Achievement Contract v1 additively from 63 to 105 underlying items; 100 count toward normal completion and the existing 5 Secret items remain excluded.
- Evaluate weapon mastery from Dashboard lifetime `pve_equipment` aggregates so raw equipment retention cannot reduce progress, while PvE and Versus survivor behavior metrics continue to be resolved in bounded bulk queries.
- Upgrade Dashboard schema from 15 to 16 so an explicitly empty badge showcase remains distinct from a player who has never configured showcase slots.

### Fixed

- Keep an explicitly linked leaderboard player as the active profile instead of replacing it with the signed-in Steam identity.
- Allow players to remove their final showcased badge without the automatic default selection immediately restoring it.
- Align scoped PvE and Versus response types with the server payloads and contain tab rendering failures instead of blanking the entire player center.

## 1.3.2 - 2026-08-14

### Added

- Add Achievement Contract v1 with a frozen 63-item catalog, automated monotonic evaluation, permanent idempotent unlocks, on-demand profile evaluation, and resumable bounded historical backfill.
- Add public, mystery, and secret visibility semantics, completion and easter-egg counts, per-player unlock rates, live/history confirmation notices, and up to three authenticated badge showcase slots.
- Add a top-level player Achievement view, three profile-header badges, one main badge in player previews, and a read-only Achievement Engine status panel without manual refresh, rebuild, claim, or threshold controls.
- Add 26 validated 256×256 WebP badge sources and a deterministic build pipeline that produces a fixed 6×5, 128-pixel-tile WebP sprite atlas with generated TypeScript coordinates.
- Add nullable survivor fall-death facts for PvE and Versus, including schema 6 migrations for SQLite, MySQL, and PostgreSQL and matching deep-doctor invariants.
- Add authenticated per-player profile visibility settings at top-level tab granularity, defaulting public access to Overview, Analysis, and Player Relationships while keeping owner access complete.

### Changed

- Upgrade the Collector and Dashboard to `1.3.2`, Stats schema from 5 to 6, and Dashboard schema from 13 to 15; gameplay `stats_version=1`, Aggregate Contract v1, Incident Contract v1, Assist Contract v1, and Player Relationship Contract v1 remain unchanged.
- Extend the Steam identity session used by self-service badge settings while keeping badge selection restricted to the authenticated player's own unlocked achievements.
- Require a fresh Steam OpenID verification before badge showcase edits, grant only a 10-minute server-signed edit capability, and reject badge writes whose browser origin does not match the configured public origin.
- Build the achievement atlas before every frontend production build and embed the refreshed frontend in the Dashboard binary.
- Merge achievement category filters into the progress card, keep easter-egg discovery visible in the compact summary, render progress as full numeric values, and add named tooltips with better alignment for profile and preview badges.
- Enforce profile visibility on the server for summary, activity, PvE, Versus, analysis, relationships, history, achievements, and badges; shared PvE and Versus payloads are scoped to the requested visible tab.

### Fixed

- Preserve historical `NULL` for fall deaths so pre-v1.3.2 rows are never interpreted as zero, while new snapshots always persist zero or a positive count bounded by total deaths.
- Keep locked mystery achievements anonymous without leaking artwork, thresholds, progress, immutable keys, or accidental grouping, and omit locked secrets entirely.
- Keep local player-query preferences separate from Steam authorization so changing browser storage cannot grant badge editing access.

## 1.3.1 - 2026-08-14

### Added

- Add Player Relationship Contract v1 and permanent sparse per-Round facts for directed human-survivor revives, control rescues, medkit healing, black-and-white restores, and friendly-fire damage in PvE and Versus.
- Add Assist Contract v1 for PvE and Versus ordinary special infected, Tank, and Witch participation, preserving historical nullable fields as "not collected" and keeping kills mutually exclusive from assists.
- Extend Incident Contract v1 append-only with Witch startle, completed medkit heal, and allowlisted objective-completion events, including map-detail composition, timeline, Boss, and recent-event views.
- Add a dedicated player-relationship profile tab with range, server and mode filters, backend pagination and sorting, four factual summaries, and bidirectional detail drawers.
- Add cross-dialect schema 5 fixtures and deep-doctor checks for relationship references, versions, sparse-row invariants, nullable Assist totals, class totals, and Boss participation consistency.

### Changed

- Upgrade the Collector and Dashboard to `1.3.1` and Stats schema from 4 to 5; Dashboard schema remains 13, while gameplay `stats_version=1`, Aggregate Contract v1, and Incident Contract v1 remain unchanged.
- Keep the player preview grouped into PvE, co-play Top 3, and Versus sections; localize headshot kills and make the saved player card directly previewable from Player Center.
- Add common-infected kills per hour and firearm headshot rate to normalized PvE career metrics, and round battle-timeline values to one decimal place.
- Move the player-relationship mode selector into a tab-scoped toolbar filter, clarify player-card and recent-record wording, and limit analysis rescuers to Top 3.
- Show Assist data in PvE and Versus survivor profile views and explicitly render historical `NULL` values as not collected instead of zero.
- Treat permanent relationship facts independently from Incident retention and continue deriving co-play from overlapping same-Round, same-side player segments.

### Fixed

- Center recent-record load-more actions with stable spacing and keep the player-preview close control clear of identity content at narrow widths.
- Keep newly appended Incident types isolated from unknown future event IDs instead of guessing their meaning.

## 1.3.0 - 2026-08-14

### Added

- Add Stats schema 4 with permanent Round Context v1 facts and independently retained low-frequency Incident v1 records for controls, incapacitations, deaths, rescues, car alarms, Tank, and Witch lifecycle events.
- Add an isolated Analysis Flush pipeline with bounded queues, 256-Incident capture batches, idempotent writes, completeness accounting, and failure isolation from core cumulative statistics.
- Add the public `/analysis` experience with range, server, mode, and campaign filters; map samples, PvE completion semantics, normalized Incident timelines, Boss lifetime analysis, and stable rule-context fingerprints.
- Add player analysis for normalized PvE and Versus metrics, synchronized multi-control episodes, recent Incident detail, and Top-3 same-Round/same-side co-play summaries in player previews.
- Add derived leaderboards for rescue, incap, death, friendly-fire, Boss-participation, and per-spawn metrics with frozen sample gates and explicit higher/lower-is-better metadata.
- Add independent Incident retention settings, preview/confirmation, batched deletion, audit history, storage projection, admin status, and deep-doctor Context/Incident contract checks.
- Add Incident, Round Context, and derived-analysis v1 contract documents plus cross-dialect migration and query coverage.

### Changed

- Replace the loopback-only Steam OpenID proxy port with an optional full proxy address supporting HTTP, HTTPS, SOCKS5, and SOCKS5H; existing ports migrate automatically to `http://127.0.0.1:<port>`.
- Upgrade Dashboard schema from 10 to 11 to persist the Steam OpenID proxy URL while retaining the legacy port column for safe downgrade.
- Upgrade the Collector and Dashboard to `1.3.0`, Stats schema from 3 to 4, and Dashboard schema from 11 to 12; existing gameplay `stats_version=1` and Aggregate Contract v1 meanings remain unchanged.
- Upgrade Dashboard schema from 12 to 13 to persist the latest successful A2S server snapshot, serve cached status immediately, and refresh stale servers in the background.
- Keep core statistics authoritative when analysis capture is disabled, incomplete, dropped, or temporarily fails; analysis readers never interpret incomplete Incident rounds as zero-event samples.
- Paginate map and rule-context analysis on the server and expose validated sorting for every displayed analysis metric.

### Fixed

- Align battle-analysis terminology, spacing, date presentation, chart annotations, announcement previews, and leaderboard table surfaces with the rest of the Dashboard.
- Show explicit first-refresh progress when no A2S snapshot exists instead of making the home page appear stalled.

## 1.2.1 - 2026-08-11

### Added

- Add an optional loopback HTTP proxy port for Steam OpenID requests, configurable from the site administration page for hosts that cannot reach Steam directly.

### Changed

- Upgrade Dashboard schema from 9 to 10 to persist the optional Steam OpenID proxy port.

### Fixed

- Keep the player activity chart on daily buckets for all-time views instead of displaying monthly bucket start dates, limit the timeline to 30 daily points, and improve chart text contrast on dark backgrounds.
- Generate valid systemd path directives for installation directories containing spaces.
- Accept Steam's HTTP and HTTPS claimed identity forms during OpenID verification and report verification transport failures clearly.

## 1.2.0 - 2026-08-10

### Added

- Add a shared SQLite/MySQL/PostgreSQL database contract fixture and CI matrix using the three production migrations and equivalent domain-model assertions.
- Freeze the 11 aggregate kinds as Aggregate Contract v1 and persist the contract version across daily, monthly, lifetime, state, and retention records.
- Add `doctor --deep` read-only data-quality checks for schema and contract versions, lifecycle links, mode/side ownership, PvE totals, boot heartbeats, and aggregate freshness.
- Add verified ZIP backup and restore commands with online SQLite snapshots, SHA-256 manifests, integrity checks, safe extraction, rollback copies, and external-database guidance.
- Add redacted diagnostics export containing doctor reports, aggregate status, storage usage, server directory, configuration, and bounded recent logs.
- Track human-survivor car-alarm triggers separately for PvE and Versus, expose them on player pages, and add raw full-server Top rankings without changing Aggregate Contract v1.
- Track completed objective interactions for Versus survivors with the same allowlist and per-Round ownership rules as PvE, and expose the raw total on player pages without adding homepage, ranking, aggregate, or achievement behavior.
- Add the multi-server collector, shared Stats database, and Dashboard architecture diagram to the repository documentation.

### Changed

- Upgrade Dashboard schema from 8 to 9 and Stats schema from 1 to 3; `stats_version` remains at 1 because existing metric semantics are unchanged.
- Apply Collector Stats migrations sequentially so existing databases receive the nullable car-alarm and Versus-objective columns while historical rows remain distinguishable as not collected.
- Reject unknown aggregate contract versions during reads, writes, status checks, rebuilds, and retention cleanup validation.
- Include aggregate contract version and source watermark in retention previews and require both to match before deletion.
- Split the Dashboard frontend API, administration pages, player pages, shared player presentation code, and PvE collector statistics into smaller modules without changing public routes or statistics semantics.
- Refresh embedded frontend assets and align collector, Dashboard, frontend, build, and packaging versions at `1.2.0`.
- Align recent player Session and chapter records with stable date, duration, and status columns plus responsive narrow-screen behavior.
- Compile the collector with SourcePawn `1.12.0.7246`, reject mismatched local compilers, and order session declarations for SourceMod 1.12 compatibility.

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
