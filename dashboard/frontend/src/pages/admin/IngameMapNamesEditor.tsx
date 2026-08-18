import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Input, Spin, Typography, message } from 'antd'
import { useRef, useState } from 'react'
import { api, type IngameMapName } from '../../api'
import styles from './AdminIngamePage.module.scss'
import { useDebouncedAutosave } from './useDebouncedAutosave'

type MapNameDraft = IngameMapName & { draftKey: string }

export function IngameMapNamesEditor({ zh }: { zh: boolean }) {
  const queryKey = ['admin-ingame-map-names']
  const query = useQuery({ queryKey, queryFn: api.ingameMapNames })
  if (query.isLoading) return <Spin />
  if (query.isError) return <Alert type="error" showIcon title={query.error.message} />
  const values = query.data ?? []
  return <MapNamesDraftEditor initialValues={values} zh={zh} queryKey={queryKey} />
}

function MapNamesDraftEditor({ initialValues, zh, queryKey }: { initialValues: IngameMapName[]; zh: boolean; queryKey: string[] }) {
  const label = (cn: string, en: string) => zh ? cn : en
  const client = useQueryClient()
  const nextDraftID = useRef(initialValues.length)
  const [drafts, setDrafts] = useState<MapNameDraft[]>(() => initialValues.map((item, index) => ({ ...item, draftKey: `saved-${index}` })))
  const autosave = useDebouncedAutosave({
    save: api.saveIngameMapNames,
    onSaved: values => {
      client.setQueryData(queryKey, values)
    },
    onError: error => void message.error(error.message),
  })
  const normalized = (values: MapNameDraft[]) => {
    const result = values.map(item => ({ map_name: item.map_name.trim().toLowerCase(), display_name: item.display_name.trim(), updated_at: item.updated_at }))
      .sort((left, right) => left.map_name.localeCompare(right.map_name))
    const names = new Set(result.map(item => item.map_name))
    return result.some(item => !item.map_name || [...item.map_name].length > 128 || !item.display_name || [...item.display_name].length > 80) || names.size !== result.length ? undefined : result
  }
  const replace = (next: MapNameDraft[], immediate = false) => {
    setDrafts(next)
    const values = normalized(next)
    if (values) autosave.schedule(values, immediate)
  }
  return <div className={styles.mapNameEditor}>
    <Typography.Paragraph type="secondary">{label('这里只保存三方图名称或对官方名称的覆盖；填写完整且无重复后自动保存。', 'Only custom maps and official-name overrides are stored here; complete, unique rows are saved automatically.')}</Typography.Paragraph>
    <div className={styles.mapNameToolbar}><Typography.Text>{autosave.saving ? label('自动保存中…', 'Saving…') : label(`已配置 ${drafts.length} 条 · 自动保存`, `${drafts.length} configured · autosave`)}</Typography.Text><Button onClick={() => { const draftKey = `new-${nextDraftID.current++}`; setDrafts(current => [...current, { map_name: '', display_name: '', updated_at: 0, draftKey }]) }}>{label('新增映射', 'Add mapping')}</Button></div>
    {drafts.length === 0 && <Typography.Text type="secondary">{label('尚未配置自定义地图名称。', 'No custom map names configured.')}</Typography.Text>}
    {drafts.map((item, index) => <div className={styles.mapNameRow} key={item.draftKey}>
      <Input aria-label={label(`地图代码 ${index + 1}`, `Map code ${index + 1}`)} value={item.map_name} maxLength={128} placeholder="c1m1_hotel" onChange={event => replace(drafts.map(value => value.draftKey === item.draftKey ? { ...value, map_name: event.target.value } : value))} onBlur={() => { const values = normalized(drafts); if (values) autosave.schedule(values, true) }} />
      <Input aria-label={label(`显示名称 ${index + 1}`, `Display name ${index + 1}`)} value={item.display_name} maxLength={80} placeholder={label('死亡中心 1/4', 'Dead Center 1/4')} onChange={event => replace(drafts.map(value => value.draftKey === item.draftKey ? { ...value, display_name: event.target.value } : value))} onBlur={() => { const values = normalized(drafts); if (values) autosave.schedule(values, true) }} />
      <Button danger onClick={() => replace(drafts.filter(value => value.draftKey !== item.draftKey), true)}>{label('删除', 'Delete')}</Button>
    </div>)}
  </div>
}
