# Collector

This directory contains the SourceMod collector. Source files are split into a
single plugin entry point under `src/` and compile-time modules under `include/`.
The build still produces one `l4d2_player_stats.smx` file.

The collector maintains separate absolute snapshots for human Versus survivor and
infected Segments. Survivor snapshots add seven fixed infected-class rows, Witch
combat, official throwables and ammo packs, self tongue cuts, destroyed Tank
rocks, and Witch techniques while retaining survival, rescue, healing,
temporary-health, and friendly-fire metrics. Infected snapshots record valid
non-ghost spawns plus damage, incapacitations, kills, pin controls, Boomer bile
victims, and Spitter acid damage split by human/Bot survivor targets. All event
paths update bounded memory only; snapshots share the asynchronous flush
transaction and never enter the PvE statistics tables.

Stats schema 7 adds five nullable absolute survivor fields to both PvE and
Versus survivor snapshots: engine-awarded teammate protections, ledge-entry
transitions, damaging Tank-rock impacts received, Hunter Skeets, and Charger
Levels. Historical `NULL` means the Segment predates collection; v1.3.5
snapshots always write zero or a positive value. The Skeet/Level detectors are
bounded repository-owned episode state machines: they latch engine state before
death, bridge event-order differences with a short lethal-damage candidate, and
reuse the existing official-melee classifier. They still require real
build-10097 validation as documented in
`docs/v1.3.5-technique-validation.md`; temporary transition diagnostics are
available through the default-off `sm_lps_technique_debug` cvar.

Chat Audit is a separate default-on data domain controlled by
`sm_lps_chat_audit_enabled 1`. Real-human `say` and `say_team` are admitted to a
bounded 1024-row memory queue independently of the gameplay mode whitelist,
then persisted in batches of at most 64 to the transient Stats outbox. The
outbox is pruned after 72 hours and never turns unsupported modes into gameplay
Sessions or Stats. Chat bodies are never written to SourceMod logs.

Versus round and Run results are stored separately from player Segments. The
collector reads L4D2 GameRules map and campaign scores, preserves raw winner
events for diagnostics, and only derives a winner for a normally completed Run.
It does not require Left4DHooks or recompute score formulas.

The Versus v1 database and reader boundary is frozen by
`contracts/versus-v1.md` and `contracts/versus-schema-v1.json`. Builds compare
that manifest with all three migrations before compiling the plugin.
