# Collector

This directory contains the SourceMod collector. Source files are split into a
single plugin entry point under `src/` and compile-time modules under `include/`.
The build still produces one `l4d2_player_stats.smx` file.

Version 0.5.3 counts Vomit Jar actions from the owned projectile because L4D2
does not reliably expose those throws through `weapon_fire`. Molotov and Pipe
Bomb actions keep their existing event path, so the two paths cannot duplicate
a single action. Statistics remain absolute snapshots owned by human survivor
Segments; Versus gameplay statistics remain deferred.
