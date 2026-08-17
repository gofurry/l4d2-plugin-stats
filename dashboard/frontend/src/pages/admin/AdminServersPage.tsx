import { ArrowDownOutlined, ArrowUpOutlined, DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined, RightOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Button, Form, Input, Modal, Popconfirm, Switch, Typography, message } from 'antd'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, type GameServer, type GameServerInput } from '../../api'
import styles from '../Portal.module.scss'
import { ServerIngameSettingsEditor } from './ServerIngameSettingsEditor'

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
      const statusState = !status ? 'unknown' : status.stale ? 'stale' : status.checking ? 'checking' : status.online ? 'online' : 'offline'
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
          {expanded && <><ServerA2SDetails server={server} /><ServerIngameSettingsEditor server={server} /></>}
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

function displayTime(value?: string) {
  if (!value) return '—'
  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'medium' }).format(new Date(value))
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
