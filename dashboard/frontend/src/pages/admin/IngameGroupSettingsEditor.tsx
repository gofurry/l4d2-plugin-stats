import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Form, Input, Select, Spin, Switch, Typography, message } from 'antd'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, type IngameGroup, type IngameQuickLink, type IngameServerDocument, type IngameServerSettings } from '../../api'
import styles from './AdminIngamePage.module.scss'

const inheritOverride = [{ value: 'inherit', cn: '继承全站', en: 'Inherit' }, { value: 'override', cn: '单独设置', en: 'Override' }]
const inheritOverrideHidden = [...inheritOverride, { value: 'hidden', cn: '隐藏', en: 'Hidden' }]

function validHTTPURL(value?: string) {
  if (!value?.trim()) return true
  try {
    const url = new URL(value)
    return value.length <= 2048 && (url.protocol === 'http:' || url.protocol === 'https:') && Boolean(url.host) && !url.username && !url.password
  } catch { return false }
}

export function IngameGroupSettingsEditor({ group }: { group: IngameGroup }) {
  const { i18n } = useTranslation()
  const zh = !i18n.language.startsWith('en')
  const label = (cn: string, en: string) => zh ? cn : en
  const client = useQueryClient()
  const queryKey = ['admin-ingame-group', group.server_key]
  const query = useQuery({ queryKey, queryFn: () => api.ingameGroup(group.server_key) })
  const [form] = Form.useForm<IngameServerSettings>()
  useEffect(() => { if (query.data) form.setFieldsValue(query.data.settings) }, [form, query.data])
  const save = useMutation({
    mutationFn: (settings: IngameServerSettings) => api.saveIngameGroup(group.server_key, settings),
    onSuccess: settings => {
      client.setQueryData(queryKey, query.data ? { ...query.data, settings } : query.data)
      form.setFieldsValue(settings)
      void client.invalidateQueries({ queryKey: ['admin-ingame-groups'] })
      void message.success(label('服务器组游戏内设置已保存', 'Server-group in-game settings saved'))
    },
  })
  const titleMode = Form.useWatch('title_mode', form)
  const descriptionMode = Form.useWatch('description_mode', form)
  const bannerMode = Form.useWatch('banner_mode', form)
  const backgroundMode = Form.useWatch('background_mode', form)
  const websiteMode = Form.useWatch('website_mode', form)
  const highlightMode = Form.useWatch('highlight_mode', form)
  const modes = (values: typeof inheritOverrideHidden) => values.map(item => ({ value: item.value, label: zh ? item.cn : item.en }))
  const metrics = query.data?.metric_catalog.map(metric => ({ value: metric.key, label: metric.label })) ?? []
  if (query.isLoading) return <Spin />
  if (query.isError || !query.data) return <Alert type="error" showIcon title={query.error?.message ?? label('无法读取服务器组设置', 'Could not load server-group settings')} />

  return <section className={styles.serverOverride}>
    <div className={styles.serverOverrideHeading}><div><Typography.Title level={4}>{query.data.title}</Typography.Title><Typography.Text type="secondary">Server key: {group.server_key}</Typography.Text></div></div>
    <div className={styles.instanceList}>{query.data.instances.map(instance => <span key={instance.server_id}><i className={instance.online ? styles.instanceOnline : styles.instanceOffline} />{instance.name} · {instance.address}</span>)}</div>
    <Form form={form} layout="vertical" onFinish={value => save.mutate({ ...query.data.settings, ...value })}>
      <Form.Item name="short_description" label={label('简短描述', 'Short description')} extra={label('仅用于服务器组选择页和游戏内首页，例如：官图 · 8人 · 上海。', 'Used on the server-group selection page and portal home, e.g. Official · 8 players · Shanghai.')} rules={[{ max: 80 }]}><Input maxLength={80} showCount /></Form.Item>
      <div className={styles.overrideGrid}>
        <div><Form.Item name="title_mode" label={label('标题', 'Title')}><Select options={modes(inheritOverride)} /></Form.Item>{titleMode === 'override' && <Form.Item name="title" rules={[{ required: true }, { max: 128 }]}><Input maxLength={128} /></Form.Item>}</div>
        <div><Form.Item name="description_mode" label={label('描述', 'Description')}><Select options={modes(inheritOverrideHidden)} /></Form.Item>{descriptionMode === 'override' && <Form.Item name="description" rules={[{ max: 1000 }]}><Input.TextArea rows={3} maxLength={1000} /></Form.Item>}</div>
        <div><Form.Item name="banner_mode" label="Banner"><Select options={modes(inheritOverrideHidden)} /></Form.Item>{bannerMode === 'override' && <Form.Item name="banner_url" rules={[{ required: true }, { validator: (_, value) => validHTTPURL(value) ? Promise.resolve() : Promise.reject(new Error(label('请输入不含账号密码的 HTTP/HTTPS 地址', 'Enter a credential-free HTTP/HTTPS URL'))) }]}><Input placeholder="https://example.com/banner.jpg?v=2" /></Form.Item>}</div>
        <div><Form.Item name="background_mode" label={label('背景图片', 'Background image')}><Select options={modes(inheritOverrideHidden)} /></Form.Item>{backgroundMode === 'override' && <Form.Item name="background_url" rules={[{ required: true }, { validator: (_, value) => validHTTPURL(value) ? Promise.resolve() : Promise.reject(new Error(label('请输入不含账号密码的 HTTP/HTTPS 地址', 'Enter a credential-free HTTP/HTTPS URL'))) }]}><Input placeholder="https://example.com/background.jpg?v=2" /></Form.Item>}</div>
        <div><Form.Item name="website_mode" label={label('完整网站', 'Full website')}><Select options={modes(inheritOverrideHidden)} /></Form.Item>{websiteMode === 'override' && <Form.Item name="website_url" rules={[{ required: true }, { validator: (_, value) => validHTTPURL(value) ? Promise.resolve() : Promise.reject(new Error(label('请输入不含账号密码的 HTTP/HTTPS 地址', 'Enter a credential-free HTTP/HTTPS URL'))) }]}><Input placeholder="https://stats.example.com" /></Form.Item>}</div>
      </div>
      <Form.Item name="highlight_mode" label={label('生涯亮点', 'Highlights')}><Select options={modes(inheritOverride)} /></Form.Item>
      {highlightMode === 'override' && <div className={styles.threeColumns}>{[0, 1, 2].map(index => <Form.Item key={index} name={['highlight_metrics', index]} label={`${label('指标', 'Metric')} ${index + 1}`} rules={[{ required: true }]}><Select options={metrics} /></Form.Item>)}</div>}
      <Button type="primary" htmlType="submit" loading={save.isPending}>{label('保存服务器组设置', 'Save server-group settings')}</Button>
    </Form>
    <GroupQuickLinksEditor key={`${group.server_key}-${query.data.quick_links.map(item => `${item.sort_order}:${item.label}:${item.url}:${item.enabled}`).join('|')}`} serverKey={group.server_key} initialLinks={query.data.quick_links} zh={zh} queryKey={queryKey} />
    <div className={styles.documentSection}>
      <Typography.Title level={5}>{label('服务器组文档', 'Server-group documents')}</Typography.Title>
      {query.data.documents.map(document => <GroupDocumentEditor key={`${document.key}-${document.updated_at}`} serverKey={group.server_key} document={document} zh={zh} queryKey={queryKey} />)}
    </div>
  </section>
}

function GroupQuickLinksEditor({ serverKey, initialLinks, zh, queryKey }: { serverKey: string; initialLinks: IngameQuickLink[]; zh: boolean; queryKey: unknown[] }) {
  const label = (cn: string, en: string) => zh ? cn : en
  const client = useQueryClient()
  const nextDraftID = useRef(initialLinks.length)
  const [quickLinks, setQuickLinks] = useState(() => initialLinks.map((item, index) => ({ ...item, draftKey: `saved-${index}` })))
  const save = useMutation({
    mutationFn: (links: IngameQuickLink[]) => api.saveIngameGroupQuickLinks(serverKey, links),
    onSuccess: links => {
      client.setQueryData(queryKey, (current: { quick_links: IngameQuickLink[] } | undefined) => current ? { ...current, quick_links: links } : current)
      setQuickLinks(links.map((item, index) => ({ ...item, draftKey: `saved-${index}` })))
      void message.success(label('快速链接已保存', 'Quick links saved'))
    },
  })
  const update = (index: number, patch: Partial<IngameQuickLink>) => setQuickLinks(current => current.map((item, itemIndex) => itemIndex === index ? { ...item, ...patch } : item))
  const move = (index: number, direction: -1 | 1) => setQuickLinks(current => {
    const target = index + direction
    if (target < 0 || target >= current.length) return current
    const next = [...current]
    ;[next[index], next[target]] = [next[target], next[index]]
    return next
  })
  const submit = () => {
    const normalized = quickLinks.map((item, index) => ({ server_key: serverKey, label: item.label.trim(), url: item.url.trim(), sort_order: index, enabled: item.enabled }))
    if (normalized.some(item => !item.label || [...item.label].length > 32 || !item.url || !validHTTPURL(item.url))) {
      void message.error(label('请填写 1-32 字符名称和有效的 HTTP/HTTPS 地址', 'Enter a 1-32 character label and a valid HTTP/HTTPS URL'))
      return
    }
    save.mutate(normalized)
  }
  return <div className={styles.quickLinkSection}>
    <div className={styles.quickLinkHeading}><div><Typography.Title level={5}>{label('快速链接', 'Quick links')}</Typography.Title><Typography.Text type="secondary">{label('最多 8 条，仅允许 HTTP/HTTPS；游戏内只显示供玩家手动复制的地址。', 'Up to 8 HTTP/HTTPS links; the in-game portal only shows addresses for manual copying.')}</Typography.Text></div><Button disabled={quickLinks.length >= 8} onClick={() => { const draftKey = `new-${nextDraftID.current++}`; setQuickLinks(current => [...current, { server_key: serverKey, label: '', url: '', sort_order: current.length, enabled: true, draftKey }]) }}>{label('新增链接', 'Add link')}</Button></div>
    {quickLinks.length === 0 && <Typography.Text type="secondary">{label('尚未配置快速链接。', 'No quick links configured.')}</Typography.Text>}
    {quickLinks.map((link, index) => <div className={styles.quickLinkRow} key={link.draftKey}>
      <Input aria-label={label(`链接 ${index + 1} 名称`, `Link ${index + 1} label`)} value={link.label} maxLength={32} placeholder={label('名称，例如：地图合集', 'Label, e.g. Map collection')} onChange={event => update(index, { label: event.target.value })} />
      <Input aria-label={label(`链接 ${index + 1} 地址`, `Link ${index + 1} URL`)} value={link.url} maxLength={2048} placeholder="https://example.com" onChange={event => update(index, { url: event.target.value })} />
      <Switch checked={link.enabled} checkedChildren={label('启用', 'On')} unCheckedChildren={label('停用', 'Off')} onChange={enabled => update(index, { enabled })} />
      <div className={styles.quickLinkActions}><Button size="small" disabled={index === 0} onClick={() => move(index, -1)}>{label('上移', 'Up')}</Button><Button size="small" disabled={index === quickLinks.length - 1} onClick={() => move(index, 1)}>{label('下移', 'Down')}</Button><Button size="small" danger onClick={() => setQuickLinks(current => current.filter((_, itemIndex) => itemIndex !== index))}>{label('删除', 'Delete')}</Button></div>
    </div>)}
    <Button disabled={quickLinks.length === 0 && initialLinks.length === 0} loading={save.isPending} onClick={submit}>{label('保存快速链接', 'Save quick links')}</Button>
  </div>
}

function GroupDocumentEditor({ serverKey, document, zh, queryKey }: { serverKey: string; document: IngameServerDocument; zh: boolean; queryKey: unknown[] }) {
  const label = (cn: string, en: string) => zh ? cn : en
  const client = useQueryClient()
  const [draft, setDraft] = useState(document)
  const save = useMutation({
    mutationFn: (value: IngameServerDocument) => api.saveIngameGroupDocument(serverKey, value),
    onSuccess: updated => {
      client.setQueryData(queryKey, (current: { documents: IngameServerDocument[] } | undefined) => current ? { ...current, documents: current.documents.map(item => item.key === updated.key ? updated : item) } : current)
      void message.success(label('文档设置已保存', 'Document setting saved'))
    },
  })
  const names = { introduction: label('本组简介', 'Introduction'), commands: label('本组指令', 'Commands'), resources: label('本组资源', 'Resources') }
  const options = inheritOverrideHidden.map(item => ({ value: item.value, label: zh ? item.cn : item.en }))
  return <div className={styles.documentEditor}>
    <div className={styles.documentToolbar}><strong>{names[draft.key]}</strong><Select value={draft.mode} options={options} onChange={mode => setDraft(current => ({ ...current, mode: mode as IngameServerDocument['mode'] }))} /></div>
    {draft.mode === 'override' && <Input.TextArea rows={5} value={draft.content_markdown} onChange={event => setDraft(current => ({ ...current, content_markdown: event.target.value }))} placeholder="Markdown" maxLength={102400} />}
    <Button loading={save.isPending} disabled={draft.mode === 'override' && !draft.content_markdown.trim()} onClick={() => save.mutate(draft)}>{label('保存文档', 'Save document')}</Button>
  </div>
}
