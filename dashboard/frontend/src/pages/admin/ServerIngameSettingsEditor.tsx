import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Form, Input, Select, Spin, Typography, message } from 'antd'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, type GameServer, type IngameServerDocument, type IngameServerSettings } from '../../api'
import styles from './AdminIngamePage.module.scss'

const inheritOverride = [{ value: 'inherit', cn: '继承全站', en: 'Inherit' }, { value: 'override', cn: '单独设置', en: 'Override' }]
const inheritOverrideHidden = [...inheritOverride, { value: 'hidden', cn: '隐藏', en: 'Hidden' }]

export function ServerIngameSettingsEditor({ server }: { server: GameServer }) {
  const { i18n } = useTranslation()
  const zh = !i18n.language.startsWith('en')
  const label = (cn: string, en: string) => zh ? cn : en
  const client = useQueryClient()
  const queryKey = ['admin-server-ingame', server.id]
  const query = useQuery({ queryKey, queryFn: () => api.serverIngameSettings(server.id!) })
  const [form] = Form.useForm<IngameServerSettings>()
  useEffect(() => { if (query.data) form.setFieldsValue(query.data.settings) }, [form, query.data])
  const save = useMutation({
    mutationFn: (settings: IngameServerSettings) => api.saveServerIngameSettings(server.id!, settings),
    onSuccess: settings => {
      client.setQueryData(queryKey, query.data ? { ...query.data, settings } : query.data)
      form.setFieldsValue(settings)
      void message.success(label('单服游戏内设置已保存', 'Server in-game settings saved'))
    },
  })
  const titleMode = Form.useWatch('title_mode', form)
  const descriptionMode = Form.useWatch('description_mode', form)
  const bannerMode = Form.useWatch('banner_mode', form)
  const websiteMode = Form.useWatch('website_mode', form)
  const highlightMode = Form.useWatch('highlight_mode', form)
  const modes = (values: typeof inheritOverrideHidden) => values.map(item => ({ value: item.value, label: zh ? item.cn : item.en }))
  const metrics = query.data?.metric_catalog.map(metric => ({ value: metric.key, label: metric.label })) ?? []
  if (query.isLoading) return <Spin />
  if (query.isError || !query.data) return <Alert type="error" showIcon title={query.error?.message ?? label('无法读取游戏内设置', 'Could not load in-game settings')} />

  return <section className={styles.serverOverride}>
    <div className={styles.serverOverrideHeading}><div><Typography.Title level={4}>{label('游戏内页面覆盖', 'In-game portal overrides')}</Typography.Title><Typography.Text type="secondary">{label('只影响从这台服务器进入的 MOTD 页面。', 'Only affects MOTD pages opened from this server.')}</Typography.Text></div><span>{query.data.server_key ? `Server key: ${query.data.server_key}` : label('尚无 server key', 'No server key yet')}</span></div>
    <Form form={form} layout="vertical" onFinish={value => save.mutate({ ...query.data.settings, ...value })}>
      <div className={styles.overrideGrid}>
        <div><Form.Item name="title_mode" label={label('标题', 'Title')}><Select options={modes(inheritOverride)} /></Form.Item>{titleMode === 'override' && <Form.Item name="title" rules={[{ required: true }, { max: 128 }]}><Input maxLength={128} /></Form.Item>}</div>
        <div><Form.Item name="description_mode" label={label('描述', 'Description')}><Select options={modes(inheritOverrideHidden)} /></Form.Item>{descriptionMode === 'override' && <Form.Item name="description" rules={[{ max: 1000 }]}><Input.TextArea rows={3} maxLength={1000} /></Form.Item>}</div>
        <div><Form.Item name="banner_mode" label="Banner"><Select options={modes(inheritOverrideHidden)} /></Form.Item>{bannerMode === 'override' && <Form.Item name="banner_url" rules={[{ required: true }, { type: 'url' }]}><Input placeholder="https://example.com/banner.jpg" /></Form.Item>}</div>
        <div><Form.Item name="website_mode" label={label('完整网站', 'Full website')}><Select options={modes(inheritOverrideHidden)} /></Form.Item>{websiteMode === 'override' && <Form.Item name="website_url" rules={[{ required: true }, { type: 'url' }]}><Input placeholder="https://stats.example.com" /></Form.Item>}</div>
      </div>
      <Form.Item name="highlight_mode" label={label('生涯亮点', 'Highlights')}><Select options={modes(inheritOverride)} /></Form.Item>
      {highlightMode === 'override' && <div className={styles.threeColumns}>{[0, 1, 2].map(index => <Form.Item key={index} name={['highlight_metrics', index]} label={`${label('指标', 'Metric')} ${index + 1}`} rules={[{ required: true }]}><Select options={metrics} /></Form.Item>)}</div>}
      <Button type="primary" htmlType="submit" loading={save.isPending}>{label('保存单服设置', 'Save server settings')}</Button>
    </Form>
    <div className={styles.documentSection}>
      <Typography.Title level={5}>{label('服务器文档', 'Server documents')}</Typography.Title>
      {query.data.documents.map(document => <ServerDocumentEditor key={document.key} serverID={server.id!} document={document} zh={zh} queryKey={queryKey} />)}
    </div>
  </section>
}

function ServerDocumentEditor({ serverID, document, zh, queryKey }: { serverID: string; document: IngameServerDocument; zh: boolean; queryKey: unknown[] }) {
  const label = (cn: string, en: string) => zh ? cn : en
  const client = useQueryClient()
  const [draft, setDraft] = useState(document)
  useEffect(() => setDraft(document), [document])
  const save = useMutation({
    mutationFn: (value: IngameServerDocument) => api.saveServerIngameDocument(serverID, value),
    onSuccess: updated => {
      client.setQueryData(queryKey, (current: { documents: IngameServerDocument[] } | undefined) => current ? { ...current, documents: current.documents.map(item => item.key === updated.key ? updated : item) } : current)
      void message.success(label('文档设置已保存', 'Document setting saved'))
    },
  })
  const names = { introduction: label('本服简介', 'Introduction'), commands: label('本服指令', 'Commands'), resources: label('本服资源', 'Resources') }
  const options = inheritOverrideHidden.map(item => ({ value: item.value, label: zh ? item.cn : item.en }))
  return <div className={styles.documentEditor}>
    <div className={styles.documentToolbar}><strong>{names[draft.key]}</strong><Select value={draft.mode} options={options} onChange={mode => setDraft(current => ({ ...current, mode: mode as IngameServerDocument['mode'] }))} /></div>
    {draft.mode === 'override' && <Input.TextArea rows={5} value={draft.content_markdown} onChange={event => setDraft(current => ({ ...current, content_markdown: event.target.value }))} placeholder="Markdown" maxLength={102400} />}
    <Button loading={save.isPending} disabled={draft.mode === 'override' && !draft.content_markdown.trim()} onClick={() => save.mutate(draft)}>{label('保存文档', 'Save document')}</Button>
  </div>
}
