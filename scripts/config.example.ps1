$SourceModRoot = "E:\SteamLibrary\steamapps\common\Left 4 Dead 2\left4dead2\addons\sourcemod"
$CompilerSourceModRoot = "E:\tools\sourcemod-1.12.0-git7246-windows\addons\sourcemod"
$ExpectedSourcePawnVersion = "1.12.0.7246"

$CompilerPath = Join-Path $CompilerSourceModRoot "scripting\spcomp.exe"
$SourceModInclude = Join-Path $CompilerSourceModRoot "scripting\include"
$PluginDirectory = Join-Path $SourceModRoot "plugins"
$RuntimeConfigDirectory = Join-Path $SourceModRoot "configs\l4d2_player_stats"
