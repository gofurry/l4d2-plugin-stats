# Atlas Build Spec v1

状态：v1.3.3 前端资源构建规范。

## 1. 目标

将 41 张：

```text
256×256 WebP source artwork
```

构建为：

```text
128×128 tile
+
单张 achievements-atlas.webp
+
编译期坐标表
```

线上 Achievement Badge 不产生 41 个独立图片请求。

## 2. 推荐目录

```text
dashboard/frontend/src/assets/achievements/
  artwork-manifest.json
  source/
    career-veteran.webp
    ...
  generated/
    achievements-atlas.webp
    achievement-atlas.generated.ts

dashboard/frontend/scripts/
  build-achievement-atlas.mjs
```

`generated/` 是否提交仓库由现有前端资产工作流决定，但 CI / release build 必须可复现生成。

## 3. Artwork Manifest

建议：

```json
[
  {
    "key": "career.veteran",
    "file": "career-veteran.webp",
    "tiered": true
  }
]
```

必须固定顺序。

不能依赖文件系统遍历顺序决定 atlas 坐标。

41 个 key 必须与 Achievement Catalog 一致。

## 4. Source 校验

构建脚本必须失败于：

- source 文件缺失；
- source 多余且未在 manifest；
- 重复 key；
- 重复 file；
- 非 256×256；
- 非 WebP；
- manifest key 无对应 Achievement；
- Achievement artwork key 无对应 manifest。

## 5. Resize

每张 source：

```text
256×256
→
128×128
```

保持：

- 透明背景；
- 正方形；
- 不裁切主体；
- 使用稳定高质量缩放算法。

推荐使用现有 Node 图像库，如项目接受新增构建依赖可用 `sharp`。

若不想新增长期 runtime 依赖，放在 devDependency。

## 6. Atlas 网格

41 张采用固定规则网格：

```text
6 columns × 7 rows
```

每 tile：

```text
128×128
```

基础 atlas 内容尺寸：

```text
768×896
```

允许在 tile 间加固定 gutter，例如 2px，以减少浏览器采样串色；若加 gutter，坐标生成必须统一。

优先简单 deterministic grid，不做复杂 bin-packing。

## 7. 输出

生成：

```text
achievements-atlas.webp
achievement-atlas.generated.ts
```

推荐把坐标编译进 TS，避免线上再请求一个 JSON。

示例：

```ts
export const achievementAtlas = {
  imageWidth: 768,
  imageHeight: 896,
  tileWidth: 128,
  tileHeight: 128,
  items: {
    "career.veteran": { x: 0, y: 0, w: 128, h: 128 },
    "career.survivor": { x: 128, y: 0, w: 128, h: 128 }
  }
} as const
```

线上网络请求：

> 只有 atlas WebP。

## 8. WebP 输出

建议：

- transparency；
- quality 约 80–85；
- 不使用过强有损压缩；
- 保证 32px/48px 下边缘清晰。

生成结果应带内容 hash 或跟随 Vite 资产 hash，允许：

```text
Cache-Control: public, max-age=31536000, immutable
```

## 9. `AchievementBadge` 组件

建议：

```tsx
<AchievementBadge
  artworkKey="boss.tank_hunter"
  tier={3}
  size={48}
/>
```

Props：

```ts
type AchievementBadgeProps = {
  artworkKey: AchievementArtworkKey
  tier?: 1 | 2 | 3 | 4
  size?: number
  locked?: boolean
  mystery?: boolean
  className?: string
}
```

职责：

- atlas 坐标；
- background-position；
- background-size；
- Tier shell；
- locked / mystery 状态；
- accessibility label。

## 10. 渲染方式

推荐一个固定方形内部节点：

```text
badge shell
  └─ atlas sprite
```

`background-image` 指向 atlas。

根据原始 atlas 尺寸与目标 size 按比例计算 background-size / background-position。

也可用一个裁切容器 + `<img>` 绝对定位，但 CSS background sprite 更简单。

## 11. Tier Shell

Tier 只由 CSS 表现：

```text
achievementBadge--tier1
achievementBadge--tier2
achievementBadge--tier3
achievementBadge--tier4
```

建议：

- Bronze；
- Silver；
- Gold；
- Diamond 蓝紫。

Tier UI 不生成额外图片请求。

## 12. Mystery

未解锁 `mystery`：

> 不使用真实 artwork。

显示统一 mystery placeholder。

不要因为 atlas 已包含真实图，就把真实 artwork 通过模糊/灰度方式泄露。

## 13. Secret

未解锁 Secret：

> API 不返回，因此组件不渲染。

已解锁后正常使用对应 atlas tile。

## 14. Showcase

Badge Showcase 与玩家 Preview 必须复用同一 `AchievementBadge` 组件，不复制渲染逻辑。

建议尺寸：

- Preview 主 Badge：18–22px
- Player Header：28–32px
- Showcase：40–48px
- Achievement Card：56–72px

## 15. 构建集成

推荐：

```json
{
  "scripts": {
    "build:achievement-atlas": "node scripts/build-achievement-atlas.mjs",
    "build": "pnpm build:achievement-atlas && tsc -b && vite build"
  }
}
```

CI 应在 frontend build 前执行 atlas build。

## 16. 测试

至少：

- manifest 41 key；
- 41 source 完整；
- atlas deterministic；
- 坐标不重叠；
- tile 全在 atlas 边界内；
- generated TS key 完整；
- `AchievementBadge` 正确计算坐标；
- Tier 1–4 CSS；
- Mystery 不暴露 artwork；
- Secret 未解锁不出现。

## 17. 版本与缓存

Artwork 内容变更时，Vite 生成新的 hashed atlas 文件名。

无需在业务 DB 保存 atlas version。

`achievement_key` / artwork key 稳定；UI 自动加载当前版本 atlas。

## 18. Codex 边界

Codex 负责：

- manifest；
- atlas build script；
- build dependency；
- generated coordinates；
- Badge component；
- Tier CSS；
- 页面接入；
- CI/build 校验。

Codex 不应自行“随便画成就素材”。

如果 source artwork 尚未提供，可以先：

- 使用明确占位资产；
- 完成整个 atlas pipeline；
- 让 source artwork 后续按同文件名替换。

不要改变 Achievement Catalog key 来适配图片。
