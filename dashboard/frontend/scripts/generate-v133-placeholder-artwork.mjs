import { access, mkdir } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import sharp from 'sharp'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const sourceRoot = path.join(root, 'src', 'assets', 'achievements', 'source')
const placeholders = [
  ['weapon-throwable-expert.webp', '#bf6b38', '<circle cx="105" cy="112" r="15"/><circle cx="151" cy="112" r="15"/><path d="m105 97 8-24m38 24-8-24M91 151h74"/>'],
  ['career-objective-master.webp', '#9a8142', '<circle cx="128" cy="119" r="28"/><path d="M128 76v-16m0 118v-16m43-43h16M69 119H53m105-30 12-12m-72 84-12 12m72-12 12 12M98 89 86 77"/>'],
  ['career-temp-health-addict.webp', '#5f8b65', '<path d="M92 82h72v30l-51 66H75v-30l51-66zM91 130h58"/>'],
  ['support-firepower-upgrade.webp', '#a25237', '<path d="M82 95h92v64H82zM105 76v19m46-19v19m-23 12v40m-20-20h40"/>'],
  ['weapon-single-shotgun.webp', '#8c6d50', '<path d="m69 151 115-58 8 16-115 58zm79-40 23 47M78 146l-18-20"/>'],
  ['weapon-chainsaw.webp', '#9c4337', '<path d="M67 113h101l22 18-22 18H67zM89 96v17m49-17v17M99 131h61"/>'],
  ['weapon-machine-gun.webp', '#666f76', '<path d="M57 109h119v30H57zm119 7h25v16h-25m-72 7-18 34m47-34 18 34M82 109 68 91"/>'],
  ['weapon-smg.webp', '#657b4f', '<path d="M58 105h118v32h-46l-13 36H91l8-36H58zm118 8h27v12h-27"/>'],
  ['weapon-bolt-sniper.webp', '#526f82', '<path d="M51 114h151v18H51zm58-15h53v15h-53m37 18-17 39h-23l10-39M74 114 62 93"/>'],
  ['weapon-heavy-primary.webp', '#765a4c', '<path d="M49 106h146v29H49zm111-13h35v13h-35m-51 29-17 39H69l10-39M66 106 54 87"/>'],
  ['weapon-grenade-launcher.webp', '#a56a34', '<path d="M55 102h137v36H55zm28 36-13 31h-23l11-31m85 0 15 31h23l-11-31M91 115h55"/>'],
  ['weapon-melee.webp', '#7c5142', '<path d="m87 176 64-99 18 12-64 99zM80 168l33 22m22-102 26 17"/>'],
]

await mkdir(sourceRoot, { recursive: true })
const force = process.argv.includes('--force')
for (const [file, accent, symbol] of placeholders) {
  const output = path.join(sourceRoot, file)
  if (!force) {
    try {
      await access(output)
      continue
    } catch {}
  }
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256" viewBox="0 0 256 256">
    <defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop stop-color="#23282b"/><stop offset="1" stop-color="#111416"/></linearGradient></defs>
    <path d="M128 13 224 50v68c0 61-38 103-96 125-58-22-96-64-96-125V50z" fill="url(#g)" stroke="${accent}" stroke-width="8"/>
    <path d="M128 35 202 64v53c0 47-27 80-74 100-47-20-74-53-74-100V64z" fill="none" stroke="#d6c49a" stroke-opacity=".55" stroke-width="3"/>
    <circle cx="128" cy="122" r="61" fill="${accent}" fill-opacity=".18" stroke="${accent}" stroke-width="4"/>
    <g fill="none" stroke="#eee4ce" stroke-width="9" stroke-linecap="round" stroke-linejoin="round">${symbol}</g>
  </svg>`
  await sharp(Buffer.from(svg)).webp({ quality: 90, alphaQuality: 100, effort: 6 }).toFile(output)
  console.log(`Generated placeholder ${file}`)
}
