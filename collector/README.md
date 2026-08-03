# Collector

This directory contains the SourceMod collector. Source files are split into a
single plugin entry point under `src/` and compile-time modules under `include/`.
The build still produces one `l4d2_player_stats.smx` file.

Version 0.6.5 maintains separate absolute snapshots for human Versus survivor and
infected Segments. Survivor snapshots add seven fixed infected-class rows, Witch
combat, official throwables and ammo packs, self tongue cuts, destroyed Tank
rocks, and Witch techniques while retaining survival, rescue, healing,
temporary-health, and friendly-fire metrics. Infected snapshots record valid
non-ghost spawns plus damage, incapacitations, kills, pin controls, Boomer bile
victims, and Spitter acid damage split by human/Bot survivor targets. All event
paths update bounded memory only; snapshots share the asynchronous flush
transaction and never enter the PvE statistics tables.

Versus round and Run results are stored separately from player Segments. The
collector reads L4D2 GameRules map and campaign scores, preserves raw winner
events for diagnostics, and only derives a winner for a normally completed Run.
It does not require Left4DHooks or recompute score formulas.
