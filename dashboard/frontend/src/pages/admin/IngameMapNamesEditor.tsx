import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Input, Spin, Typography, message } from 'antd'
import { useRef, useState } from 'react'
import { api, type IngameMapName } from '../../api'
import styles from './AdminIngamePage.module.scss'

type MapNameDraft = IngameMapName & { draftKey: string }

export function IngameMapNamesEditor({ zh }: { zh: boolean }) {
  const queryKey = ['admin-ingame-map-names']
  const query = useQuery({ queryKey, queryFn: api.ingameMapNames })
  if (query.isLoading) return <Spin />
  if (query.isError) return <Alert type="error" showIcon title={query.error.message} />
  const values = query.data ?? []
  return <MapNamesDraftEditor key={values.map(item => `${item.map_name}:${item.updated_at}`).join('|')} initialValues={values} zh={zh} queryKey={queryKey} />
}

function MapNamesDraftEditor({ initialValues, zh, queryKey }: { initialValues: IngameMapName[]; zh: boolean; queryKey: string[] }) {
  const label = (cn: string, en: string) => zh ? cn : en
  const client = useQueryClient()
  const nextDraftID = useRef(0)
  const [drafts, setDrafts] = useState<MapNameDraft[]>(() => initialValues.map((item, index) => ({ ...item, draftKey: `saved-${index}` })))
  const save = useMutation({
    mutationFn: (values: IngameMapName[]) => api.saveIngameMapNames(values),
    onSuccess: values => {
      client.setQueryData(queryKey, values)
      setDrafts(values.map((item, index) => ({ ...item, draftKey: `saved-${index}` })))
      void message.success(label('自定义地图名称已保存', 'Custom map names saved'))
    },
  })
  const update = (draftKey: string, patch: Partial<IngameMapName>) => setDrafts(current => current.map(item => item.draftKey === draftKey ? { ...item, ...patch } : item))
  const submit = () => {
    const normalized = drafts.map(item => ({ map_name: item.map_name.trim().toLowerCase(), display_name: item.display_name.trim(), updated_at: item.updated_at }))
      .sort((left, right) => left.map_name.localeCompare(right.map_name))
    const names = new Set(normalized.map(item => item.map_name))
    if (normalized.some(item => !item.map_name || [...item.map_name].length > 128 || !item.display_name || [...item.display_name].length > 80) || names.size !== normalized.length) {
      void message.error(label('地图代码需唯一且为 1-128 字符，显示名称需为 1-80 字符', 'Map codes must be unique (1-128 characters) and display names must contain 1-80 characters'))
      return
    }
    save.mutate(normalized)
  }
  return <div className={styles.mapNameEditor}>
    <Typography.Paragraph type="secondary">{label('这里只保存三方图名称或对官方名称的覆盖；内置官方地图不会复制到数据库。', 'Only custom maps and official-name overrides are stored here; the built-in official catalog is not copied into the database.')}</Typography.Paragraph>
    <div className={styles.mapNameToolbar}><Typography.Text>{label(`已配置 ${drafts.length} 条`, `${drafts.length} configured`)}</Typography.Text><Button onClick={() => { const draftKey = `new-${nextDraftID.current++}`; setDrafts(current => [...current, { map_name: '', display_name: '', updated_at: 0, draftKey }]) }}>{label('新增映射', 'Add mapping')}</Button></div>
    {drafts.length === 0 && <Typography.Text type="secondary">{label('尚未配置自定义地图名称。', 'No custom map names configured.')}</Typography.Text>}
    {drafts.map((item, index) => <div className={styles.mapNameRow} key={item.draftKey}>
      <Input aria-label={label(`地图代码 ${index + 1}`, `Map code ${index + 1}`)} value={item.map_name} maxLength={128} placeholder="c1m1_hotel" onChange={event => update(item.draftKey, { map_name: event.target.value })} />
      <Input aria-label={label(`显示名称 ${index + 1}`, `Display name ${index + 1}`)} value={item.display_name} maxLength={80} placeholder={label('死亡中心 1/4', 'Dead Center 1/4')} onChange={event => update(item.draftKey, { display_name: event.target.value })} />
      <Button danger onClick={() => setDrafts(current => current.filter(value => value.draftKey !== item.draftKey))}>{label('删除', 'Delete')}</Button>
    </div>)}
    <Button type="primary" loading={save.isPending} onClick={submit}>{label('保存地图名称', 'Save map names')}</Button>
  </div>
}
