# Collector

This directory contains the SourceMod collector. Source files are split into a
single plugin entry point under `src/` and compile-time modules under `include/`.
The build still produces one `l4d2_player_stats.smx` file.

Version 0.5 adds `coop` and `realism` medkit healing, temporary-health item use,
chapter participation, completion state, and campaign completion on top of the
v0.4 combat statistics. Statistics are stored as absolute snapshots owned by
human survivor Segments; Versus gameplay statistics remain deferred.
