# Collector

This directory contains the SourceMod collector. Source files are split into a
single plugin entry point under `src/` and compile-time modules under `include/`.
The build still produces one `l4d2_player_stats.smx` file.

Version 0.5.2 adds successful objective interactions, ammo-pile uses,
incapacitated/ledge-hanging duration, and medkit restores of black-and-white
teammates. Objective entities are allowlisted and counted at most once per
Round; duration tracking uses event boundaries and periodic snapshot
accumulation rather than per-second timers. Statistics remain absolute
snapshots owned by human survivor Segments; Versus gameplay statistics remain
deferred.
