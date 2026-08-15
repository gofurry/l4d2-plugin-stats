import type { EChartsCoreOption } from 'echarts/core'

export const playerStorageKey = 'l4d2-stats.player.steam-id.v1'
export const validSteamID = (value: string) => /^7656119\d{10}$/.test(value)
export const preferredPlayerSteamID = (routeSteamID: string, authenticatedSteamID: string, savedSteamID: string) => {
  if (validSteamID(routeSteamID)) return routeSteamID
  if (validSteamID(authenticatedSteamID)) return authenticatedSteamID
  return validSteamID(savedSteamID) ? savedSteamID : ''
}
export const numberFormat = new Intl.NumberFormat()
export const hours = (seconds: number) => `${(seconds / 3600).toFixed(seconds >= 36000 ? 0 : 1)} h`
export const duration = (seconds: number) => seconds >= 3600 ? hours(seconds) : `${Math.round(seconds / 60)} min`
export const date = (unix?: number) => unix ? new Date(unix * 1000).toLocaleString() : '—'

export const infectedNames = ['Smoker', 'Boomer', 'Hunter', 'Spitter', 'Jockey', 'Charger', 'Unknown', 'Tank']
export const equipmentNames = [
  '', 'Other Firearm', 'Pistol', 'Dual Pistols', 'Magnum', 'Uzi', 'Silenced SMG', 'MP5',
  'Pump Shotgun', 'Chrome Shotgun', 'Auto Shotgun', 'SPAS', 'M16', 'AK-47', 'SCAR', 'SG552',
  'Hunting Rifle', 'Military Sniper', 'Scout', 'AWP', 'Grenade Launcher', 'M60', 'Chainsaw',
  'Mounted Gun', 'Minigun', 'Baseball Bat', 'Cricket Bat', 'Crowbar', 'Electric Guitar', 'Fire Axe',
  'Frying Pan', 'Golf Club', 'Katana', 'Knife', 'Machete', 'Pitchfork', 'Shovel', 'Tonfa',
  'Molotov', 'Pipe Bomb', 'Vomit Jar',
]

export const palette = ['#c66f3b', '#899a67', '#d3a65f', '#9a7564', '#b65d50', '#6f8e90']
export const chartBase: EChartsCoreOption = {
  animationDuration: 450,
  textStyle: { color: '#5b473c', fontFamily: 'Inter, Segoe UI, Microsoft YaHei, sans-serif' },
  tooltip: { trigger: 'axis', backgroundColor: 'rgba(67,48,38,.94)', borderWidth: 0, textStyle: { color: '#fff7ed' } },
  grid: { left: 42, right: 18, top: 36, bottom: 34, containLabel: true },
}

export type PlayerCopy = Record<string, string>
