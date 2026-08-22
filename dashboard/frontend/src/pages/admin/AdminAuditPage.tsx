import { DownloadOutlined, FilterOutlined, KeyOutlined, LeftOutlined, ReloadOutlined, RightOutlined, SearchOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, DatePicker, Form, Input, Modal, Select, Space, Spin, Switch, Table, Tabs, Typography, message } from 'antd'
import dayjs, { type Dayjs } from 'dayjs'
import { useEffect, useState } from 'react'
import { api, type ChatMessage, type ChatRetentionPlan, type ChatSearchFilter, type ConnectionAuditFilter, type ConnectionAuditRow } from '../../api'
import { FloatingToolbar } from '../../components/FloatingToolbar'
import styles from '../Portal.module.scss'

const { RangePicker } = DatePicker
const retentionOptions = [7, 14, 30, 60, 90, 180, 365, 0].map(value => ({ value, label: value === 0 ? '永久' : `${value} 天` }))
const qpsOptions = [1, 2, 3].map(value => ({ value, label: `${value} QPS` }))
const initialRange = (): [Dayjs, Dayjs] => [dayjs().subtract(24, 'hour'), dayjs()]
const time = (value = 0) => value ? new Date(value * 1000).toLocaleString() : '-'
const duration = (seconds: number) => seconds < 60 ? `${seconds}s` : `${Math.floor(seconds / 60)}m`
interface AuditCursor { cursor_at?: number; cursor_id?: string }

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
  const [filterOpen, setFilterOpen] = useState(false)
  const [keyOpen, setKeyOpen] = useState(false)
  const [page, setPage] = useState(0)
  const [cursors, setCursors] = useState<AuditCursor[]>([{}])
  const [filterForm] = Form.useForm<ConnectionAuditFilter>()
  const geo = useQuery({ queryKey: ['admin-geoip-settings'], queryFn: api.geoIPSettings })
  const search = useMutation({ mutationFn: api.searchConnections })
  const saveGeo = useMutation({
    mutationFn: api.saveGeoIPSettings,
    onSuccess: () => {
      void message.success('GeoIP 设置已保存')
      void client.invalidateQueries({ queryKey: ['admin-geoip-settings'] })
      setKeyOpen(false)
      setNewKey('')
    },
    onError: error => void message.error(error instanceof Error ? error.message : '保存失败'),
  })
  const testGeo = useMutation({ mutationFn: api.testGeoIP, onSuccess: result => void message.success(`测试成功：${[result.country, result.province, result.city].filter(Boolean).join(' ')}`), onError: error => void message.error(error instanceof Error ? error.message : '测试失败') })
  const [newKey, setNewKey] = useState('')
  const [qpsLimit, setQPSLimit] = useState(2)
  const [testIP, setTestIP] = useState('')
  useEffect(() => { search.mutate(filter) }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const submit = (values: ConnectionAuditFilter) => {
    const next: ConnectionAuditFilter = { ...values, from: range[0].unix(), to: range[1].unix(), limit: 100 }
    setFilter(next)
    setCursors([{}])
    setPage(0)
    search.mutate(next)
    setFilterOpen(false)
  }
  const goToPage = (nextPage: number, cursor: AuditCursor) => {
    setPage(nextPage)
    search.mutate({ ...filter, ...cursor })
  }
  const nextPage = () => {
    if (!search.data?.next_cursor_id) return
    const cursor = { cursor_at: search.data.next_cursor_at, cursor_id: search.data.next_cursor_id }
    setCursors(current => [...current.slice(0, page + 1), cursor])
    goToPage(page + 1, cursor)
  }
  const previousPage = () => page > 0 && goToPage(page - 1, cursors[page - 1] ?? {})
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
    <FloatingToolbar ariaLabel="连接审计工具" items={[
      { key: 'key', label: '密钥设置', icon: <KeyOutlined/>, active: keyOpen, onClick: () => { setQPSLimit(geo.data?.qps_limit ?? 2); setKeyOpen(true) } },
      { key: 'filter', label: '筛选连接记录', icon: <FilterOutlined/>, active: filterOpen, onClick: () => setFilterOpen(true) },
    ]}/>
    <section className={styles.dataSection}>
      <div className={styles.formSectionHeading}><strong>GeoIP 状态</strong><span>配置 AK 后自动解析；清除 AK 即停止新请求。Dashboard 只缓存 IP 的 HMAC 与城市级近似结果。</span></div>
      {geo.isLoading ? <Spin size="small"/> : <div className={`${styles.dataStatusRows} ${styles.threePairRows}`}>
        <span>Provider</span><strong>Baidu</strong><span>AK</span><strong>{geo.data?.api_key_configured ? geo.data.api_key_masked : '未配置'}</strong><span>限速</span><strong>{geo.data?.qps_limit ?? 2} QPS</strong>
        <span>缓存</span><strong>{geo.data?.cache_count ?? 0}</strong><span>待处理</span><strong>{geo.data?.pending_count ?? 0}</strong><span>IPv4 / IPv6</span><strong>{geo.data?.ipv4_status ?? 'unknown'} / {geo.data?.ipv6_status ?? 'unknown'}</strong>
        <span>上次成功</span><strong>{time(geo.data?.last_success_at)}</strong><span>最近错误</span><strong>{time(geo.data?.last_error_at)}</strong><span>错误状态</span><strong>{geo.data?.last_error_code || '-'}</strong>
      </div>}
      {geo.data?.last_error_code && <Alert type="warning" showIcon message={`最近错误：${geo.data.last_error_code}`} />}
    </section>
    <section className={styles.dataSection}>
      {search.data?.location_pending && <Alert type="info" showIcon message="部分位置仍在解析，请稍后刷新" />}
      <Table<ConnectionAuditRow> columns={columns} dataSource={search.data?.items ?? []} rowKey="session_id" loading={search.isPending} pagination={false} scroll={{ x: 1050 }} />
      <AuditPager page={page} loading={search.isPending} hasNext={Boolean(search.data?.next_cursor_id)} onPrevious={previousPage} onNext={nextPage}/>
    </section>
    <Modal title="GeoIP 密钥与限速" open={keyOpen} footer={null} onCancel={() => setKeyOpen(false)}>
      <Form layout="vertical">
        <Form.Item label="新 Baidu AK" extra="留空保留现有密钥；密钥存在时自动启用 GeoIP。"><Input.Password value={newKey} onChange={event => setNewKey(event.target.value)} placeholder={geo.data?.api_key_configured ? '留空保留现有密钥' : '请输入 Baidu AK'}/></Form.Item>
        <Form.Item label="百度请求限速"><Select value={qpsLimit} options={qpsOptions} onChange={setQPSLimit}/></Form.Item>
        <Form.Item label="公网测试 IP"><Space.Compact block><Input value={testIP} onChange={event => setTestIP(event.target.value)} placeholder="8.8.8.8"/><Button loading={testGeo.isPending} disabled={!geo.data?.api_key_configured || !testIP.trim()} onClick={() => testGeo.mutate(testIP.trim())}>测试</Button></Space.Compact></Form.Item>
        <div className={styles.sectionTitleRow}>
          <Button danger disabled={!geo.data?.api_key_configured || saveGeo.isPending} onClick={() => Modal.confirm({ title: '确认清除 Baidu AK？', content: '清除后停止所有新 GeoIP 请求，已有 HMAC 缓存保留到过期。', okButtonProps: { danger: true }, onOk: async () => { await saveGeo.mutateAsync({ clear_api_key: true, qps_limit: qpsLimit }) } })}>清除密钥</Button>
          <Button type="primary" loading={saveGeo.isPending} onClick={() => saveGeo.mutate({ api_key: newKey, qps_limit: qpsLimit })}>保存</Button>
        </div>
      </Form>
    </Modal>
    <Modal title="筛选连接记录" open={filterOpen} footer={null} onCancel={() => setFilterOpen(false)}>
      <Form<ConnectionAuditFilter> form={filterForm} layout="vertical" onFinish={submit}>
        <Form.Item label="时间范围"><RangePicker showTime value={range} onChange={value => value && setRange(value as [Dayjs, Dayjs])} style={{ width: '100%' }}/></Form.Item>
        <Form.Item name="server_key" label="Server Key"><Input/></Form.Item><Form.Item name="steam_id" label="SteamID64"><Input/></Form.Item>
        <Form.Item name="nickname" label="玩家名"><Input/></Form.Item><Form.Item name="ip_address" label="IP"><Input/></Form.Item><Form.Item name="location" label="国家 / 省 / 市"><Input/></Form.Item>
        <Button htmlType="submit" type="primary" icon={<SearchOutlined/>} block>应用筛选</Button>
      </Form>
    </Modal>
  </>
}

function ChatAudit() {
  const client = useQueryClient()
  const settings = useQuery({ queryKey: ['admin-chat-settings'], queryFn: api.chatAuditSettings })
  const status = useQuery({ queryKey: ['admin-chat-status'], queryFn: api.chatAuditStatus })
  const [range, setRange] = useState<[Dayjs, Dayjs]>(initialRange)
  const [filter, setFilter] = useState<ChatSearchFilter>({ from: dayjs().subtract(24, 'hour').unix(), to: dayjs().unix(), limit: 100 })
  const [filterOpen, setFilterOpen] = useState(false)
  const [page, setPage] = useState(0)
  const [cursors, setCursors] = useState<AuditCursor[]>([{}])
  const [filterForm] = Form.useForm<ChatSearchFilter>()
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
    const result = await api.confirmChatAuditSettings(plan.plan_id, { ...settings.data, retention_days: plan.retention_days })
    if (result.cleanup_status === 'pending') void message.warning('保留策略已保存，清理尚未完成；后台将按新策略继续清理')
    else void message.success(`聊天审计设置已保存，已清理 ${result.deleted.toLocaleString()} 条记录`)
    void client.invalidateQueries({ queryKey: ['admin-chat-settings'] }); void client.invalidateQueries({ queryKey: ['admin-chat-status'] })
  } })
  const submit = (values: ChatSearchFilter) => {
    const next: ChatSearchFilter = { ...values, from: range[0].unix(), to: range[1].unix(), limit: 100 }
    setFilter(next)
    setCursors([{}])
    setPage(0)
    search.mutate(next)
    setFilterOpen(false)
  }
  const goToPage = (nextPage: number, cursor: AuditCursor) => {
    setPage(nextPage)
    search.mutate({ ...filter, ...cursor })
  }
  const nextPage = () => {
    if (!search.data?.next_cursor_id) return
    const cursor = { cursor_at: search.data.next_cursor_at, cursor_id: search.data.next_cursor_id }
    setCursors(current => [...current.slice(0, page + 1), cursor])
    goToPage(page + 1, cursor)
  }
  const previousPage = () => page > 0 && goToPage(page - 1, cursors[page - 1] ?? {})
  const columns = [
    { title: '时间', dataIndex: 'occurred_at', render: time }, { title: '服务器', dataIndex: 'server_key' },
    { title: '玩家', key: 'player', render: (_: unknown, row: ChatMessage) => <><strong>{row.player_name}</strong><br/><Typography.Text type="secondary">{row.steam_id || `userid:${row.source_user_id}`}</Typography.Text></> },
    { title: '环境', key: 'context', render: (_: unknown, row: ChatMessage) => `${row.map_name || '-'} · ${row.game_mode || '-'} · ${row.team}` },
    { title: '频道', key: 'channel', render: (_: unknown, row: ChatMessage) => `${row.channel}${row.command_like ? ' · 命令样式' : ''}` },
    { title: '内容', dataIndex: 'content' },
  ]
  return <>
    <FloatingToolbar ariaLabel="聊天审计工具" items={[{ key: 'filter', label: '筛选聊天记录', icon: <FilterOutlined/>, active: filterOpen, onClick: () => setFilterOpen(true) }]}/>
    <section className={styles.dataSection}>
      <div className={styles.auditControlRow}>
        <Space wrap>
          <Typography.Text>聊天审计</Typography.Text>
          {settings.data && <Switch checked={settings.data.enabled} onChange={enabled => save.mutate({ ...settings.data, enabled })} checkedChildren="开启" unCheckedChildren="关闭"/>}
          {settings.data && <Select
            value={settings.data.retention_days}
            options={retentionOptions}
            onChange={retention_days => save.mutate({ ...settings.data, retention_days })}
            style={{ width: 110 }}
          />}
        </Space>
        <Space wrap>
          <Button icon={<DownloadOutlined/>} loading={download.isPending} onClick={() => download.mutate({ format: 'csv', value: filter })}>CSV</Button>
          <Button icon={<DownloadOutlined/>} loading={download.isPending} onClick={() => download.mutate({ format: 'jsonl', value: filter })}>JSONL</Button>
          <Button icon={<ReloadOutlined/>} onClick={() => { void settings.refetch(); void status.refetch(); search.mutate({ ...filter, ...(cursors[page] ?? {}) }) }}>刷新</Button>
        </Space>
      </div>
      <div className={`${styles.dataStatusRows} ${styles.threePairRows}`}>
        <span>消息</span><strong>{status.data?.message_count.toLocaleString() ?? '-'}</strong><span>摄取延迟</span><strong>{status.data?.ingestion_lag ?? '-'}</strong><span>已知缺口</span><strong>{status.data?.known_gap_count ?? '-'}</strong>
        <span>Collector 丢弃</span><strong>{status.data?.dropped_count ?? '-'}</strong><span>最早</span><strong>{time(status.data?.oldest_message_at)}</strong><span>最新</span><strong>{time(status.data?.newest_message_at)}</strong>
      </div>
    </section>
    <section className={styles.dataSection}>
      <Table<ChatMessage> columns={columns} dataSource={search.data?.items ?? []} rowKey="message_id" loading={search.isPending} pagination={false} scroll={{ x: 1100 }}/>
      <AuditPager page={page} loading={search.isPending} hasNext={Boolean(search.data?.next_cursor_id)} onPrevious={previousPage} onNext={nextPage}/>
    </section>
    <Modal title="筛选聊天记录" open={filterOpen} footer={null} onCancel={() => setFilterOpen(false)}>
      <Form<ChatSearchFilter> form={filterForm} layout="vertical" onFinish={submit}>
        <Form.Item label="时间范围"><RangePicker showTime value={range} onChange={value => value && setRange(value as [Dayjs, Dayjs])} style={{ width: '100%' }}/></Form.Item>
        <Form.Item name="server_key" label="Server Key"><Input/></Form.Item><Form.Item name="steam_id" label="SteamID64"><Input/></Form.Item><Form.Item name="nickname" label="玩家名"><Input/></Form.Item>
        <Form.Item name="map_name" label="地图"><Input/></Form.Item><Form.Item name="game_mode" label="模式"><Input/></Form.Item>
        <Form.Item name="team" label="队伍"><Select allowClear options={['survivor', 'infected', 'spectator'].map(value => ({ value }))}/></Form.Item>
        <Form.Item name="channel" label="频道"><Select allowClear options={['global', 'team'].map(value => ({ value }))}/></Form.Item>
        <Form.Item name="message_kind" label="类型"><Select allowClear options={[{ value: 'normal', label: '普通' }, { value: 'command', label: '命令样式' }]}/></Form.Item>
        <Form.Item name="keyword" label="内容关键词"><Input/></Form.Item><Form.Item name="boot_id" label="Boot ID（高级）"><Input/></Form.Item>
        <Button htmlType="submit" type="primary" icon={<SearchOutlined/>} block>应用筛选</Button>
      </Form>
    </Modal>
  </>
}

function AuditPager({ page, loading, hasNext, onPrevious, onNext }: { page: number; loading: boolean; hasNext: boolean; onPrevious: () => void; onNext: () => void }) {
  return <div className={styles.auditPager}>
    <Button icon={<LeftOutlined/>} disabled={loading || page === 0} onClick={onPrevious}>上一页</Button>
    <Typography.Text type="secondary">第 {page + 1} 页</Typography.Text>
    <Button icon={<RightOutlined/>} disabled={loading || !hasNext} onClick={onNext}>下一页</Button>
  </div>
}
