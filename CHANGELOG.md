# Changelog

All notable changes to this project will be documented in this file.

## Unreleased

- Add separate absolute snapshot collectors for Versus survivor and infected Segments.
- Split Versus survivor special infected and Tank kills/damage by human or Bot victim.
- Record Versus survivor common kills, infected damage taken, friendly fire, survival, rescue, healing, and temporary health.
- Record valid human infected spawns plus damage, incapacitations, and kills split by human or Bot survivor victim.
- Add bounded closed Versus queues, shared transactional flush integration, status diagnostics, and a runtime enable ConVar.
- Extend SQLite integration and inspection tools with Versus table, side, mode, orphan, and snapshot checks.
- Count Vomit Jar actions from `vomitjar_projectile` ownership when `weapon_fire` does not expose the throw.
- Keep Vomit Jar projectile accounting separate from the Molotov/Pipe Bomb event path to prevent duplicate actions.
- Add allowlisted, success-only objective interaction counts with one attribution per entity and Round.
- Add ammo-pile refill counts, incapacitated seconds, and ledge-hanging seconds.
- Add successful medkit restores of black-and-white teammates.
- Accumulate active duration timers into periodic absolute snapshots without per-second polling.
- Add fixed-ID per-equipment PvE snapshots for official firearms, official melee weapons, and official throwables.
- Aggregate every unknown/custom firearm into one bounded `Other Firearm` row while ignoring custom melee and throwables.
- Add per-class special infected kills and effective damage.
- Add Smoker/Hunter/Jockey/Charger control counts, durations, and attributable teammate saves.
- Add self tongue cuts with official melee, destroyed Tank rocks, Witch one-shots and solo kills.
- Add Tank/Witch encounters and kill participation plus incendiary/explosive ammo-pack deployments.
- Keep firearm shots, hits, accuracy, ammunition, reloads, common-infected damage, melee hit metrics, decapitations, and laser sights out of scope.
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
