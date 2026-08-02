# Collector

This directory contains the SourceMod collector. Source files are split into a
single plugin entry point under `src/` and compile-time modules under `include/`.
The build still produces one `l4d2_player_stats.smx` file.

Version 0.1 only owns database connection, migrations, server registration,
heartbeats, retry control, and administrator diagnostics. Player collection is
introduced in later versions after the database foundation is stable.

