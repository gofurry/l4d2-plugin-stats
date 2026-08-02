# Collector

This directory contains the SourceMod collector. Source files are split into a
single plugin entry point under `src/` and compile-time modules under `include/`.
The build still produces one `l4d2_player_stats.smx` file.

Version 0.3 adds PvE/Versus Run and Round state machines plus human participation
Segments on top of authenticated identity and Session collection. Gameplay
statistics remain intentionally deferred; this version only establishes their
ownership boundaries.
