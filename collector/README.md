# Collector

This directory contains the SourceMod collector. Source files are split into a
single plugin entry point under `src/` and compile-time modules under `include/`.
The build still produces one `l4d2_player_stats.smx` file.

Version 0.5.1 adds fixed-ID equipment rows, per-class special infected results,
control/rescue durations, Boss participation, skill actions, throwable uses,
and ammo-upgrade pack deployments. Unknown/custom firearms share one bounded
`Other Firearm` row; custom melee and throwables are ignored. Statistics remain
absolute snapshots owned by human survivor Segments; Versus gameplay statistics
remain deferred.
