import { DownloadOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, DatePicker, Form, Input, Modal, Select, Space, Spin, Switch, Table, Tabs, Typography, message } from 'antd'
import dayjs, { type Dayjs } from 'dayjs'
import { useEffect, useState } from 'react'
import { api, type ChatMessage, type ChatRetentionPlan, type ChatSearchFilter, type ConnectionAuditFilter, type ConnectionAuditRow } from '../../api'
import styles from '../Portal.module.scss'

const { RangePicker } = DatePicker
const retentionOptions = [7, 14, 30, 60, 90, 180, 365, 0].map(value => ({ value, label: value === 0 ? '永久' : `${value} 天` }))
const initialRange = (): [Dayjs, Dayjs] => [dayjs().subtract(24, 'hour'), dayjs()]
const time = (value = 0) => value ? new Date(value * 1000).toLocaleString() : '-'
const duration = (seconds: number) => seconds < 60 ? `${seconds}s` : `${Math.floor(seconds / 60)}m`

export function AdminAuditPage() {
  return <div className={styles.adminPage}>
    <div className={styles.toolbar}><div><Typography.Title level={2}>审计</Typography.Title><Typography.Text type="secondary">仅管理员可访问的连接、聊天与近似 IP 位置记录。</Typography.Text></div></div>
    <Tabs items={[{ key: 'connections', label: '连接记录', children: <ConnectionsAudit /> }, { key: 'chat', label: '聊天记录', children: <ChatAudit /> }]} />
  </div>
}

function ConnectionsAudit() {
  const client = useQueryClient()
  const [filter, setFilter] = useState<ConnectionAuditFilter>({ from: dayjs().subtract(24, 'hour').unix(), to: dayjs().unix(), limit: 100 })
  const [range, setRange] = useState<[Dayjs, Dayjs]>(initialRange)
  const geo = useQuery({ queryKey: ['admin-geoip-settings'], queryFn: api.geoIPSettings })
  const search = useMutation({ mutationFn: api.searchConnections })
  const saveGeo = useMutation({ mutationFn: api.saveGeoIPSettings, onSuccess: () => { void message.success('GeoIP 设置已保存'); void client.invalidateQueries({ queryKey: ['admin-geoip-settings'] }) }, onError: error => void message.error(error instanceof Error ? error.message : '保存失败') })
  const testGeo = useMutation({ mutationFn: api.testGeoIP, onSuccess: result => void message.success(`测试成功：${[result.country, result.province, result.city].filter(Boolean).join(' ')}`), onError: error => void message.error(error instanceof Error ? error.message : '测试失败') })
  const [geoEnabled, setGeoEnabled] = useState<boolean>()
  const [newKey, setNewKey] = useState('')
  const [testIP, setTestIP] = useState('')
  useEffect(() => { search.mutate(filter) }, []) // eslint-disable-line react-hooks/exhaustive-deps
	const effectiveGeoEnabled = geoEnabled ?? geo.data?.enabled ?? false
  const submit = (values: Record<string, string>) => {
    const next: ConnectionAuditFilter = { ...values, from: range[0].unix(), to: range[1].unix(), limit: 100 }
    setFilter(next)
    search.mutate(next)
  }
  const columns = [
    { title: '时间', dataIndex: 'started_at', render: time },
    { title: '玩家', key: 'player', render: (_: unknown, row: ConnectionAuditRow) => <><strong>{row.player_name}</strong><br/><Typography.Text type="secondary" copyable>{row.steam_id}</Typography.Text></> },
    { title: '服务器', dataIndex: 'server_key' },
    { title: 'IP', dataIndex: 'ip_address', render: (value: string) => <Typography.Text copyable>{value}</Typography.Text> },
    { title: '近似位置', key: 'location', render: (_: unknown, row: ConnectionAuditRow) => row.geoip ? [row.geoip.country, row.geoip.province, row.geoip.city].filter(Boolean).join(' ') || row.geoip.status : '待解析' },
    { title: '连接时长', dataIndex: 'connected_seconds', render: duration },
    { title: '状态 / 原因', key: 'status', render: (_: unknown, row: ConnectionAuditRow) => `${row.status}${row.disconnect_reason ? ` / ${row.disconnect_reason}` : ''}` },
  ]
  return <>
    <section className={styles.dataSection}>
      <div className={styles.formSectionHeading}><strong>GeoIP 设置</strong><span>仅服务端使用百度普通 IP 定位；Dashboard 缓存不保存原始 IP，位置为城市级近似结果。</span></div>
      {geo.isLoading ? <Spin size="small"/> : <div className={`${styles.dataStatusRows} ${styles.threePairRows}`}>
        <span>Provider</span><strong>Baidu</strong><span>AK</span><strong>{geo.data?.api_key_configured ? geo.data.api_key_masked : '未配置'}</strong><span>缓存</span><strong>{geo.data?.cache_count ?? 0}</strong>
        <span>IPv4</span><strong>{geo.data?.ipv4_status ?? 'unknown'}</strong><span>IPv6</span><strong>{geo.data?.ipv6_status ?? 'unknown'}</strong><span>待处理</span><strong>{geo.data?.pending_count ?? 0}</strong>
		<span>上次成功</span><strong>{time(geo.data?.last_success_at)}</strong><span>最近错误</span><strong>{time(geo.data?.last_error_at)}</strong><span>错误状态</span><strong>{geo.data?.last_error_code || '-'}</strong>
      </div>}
      <Space wrap>
        <Switch checked={effectiveGeoEnabled} onChange={setGeoEnabled} checkedChildren="启用" unCheckedChildren="禁用" />
        <Input.Password value={newKey} onChange={event => setNewKey(event.target.value)} placeholder="新 Baidu AK（留空保留原值）" style={{ width: 280 }} />
        <Button type="primary" loading={saveGeo.isPending} onClick={() => saveGeo.mutate({ enabled: effectiveGeoEnabled, api_key: newKey })}>保存</Button>
		<Button danger disabled={!geo.data?.api_key_configured || saveGeo.isPending} onClick={() => Modal.confirm({ title: '确认清除 Baidu AK？', content: 'GeoIP 将同时停用；现有 HMAC 缓存不会暴露或删除原始 IP。', okButtonProps: { danger: true }, onOk: async () => { await saveGeo.mutateAsync({ enabled: false, clear_api_key: true }); setGeoEnabled(false); setNewKey('') } })}>清除 AK</Button>
        <Input value={testIP} onChange={event => setTestIP(event.target.value)} placeholder="公开测试 IP" style={{ width: 180 }} />
        <Button loading={testGeo.isPending} onClick={() => testGeo.mutate(testIP.trim())}>测试配置</Button>
      </Space>
      {geo.data?.last_error_code && <Alert type="warning" showIcon message={`最近错误：${geo.data.last_error_code}`} />}
    </section>
    <section className={styles.dataSection}>
      <Form layout="inline" onFinish={submit}>
        <Form.Item><RangePicker showTime value={range} onChange={value => value && setRange(value as [Dayjs, Dayjs])}/></Form.Item>
        <Form.Item name="server_key"><Input placeholder="server_key"/></Form.Item><Form.Item name="steam_id"><Input placeholder="SteamID64"/></Form.Item>
        <Form.Item name="nickname"><Input placeholder="玩家名"/></Form.Item><Form.Item name="ip_address"><Input placeholder="IP"/></Form.Item><Form.Item name="location"><Input placeholder="国家/省/市"/></Form.Item>
        <Form.Item><Button htmlType="submit" type="primary" icon={<SearchOutlined/>}>查询</Button></Form.Item>
      </Form>
      <Table<ConnectionAuditRow> columns={columns} dataSource={search.data?.items ?? []} rowKey="session_id" loading={search.isPending} pagination={false} scroll={{ x: 1050 }} />
      {search.data?.next_cursor_id && <Button onClick={() => { const next = { ...filter, cursor_at: search.data?.next_cursor_at, cursor_id: search.data?.next_cursor_id }; setFilter(next); search.mutate(next) }}>下一页</Button>}
    </section>
  </>
}

function ChatAudit() {
  const client = useQueryClient()
  const settings = useQuery({ queryKey: ['admin-chat-settings'], queryFn: api.chatAuditSettings })
  const status = useQuery({ queryKey: ['admin-chat-status'], queryFn: api.chatAuditStatus })
  const [range, setRange] = useState<[Dayjs, Dayjs]>(initialRange)
  const [filter, setFilter] = useState<ChatSearchFilter>({ from: dayjs().subtract(24, 'hour').unix(), to: dayjs().unix(), limit: 100 })
  const search = useMutation({ mutationFn: api.searchChatAudit })
  const save = useMutation({ mutationFn: api.saveChatAuditSettings, onSuccess: result => {
    if ('plan_id' in result && result.plan_id) confirmRetention(result)
    else { void message.success('聊天审计设置已保存'); void client.invalidateQueries({ queryKey: ['admin-chat-settings'] }) }
  } })
  const download = useMutation({ mutationFn: ({ format, value }: { format: 'csv' | 'jsonl'; value: ChatSearchFilter }) => api.exportChatAudit(format, value), onSuccess: (blob, variables) => {
    const link = document.createElement('a'); link.href = URL.createObjectURL(blob); link.download = `chat-audit.${variables.format}`; link.click(); URL.revokeObjectURL(link.href)
  } })
  useEffect(() => { search.mutate(filter) }, []) // eslint-disable-line react-hooks/exhaustive-deps
  const confirmRetention = (plan: ChatRetentionPlan) => Modal.confirm({ title: '确认缩短聊天保留时间？', content: `将分批删除约 ${plan.delete_count.toLocaleString()} 条聊天记录。`, okButtonProps: { danger: true }, onOk: async () => {
    if (!settings.data) return
    await api.confirmChatAuditSettings(plan.plan_id, { ...settings.data, retention_days: plan.retention_days })
    void client.invalidateQueries({ queryKey: ['admin-chat-settings'] }); void client.invalidateQueries({ queryKey: ['admin-chat-status'] })
  } })
  const submit = (values: Record<string, string>) => { const next = { ...values, from: range[0].unix(), to: range[1].unix(), limit: 100 }; setFilter(next); search.mutate(next) }
  const columns = [
    { title: '时间', dataIndex: 'occurred_at', render: time }, { title: '服务器', dataIndex: 'server_key' },
    { title: '玩家', key: 'player', render: (_: unknown, row: ChatMessage) => <><strong>{row.player_name}</strong><br/><Typography.Text type="secondary">{row.steam_id || `userid:${row.source_user_id}`}</Typography.Text></> },
    { title: '环境', key: 'context', render: (_: unknown, row: ChatMessage) => `${row.map_name || '-'} · ${row.game_mode || '-'} · ${row.team}` },
    { title: '频道', key: 'channel', render: (_: unknown, row: ChatMessage) => `${row.channel}${row.command_like ? ' · 命令样式' : ''}` },
    { title: '内容', dataIndex: 'content' },
  ]
  return <>
    <section className={styles.dataSection}>
      <div className={styles.sectionTitleRow}><div className={styles.formSectionHeading}><strong>聊天审计状态</strong><span>Collector 临时 outbox 会被只读摄取到独立 chat-audit.db。</span></div><Button icon={<ReloadOutlined/>} onClick={() => { void settings.refetch(); void status.refetch() }}>刷新</Button></div>
      <div className={`${styles.dataStatusRows} ${styles.threePairRows}`}>
        <span>消息</span><strong>{status.data?.message_count.toLocaleString() ?? '-'}</strong><span>摄取延迟</span><strong>{status.data?.ingestion_lag ?? '-'}</strong><span>已知缺口</span><strong>{status.data?.known_gap_count ?? '-'}</strong>
        <span>Collector 丢弃</span><strong>{status.data?.dropped_count ?? '-'}</strong><span>最早</span><strong>{time(status.data?.oldest_message_at)}</strong><span>最新</span><strong>{time(status.data?.newest_message_at)}</strong>
      </div>
      {settings.data && <Space wrap><Switch checked={settings.data.enabled} onChange={enabled => save.mutate({ ...settings.data, enabled })}/><Select value={settings.data.retention_days} options={retentionOptions} onChange={retention_days => save.mutate({ ...settings.data, retention_days })}/></Space>}
    </section>
    <section className={styles.dataSection}>
      <Form layout="inline" onFinish={submit}>
        <Form.Item><RangePicker showTime value={range} onChange={value => value && setRange(value as [Dayjs, Dayjs])}/></Form.Item>
        <Form.Item name="server_key"><Input placeholder="server_key"/></Form.Item><Form.Item name="steam_id"><Input placeholder="SteamID64"/></Form.Item><Form.Item name="nickname"><Input placeholder="玩家名"/></Form.Item>
        <Form.Item name="map_name"><Input placeholder="地图"/></Form.Item><Form.Item name="game_mode"><Input placeholder="模式"/></Form.Item>
        <Form.Item name="team"><Select allowClear placeholder="队伍" style={{ width: 110 }} options={['survivor','infected','spectator'].map(value => ({ value }))}/></Form.Item>
        <Form.Item name="channel"><Select allowClear placeholder="频道" style={{ width: 110 }} options={['global','team'].map(value => ({ value }))}/></Form.Item>
        <Form.Item name="message_kind"><Select allowClear placeholder="类型" style={{ width: 120 }} options={[{value:'normal',label:'普通'},{value:'command',label:'命令样式'}]}/></Form.Item>
        <Form.Item name="keyword"><Input placeholder="内容关键词"/></Form.Item><Form.Item name="boot_id"><Input placeholder="Boot ID（高级）"/></Form.Item>
        <Form.Item><Button htmlType="submit" type="primary" icon={<SearchOutlined/>}>查询</Button></Form.Item>
      </Form>
      <Space><Button icon={<DownloadOutlined/>} loading={download.isPending} onClick={() => download.mutate({ format: 'csv', value: filter })}>CSV</Button><Button icon={<DownloadOutlined/>} loading={download.isPending} onClick={() => download.mutate({ format: 'jsonl', value: filter })}>JSONL</Button></Space>
      <Table<ChatMessage> columns={columns} dataSource={search.data?.items ?? []} rowKey="message_id" loading={search.isPending} pagination={false} scroll={{ x: 1100 }}/>
      {search.data?.next_cursor_id && <Button onClick={() => { const next = { ...filter, cursor_at: search.data?.next_cursor_at, cursor_id: search.data?.next_cursor_id }; setFilter(next); search.mutate(next) }}>下一页</Button>}
    </section>
  </>
}
