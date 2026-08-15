import { Alert, Button, Switch, message } from 'antd'
import { useMemo, useState } from 'react'
import type { PlayerProfileSection } from '../../api'
import { api } from '../../api'
import styles from './PlayerPage.module.scss'
import { defaultPlayerProfileSections, playerProfileSections } from './playerVisibility'

export function PlayerVisibilitySettings({ value, onSaved, zh }: { value: PlayerProfileSection[]; onSaved: (value: PlayerProfileSection[]) => void; zh: boolean }) {
  const [selected, setSelected] = useState<PlayerProfileSection[]>(value)
  const [saving, setSaving] = useState(false)
  const labels = useMemo(() => sectionLabels(zh), [zh])
  const dirty = playerProfileSections.some(section => selected.includes(section) !== value.includes(section))
  const toggle = (section: PlayerProfileSection, checked: boolean) => setSelected(current => checked ? [...current, section] : current.filter(item => item !== section))
  const save = async () => {
    setSaving(true)
    try {
      const result = await api.savePlayerProfileVisibility(selected)
      setSelected(result.visible_sections)
      onSaved(result.visible_sections)
      void message.success(zh ? '个人资料可见性已保存' : 'Profile visibility saved')
    } catch {
      void message.error(zh ? '保存失败，请确认 Steam 登录状态后重试' : 'Save failed. Check your Steam login and try again.')
    } finally {
      setSaving(false)
    }
  }
  return <div className={styles.visibilitySettings}>
    <div className={styles.visibilityHeading}>
      <div><h3>{zh ? '个人资料可见性' : 'Profile visibility'}</h3><p>{zh ? '选择其他玩家查询你的个人中心时可以看到的栏目。你自己始终可以查看全部内容。' : 'Choose which sections other players can see. You can always see every section.'}</p></div>
      <div><Button onClick={() => setSelected(defaultPlayerProfileSections)}>{zh ? '恢复默认' : 'Restore defaults'}</Button><Button type="primary" loading={saving} disabled={!dirty} onClick={() => void save()}>{zh ? '保存设置' : 'Save'}</Button></div>
    </div>
    {selected.length === 0 && <Alert type="info" showIcon title={zh ? '关闭全部栏目后，访客只能看到你的玩家名称和 SteamID。' : 'With every section hidden, visitors only see your player name and SteamID.'} />}
    <div className={styles.visibilityList}>
      {playerProfileSections.map(section => <div key={section}>
        <div><strong>{labels[section]}</strong>{defaultPlayerProfileSections.includes(section) && <small>{zh ? '默认公开' : 'Public by default'}</small>}</div>
        <Switch checked={selected.includes(section)} onChange={checked => toggle(section, checked)} aria-label={labels[section]} />
      </div>)}
    </div>
  </div>
}

function sectionLabels(zh: boolean): Record<PlayerProfileSection, string> {
  return zh ? {
    overview: '概览', achievements: '成就', analysis: '分析', pve: 'PvE', 'pve-details': 'PvE 明细',
    'versus-survivor': '对抗 · 幸存者', 'versus-survivor-details': '对抗幸存者明细',
    'versus-infected': '对抗 · 感染者', 'versus-infected-details': '对抗感染者明细', relationships: '玩家关系', history: '最近记录',
  } : {
    overview: 'Overview', achievements: 'Achievements', analysis: 'Analysis', pve: 'PvE', 'pve-details': 'PvE details',
    'versus-survivor': 'Versus · Survivor', 'versus-survivor-details': 'Versus survivor details',
    'versus-infected': 'Versus · Infected', 'versus-infected-details': 'Versus infected details', relationships: 'Player relationships', history: 'Recent records',
  }
}
