import { mkdir, readFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import sharp from 'sharp'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const assetRoot = path.join(root, 'src', 'assets', 'achievements')
const sourceRoot = path.join(assetRoot, 'source')
const manifest = JSON.parse(await readFile(path.join(assetRoot, 'artwork-manifest.json'), 'utf8'))
await mkdir(sourceRoot, { recursive: true })

for (const [index, item] of manifest.entries()) {
  const hue = (index * 47 + 18) % 360
  const svg = `<svg width="256" height="256" viewBox="0 0 256 256" xmlns="http://www.w3.org/2000/svg">
    <circle cx="128" cy="128" r="104" fill="#202427" fill-opacity="0.94" stroke="hsl(${hue} 54% 58%)" stroke-width="8" stroke-dasharray="8 10"/>
    <path d="M128 54 190 90v76l-62 36-62-36V90z" fill="hsl(${hue} 38% 34%)" fill-opacity="0.78" stroke="hsl(${hue} 62% 70%)" stroke-width="6"/>
    <circle cx="128" cy="128" r="34" fill="none" stroke="#f2e6d2" stroke-opacity="0.74" stroke-width="8"/>
    <path d="M104 128h48M128 104v48" stroke="#f2e6d2" stroke-opacity="0.52" stroke-width="6" stroke-linecap="round"/>
  </svg>`
  await sharp(Buffer.from(svg)).webp({ quality: 90, alphaQuality: 100, effort: 6 }).toFile(path.join(sourceRoot, item.file))
}

console.log(`Created ${manifest.length} explicit placeholder WebP sources.`)
