# Collector

This directory contains the SourceMod collector. Source files are split into a
single plugin entry point under `src/` and compile-time modules under `include/`.
The build still produces one `l4d2_player_stats.smx` file.

Version 0.2 adds authenticated human player identity and Session collection on
top of the v0.1 database foundation. Run, Round, Segment, and gameplay
statistics remain intentionally deferred until their lifecycle state machines
are implemented.
