# Collector

This directory contains the SourceMod collector. Source files are split into a
single plugin entry point under `src/` and compile-time modules under `include/`.
The build still produces one `l4d2_player_stats.smx` file.

Version 0.6.3 maintains separate absolute snapshots for human Versus survivor and
infected Segments. Survivor snapshots split human/Bot special infected and Tank
combat while retaining survival, rescue, healing, temporary-health, and friendly
fire metrics. Infected snapshots record valid non-ghost spawns plus damage,
incapacitations, and kills split by human/Bot survivor targets, seven fixed class
rows, four pin controls, Boomer bile victims, and Spitter acid damage. These
snapshots share the existing bounded closure queues and asynchronous flush
transaction but never enter the PvE statistics tables.
