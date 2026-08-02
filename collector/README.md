# Collector

This directory contains the SourceMod collector. Source files are split into a
single plugin entry point under `src/` and compile-time modules under `include/`.
The build still produces one `l4d2_player_stats.smx` file.

Version 0.4 adds `coop` and `realism` kill, effective damage, damage taken,
friendly fire, incapacitation, death, and rescue statistics on top of the v0.3
PvE/Versus lifecycle state machines. Statistics are stored as absolute snapshots
owned by human survivor Segments; Versus gameplay statistics remain deferred.
