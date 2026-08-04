import { ArrowDownOutlined, ArrowUpOutlined, DeleteOutlined, EditOutlined, HolderOutlined, PlusOutlined, ReloadOutlined, RightOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Form, Input, Layout, Modal, Popconfirm, Select, Switch, Typography, message } from 'antd'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Navigate, Outlet, useNavigate } from 'react-router-dom'
import { api, APIError, resetCSRF, type FooterLink, type GameServer, type GameServerInput, type SiteSettings } from '../api'
import { FloatingNav } from '../components/FloatingNav'
import i18n from '../i18n'
import styles from './Portal.module.scss'

function validHTTPURL(value?: string) {
  if (!value) return true
  try { const url = new URL(value); return url.protocol === 'http:' || url.protocol === 'https:' } catch { return false }
}

function validPublicOrigin(value?: string) {
  if (!value) return false
  try {
    const url = new URL(value)
    return (url.protocol === 'http:' || url.protocol === 'https:') && url.username === '' && url.password === '' &&
      (url.pathname === '/' || url.pathname === '') && !url.search && !url.hash
  } catch { return false }
}

function displayTime(value?: string) {
  if (!value) return '—'
  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'medium' }).format(new Date(value))
}

export function AdminLayout() {
  const nav = useNavigate()
  const client = useQueryClient()
  const me = useQuery({ queryKey: ['admin-me'], queryFn: api.adminMe, retry: false })
  const logout = useMutation({
    mutationFn: api.logout,
    onSettled: () => {
      resetCSRF()
      client.clear()
      nav('/admin/login')
    },
  })
  if (me.isLoading) return <Layout className={styles.layout} />
  if (me.error instanceof APIError && me.error.status === 401) return <Navigate to="/admin/login" replace />
  return <Layout className={styles.adminShell}>
    <FloatingNav mode="admin" loggingOut={logout.isPending} monitorEnabled={me.data?.monitor_enabled} onLogout={() => logout.mutate()} />
    <Layout.Content className={styles.adminContent}><Outlet /></Layout.Content>
  </Layout>
}

export function AdminSitePage() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const query = useQuery({ queryKey: ['admin-site'], queryFn: api.adminSite })
  const save = useMutation({
    mutationFn: api.saveSite,
    onSuccess: async data => {
      client.setQueryData(['admin-site'], data)
      setFooterLinks(data.footer_links)
      void client.invalidateQueries({ queryKey: ['site'] })
      await i18n.changeLanguage(data.language)
      void message.success(i18n.t('saved'))
    },
  })
  const [form] = Form.useForm<SiteSettings>()
  const [footerEditorForm] = Form.useForm<FooterLink>()
  const [footerLinks, setFooterLinks] = useState<FooterLink[] | null>(null)
  const [footerEditor, setFooterEditor] = useState<{ index: number | null } | null>(null)
  const [draggingLink, setDraggingLink] = useState<number | null>(null)
  useEffect(() => {
    if (!query.data) return
    form.setFieldsValue(query.data)
  }, [form, query.data])
  const activeFooterLinks = footerLinks ?? query.data?.footer_links ?? []
  const origin = Form.useWatch('public_origin', form)
  const steamLoginEnabled = Form.useWatch('steam_openid_enabled', form)
  const footerEnabled = Form.useWatch('footer_enabled', form)
  const openFooterEditor = (index: number | null) => {
    footerEditorForm.resetFields()
    if (index !== null) footerEditorForm.setFieldsValue(activeFooterLinks[index])
    setFooterEditor({ index })
  }
  const saveFooterLink = (value: FooterLink) => {
    setFooterLinks(current => {
      const links = current ?? query.data?.footer_links ?? []
      return footerEditor?.index === null
        ? [...links, value]
        : links.map((link, index) => index === footerEditor?.index ? { ...link, ...value } : link)
    })
    setFooterEditor(null)
  }
  const moveFooterLink = (from: number, to: number) => {
    setFooterLinks(current => {
      const next = [...(current ?? query.data?.footer_links ?? [])]
      const [link] = next.splice(from, 1)
      next.splice(to, 0, link)
      return next
    })
  }

  return <div className={`${styles.adminPage} ${styles.siteSettingsPage}`}>
    <div className={styles.pageHeader}><Typography.Title level={2}>{t('siteSettings')}</Typography.Title></div>
    {save.isError && <Alert className={styles.inlineAlert} type="error" showIcon title={save.error.message} />}
    <Form form={form} layout="vertical" className={styles.settingsForm} onFinish={values => save.mutate({ ...values, footer_links: activeFooterLinks })}>
      <section className={styles.formSection}>
        <div className={styles.formSectionHeading}><strong>{t('siteLanguage')}</strong><span>{t('siteLanguageHint')}</span></div>
          <Form.Item name="language" rules={[{ required: true, message: t('requiredField') }]}>
          <Select options={[{ value: 'zh-CN', label: t('chinese') }, { value: 'en', label: t('english') }]} />
        </Form.Item>
      </section>
      <section className={styles.formSection}>
        <div className={styles.formSectionHeading}><strong>{t('siteTheme')}</strong><span>{t('siteThemeHint')}</span></div>
        <Form.Item name="theme" rules={[{ required: true, message: t('requiredField') }]}>
          <Select options={[{ value: 'light', label: t('lightTheme') }, { value: 'dark', label: t('darkTheme') }]} />
        </Form.Item>
      </section>
      <section className={styles.formSection}>
        <div className={styles.formSectionHeading}><strong>{t('browserTitle')}</strong><span>{t('browserTitleHint')}</span></div>
        <Form.Item name="browser_title" rules={[{ required: true, message: t('requiredField') }, { max: 80, message: t('browserTitleLength') }]}>
          <Input maxLength={80} showCount />
        </Form.Item>
      </section>
      <section className={styles.formSection}>
        <div className={styles.formSectionHeading}><strong>{t('siteBackground')}</strong><span>{t('siteBackgroundHint')}</span></div>
        <Form.Item name="background_image_url" rules={[{ validator: (_, value) => validHTTPURL(value) ? Promise.resolve() : Promise.reject(new Error(t('invalidURL'))) }]}>
          <Input placeholder="https://example.com/background.jpg" />
        </Form.Item>
      </section>
      <section className={styles.formSection}>
        <div className={styles.formSectionHeading}><strong>{t('a2sQuerySettings')}</strong><span>{t('a2sQuerySettingsHint')}</span></div>
        <div className={styles.querySettingsGrid}>
          <Form.Item name="a2s_refresh_seconds" label={t('a2sRefreshInterval')} rules={[{ required: true, message: t('requiredField') }]}>
            <Select options={[5, 10, 15, 30, 45, 60].map(value => ({ value, label: t('secondsValue', { value }) }))} />
          </Form.Item>
          <Form.Item name="a2s_jitter_seconds" label={t('a2sJitter')} rules={[{ required: true, message: t('requiredField') }]}>
            <Select options={[2, 5].map(value => ({ value, label: t('upToSeconds', { value }) }))} />
          </Form.Item>
          <Form.Item name="a2s_retry_count" label={t('a2sRetries')} rules={[{ required: true, message: t('requiredField') }]}>
            <Select options={[1, 2, 3].map(value => ({ value, label: t('timesValue', { value }) }))} />
          </Form.Item>
        </div>
      </section>
      <section className={styles.formSection}>
        <div className={styles.sectionTitleRow}>
          <strong>{t('showFooter')}</strong>
          <div className={styles.sectionActions}>
            {footerEnabled && <Button className={styles.sectionIconAction} type="text" icon={<PlusOutlined />}
              aria-label={t('addLink')} title={t('addLink')} onClick={() => openFooterEditor(null)} />}
            <Form.Item name="footer_enabled" valuePropName="checked" noStyle><Switch /></Form.Item>
          </div>
        </div>
        {footerEnabled && <div className={styles.footerLinks}>{activeFooterLinks.map((link, index) => <div className={`${styles.footerLinkRow} ${draggingLink === index ? styles.dragging : ''}`} key={link.id ?? `${link.label}-${index}`}
            onDragOver={event => event.preventDefault()}
            onDrop={() => { if (draggingLink !== null && draggingLink !== index) moveFooterLink(draggingLink, index); setDraggingLink(null) }}>
            <button className={styles.dragHandle} type="button" draggable aria-label={t('dragToSort')}
              onDragStart={() => setDraggingLink(index)} onDragEnd={() => setDraggingLink(null)}><HolderOutlined /></button>
            <div className={styles.footerLinkIdentity}><strong>{link.label}</strong><span>{link.url}</span></div>
            <div className={styles.footerLinkActions}>
              <Button type="text" icon={<EditOutlined />} aria-label={t('editLink')} title={t('editLink')} onClick={() => openFooterEditor(index)} />
              <Popconfirm title={t('confirmDeleteLink')} onConfirm={() => setFooterLinks(current => (current ?? query.data?.footer_links ?? []).filter((_, linkIndex) => linkIndex !== index))}>
                <Button danger type="text" icon={<DeleteOutlined />} aria-label={t('deleteLink')} title={t('deleteLink')} />
              </Popconfirm>
            </div>
          </div>)}</div>}
      </section>
      <section className={styles.formSection}>
        <div className={styles.sectionTitleRow}>
          <strong>{t('steamOpenID')}</strong>
          <Form.Item name="steam_openid_enabled" valuePropName="checked" noStyle><Switch /></Form.Item>
        </div>
        {steamLoginEnabled && <>
          <Form.Item name="public_origin" label={t('publicOrigin')} extra={t('publicOriginHint')} rules={[{ required: true, message: t('requiredField') }, { validator: (_, value) => validPublicOrigin(value) ? Promise.resolve() : Promise.reject(new Error(t('invalidOrigin'))) }]}><Input placeholder={window.location.origin} /></Form.Item>
          {String(origin ?? '').startsWith('http://') && <Alert type="warning" showIcon title={t('insecureOrigin')} />}
        </>}
      </section>
      <Button type="primary" htmlType="submit" loading={save.isPending}>{t('save')}</Button>
    </Form>
    <Modal open={footerEditor !== null} title={footerEditor?.index === null ? t('addLink') : t('editLink')}
      onCancel={() => setFooterEditor(null)} onOk={() => footerEditorForm.submit()} destroyOnHidden>
      <Form form={footerEditorForm} layout="vertical" onFinish={saveFooterLink}>
        <Form.Item name="label" label={t('label')} rules={[{ required: true, message: t('requiredField') }]}><Input /></Form.Item>
        <Form.Item name="url" label={t('linkAddress')} rules={[{ required: true, message: t('requiredField') }, { validator: (_, value) => validHTTPURL(value) ? Promise.resolve() : Promise.reject(new Error(t('invalidURL'))) }]}><Input placeholder="https://..." /></Form.Item>
      </Form>
    </Modal>
  </div>
}

const emptyServer: GameServerInput = { display_name: '', address: '' }

export function AdminServersPage() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const query = useQuery({ queryKey: ['admin-servers'], queryFn: api.servers })
  const site = useQuery({ queryKey: ['site'], queryFn: api.site, staleTime: 5 * 60_000 })
  const statuses = useQuery({ queryKey: ['server-statuses'], queryFn: api.serverStatuses, refetchInterval: (site.data?.a2s_refresh_seconds ?? 30) * 1000 })
  const [editing, setEditing] = useState<GameServer | null>(null)
  const [creating, setCreating] = useState(false)
  const [expandedID, setExpandedID] = useState<string | null>(null)
  const [form] = Form.useForm<GameServerInput>()
  const close = () => setEditing(null)
  const refresh = () => Promise.all([client.invalidateQueries({ queryKey: ['admin-servers'] }), client.invalidateQueries({ queryKey: ['server-statuses'] })])
  const save = useMutation({ mutationFn: (value: GameServerInput) => editing?.id ? api.updateServer(editing.id, value) : api.createServer(value), onSuccess: () => { void refresh(); close(); void message.success(t('saved')) } })
  const remove = useMutation({ mutationFn: api.deleteServer, onSuccess: () => void refresh() })
  const enable = useMutation({ mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => api.setServerEnabled(id, enabled), onSuccess: () => void refresh() })
  const move = useMutation({ mutationFn: ({ id, direction }: { id: string; direction: 'up' | 'down' }) => api.moveServer(id, direction), onSuccess: () => void refresh() })
  const openCreate = () => { form.resetFields(); form.setFieldsValue(emptyServer); setCreating(true); setEditing({ display_name: '', address: '', enabled: true, sort_order: 0 }) }
  const openEdit = (server: GameServer) => { form.resetFields(); form.setFieldsValue({ display_name: server.display_name, address: server.address }); setCreating(false); setEditing(server) }

  return <div className={styles.adminPage}>
    <div className={styles.toolbar}><div><Typography.Title level={2}>{t('serverManagement')}</Typography.Title><Typography.Text type="secondary">{t('serverHint')}</Typography.Text></div><Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>{t('addServer')}</Button></div>
    {query.isLoading && <div className={styles.listNotice}>Loading…</div>}
    <div className={styles.serverAdminList}>{query.data?.map((server, index, servers) => {
      const expanded = expandedID === server.id
      const status = statuses.data?.find(item => item.server_id === server.id)
      const statusState = !status ? 'unknown' : status.stale ? 'stale' : status.online ? 'online' : 'offline'
      return <article className={`${styles.serverAdminItem} ${expanded ? styles.expanded : ''}`} key={server.id}>
        <div className={styles.serverAdminRow}>
          <button className={styles.serverAdminToggle} type="button" aria-expanded={expanded} onClick={() => setExpandedID(expanded ? null : server.id!)}>
            <span className={styles.serverAdminIndex}>{String(index + 1).padStart(2, '0')}</span>
            <span className={styles.serverAdminIdentity}>
              <span className={styles.serverNameLine}><i className={`${styles.statusDot} ${styles[`statusDot_${statusState}`]}`} aria-label={t(statusState)} title={t(statusState)} /><strong>{server.display_name}</strong></span>
              <span>{server.address}</span>
            </span>
          </button>
          <div className={styles.serverAdminActions}>
            <Switch size="small" checked={server.enabled} aria-label={t('serverVisibility')} title={t('serverVisibility')} loading={enable.isPending} onChange={checked => enable.mutate({ id: server.id!, enabled: checked })} />
            <button className={styles.iconAction} aria-label={t('moveUp')} title={t('moveUp')} type="button" disabled={index === 0 || move.isPending} onClick={() => move.mutate({ id: server.id!, direction: 'up' })}><ArrowUpOutlined /></button>
            <button className={styles.iconAction} aria-label={t('moveDown')} title={t('moveDown')} type="button" disabled={index === servers.length - 1 || move.isPending} onClick={() => move.mutate({ id: server.id!, direction: 'down' })}><ArrowDownOutlined /></button>
            <button className={styles.iconAction} aria-label={t('editServer')} title={t('editServer')} type="button" onClick={() => openEdit(server)}><EditOutlined /></button>
            <Popconfirm title={t('confirmDelete')} onConfirm={() => remove.mutate(server.id!)}><button className={`${styles.iconAction} ${styles.danger}`} aria-label={t('confirmDelete')} title={t('confirmDelete')} type="button"><DeleteOutlined /></button></Popconfirm>
            <RightOutlined className={styles.expandIcon} aria-hidden="true" />
          </div>
        </div>
        <div className={`${styles.serverExpandRegion} ${expanded ? styles.open : ''}`}><div className={styles.serverExpandInner}>
          {expanded && <ServerA2SDetails server={server} />}
        </div></div>
      </article>
    })}</div>
    {!query.isLoading && query.data?.length === 0 && <div className={styles.listNotice}>{t('notConfigured')}</div>}
    <Modal open={editing !== null} title={creating ? t('addServer') : t('editServer')} onCancel={close} onOk={() => form.submit()} confirmLoading={save.isPending} destroyOnHidden>
      <Form form={form} layout="vertical" onFinish={value => save.mutate(value)}>
        <Form.Item name="display_name" label={t('displayName')} rules={[{ required: true, message: t('requiredField') }]}><Input /></Form.Item>
        <Form.Item name="address" label={t('serverAddress')} rules={[{ required: true, message: t('requiredField') }]}><Input placeholder="127.0.0.1:27015" /></Form.Item>
      </Form>
    </Modal>
  </div>
}

function ServerA2SDetails({ server }: { server: GameServer }) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const queryKey = ['admin-server-a2s', server.id]
  const query = useQuery({ queryKey, queryFn: () => api.serverA2S(server.id!), retry: false })
  const refresh = useMutation({
    mutationFn: () => api.refreshServerA2S(server.id!),
    onSuccess: status => {
      client.setQueryData(queryKey, { available: true, status })
      void client.invalidateQueries({ queryKey: ['server-statuses'] })
    },
  })
  const status = query.data?.status
  return <div className={styles.serverExpandedContent}>
    {query.isLoading && <span className={styles.a2sNotice}>{t('loading')}</span>}
    {!query.isLoading && !status && <span className={styles.a2sNotice}>{t('noA2SRecord')}</span>}
    {status && <div className={styles.a2sSummary}>
      <strong>{status.name || server.display_name}</strong>
      <span>{status.map || '—'}</span>
      <span>{status.players} / {status.max_players} {t('players')}</span>
      <span>{status.latency_ms == null ? '—' : `${status.latency_ms} ms`}</span>
      <span>{displayTime(status.checked_at)}</span>
    </div>}
    {query.isError && <span className={styles.a2sNotice}>{t('a2sLoadFailed')}</span>}
    <Button icon={<ReloadOutlined />} loading={refresh.isPending} onClick={() => refresh.mutate()}>{t('testA2SAgain')}</Button>
    {status && <div className={styles.rulesPanel}>
      {(status.rules?.length ?? 0) === 0 && <span className={styles.rulesNotice}>{t('noServerRules')}</span>}
      {(status.rules?.length ?? 0) > 0 && <>
        <div className={styles.ruleHeader}><span>{t('ruleName')}</span><span>{t('ruleValue')}</span></div>
        {status.rules?.map(rule => <div className={styles.ruleRow} key={rule.name}><strong>{rule.name}</strong><span>{rule.value}</span></div>)}
      </>}
    </div>}
  </div>
}

export function AdminSecurityPage() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const me = useQuery({ queryKey: ['admin-me'], queryFn: api.adminMe })
  const username = useMutation({ mutationFn: (v: { username: string }) => api.updateAccount(v.username), onSuccess: () => { void client.invalidateQueries({ queryKey: ['admin-me'] }); void message.success(t('saved')) } })
  const password = useMutation({ mutationFn: (v: { current_password: string; new_password: string }) => api.updatePassword(v.current_password, v.new_password), onSuccess: () => void message.success(t('passwordUpdated')) })

  return <div className={styles.adminPage}>
    <div className={styles.pageHeader}><Typography.Title level={2}>{t('account')}</Typography.Title></div>
    <div className={styles.accountGrid}>
      <section className={styles.accountColumn}>
        <Form layout="vertical" initialValues={{ username: me.data?.username }} onFinish={v => username.mutate(v)}>
          <Form.Item name="username" label={t('username')} rules={[{ required: true, message: t('usernameRequired') }, { min: 3, max: 64, message: t('usernameLength') }]}><Input /></Form.Item>
          <Button htmlType="submit" type="primary" loading={username.isPending}>{t('save')}</Button>
        </Form>
      </section>
      <section className={styles.accountColumn}>
        <Form layout="vertical" onFinish={v => password.mutate(v)}>
          <Form.Item name="current_password" label={t('currentPassword')} rules={[{ required: true, message: t('passwordRequired') }]}><Input.Password autoComplete="current-password" /></Form.Item>
          <Form.Item name="new_password" label={t('newPassword')} rules={[{ required: true, message: t('passwordRequired') }, { min: 12, max: 72, message: t('passwordLength') }]}><Input.Password autoComplete="new-password" /></Form.Item>
          <Form.Item name="confirm_password" label={t('confirmPassword')} dependencies={['new_password']} rules={[{ required: true, message: t('confirmPasswordRequired') }, ({ getFieldValue }) => ({ validator: (_, value) => !value || getFieldValue('new_password') === value ? Promise.resolve() : Promise.reject(new Error(t('passwordMismatch'))) })]}><Input.Password autoComplete="new-password" /></Form.Item>
          <Button htmlType="submit" loading={password.isPending}>{t('update')}</Button>
        </Form>
      </section>
    </div>
  </div>
}
