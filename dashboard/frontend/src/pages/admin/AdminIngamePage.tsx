import { CopyOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Card, Form, Input, Select, Switch, Typography, message } from 'antd'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, type IngameSettings } from '../../api'
import styles from './AdminIngamePage.module.scss'

const cachePresets = {
  home_cache_seconds: [10, 30, 60, 120],
  player_cache_seconds: [30, 60, 120, 300],
  ranking_cache_seconds: [60, 120, 300, 600],
  content_cache_seconds: [60, 300, 600, 1800],
} as const

function validHTTPURL(value?: string) {
  if (!value?.trim()) return true
  try {
    const url = new URL(value)
    return (url.protocol === 'http:' || url.protocol === 'https:') && !url.username && !url.password
  } catch { return false }
}

export function AdminIngamePage() {
  const { i18n } = useTranslation()
  const zh = !i18n.language.startsWith('en')
  const label = (cn: string, en: string) => zh ? cn : en
  const client = useQueryClient()
  const query = useQuery({ queryKey: ['admin-ingame'], queryFn: api.ingameSettings })
  const servers = useQuery({ queryKey: ['admin-servers'], queryFn: api.servers })
  const [selectedServer, setSelectedServer] = useState<string>()
  const deployment = useQuery({ queryKey: ['admin-server-ingame', selectedServer], queryFn: () => api.serverIngameSettings(selectedServer!), enabled: Boolean(selectedServer) })
  const [form] = Form.useForm<IngameSettings>()
  useEffect(() => { if (query.data) form.setFieldsValue(query.data.settings) }, [form, query.data])
  const save = useMutation({
    mutationFn: api.saveIngameSettings,
    onSuccess: data => {
      client.setQueryData(['admin-ingame'], data)
      form.setFieldsValue(data.settings)
      void message.success(label('游戏内页面设置已保存', 'In-game portal settings saved'))
    },
  })
  const metrics = query.data?.metric_catalog.map(metric => ({ value: metric.key, label: metric.label })) ?? []
  const serverKey = deployment.data?.server_key ?? ''
  const publicOrigin = deployment.data?.public_origin || query.data?.public_origin || ''
  const portalURL = publicOrigin && serverKey ? `${publicOrigin.replace(/\/$/, '')}/ingame?server=${encodeURIComponent(serverKey)}` : ''
  const motd = portalURL ? `<html>\n<head>\n<meta http-equiv="refresh"\ncontent="0;url=${portalURL}">\n</head>\n<body>\nLoading...\n</body>\n</html>` : ''
  const copyMotd = async () => {
    await navigator.clipboard.writeText(motd)
    void message.success(label('已复制 motd.txt 内容', 'motd.txt content copied'))
  }

  return <div className={styles.page}>
    <div className={styles.header}><div><Typography.Title level={2}>{label('游戏内页面', 'In-game portal')}</Typography.Title><Typography.Text type="secondary">{label('配置原生 L4D2 MOTD 浏览器使用的轻量页面。', 'Configure the lightweight portal used by the native L4D2 MOTD browser.')}</Typography.Text></div></div>
    {query.isError && <Alert type="error" showIcon title={query.error.message} />}
    <Form form={form} layout="vertical" onFinish={value => save.mutate(value)}>
      <Card title={label('默认外观', 'Default appearance')}>
        <div className={styles.switchRow}><div><strong>{label('启用游戏内页面', 'Enable in-game portal')}</strong><span>{label('关闭后所有 /ingame 页面返回不可用提示。', 'When disabled, all /ingame pages show an unavailable message.')}</span></div><Form.Item name="enabled" valuePropName="checked" noStyle><Switch /></Form.Item></div>
        <Form.Item name="title" label={label('默认标题', 'Default title')} rules={[{ max: 128 }]}><Input maxLength={128} /></Form.Item>
        <Form.Item name="description" label={label('默认描述', 'Default description')} rules={[{ max: 1000 }]}><Input.TextArea rows={3} maxLength={1000} showCount /></Form.Item>
        <Form.Item name="banner_url" label={label('默认 Banner URL', 'Default banner URL')} extra={label('Dashboard 不会主动访问或检查这个地址。', 'Dashboard never fetches or inspects this address.')} rules={[{ validator: (_, value) => validHTTPURL(value) ? Promise.resolve() : Promise.reject(new Error(label('请输入不含账号密码的 HTTP/HTTPS 地址', 'Enter a credential-free HTTP/HTTPS URL'))) }]}><Input placeholder="https://example.com/banner.jpg" /></Form.Item>
        <Form.Item name="website_url" label={label('默认完整网站 URL', 'Default full website URL')} rules={[{ validator: (_, value) => validHTTPURL(value) ? Promise.resolve() : Promise.reject(new Error(label('请输入不含账号密码的 HTTP/HTTPS 地址', 'Enter a credential-free HTTP/HTTPS URL'))) }]}><Input placeholder="https://stats.example.com" /></Form.Item>
      </Card>

      <Card title={label('首页模块', 'Modules')}>
        <div className={styles.switchGrid}>
          <Form.Item name="show_announcements" valuePropName="checked" label={label('显示最新公告', 'Show latest announcement')}><Switch /></Form.Item>
          <Form.Item name="show_players" valuePropName="checked" label={label('显示在线玩家', 'Show online players')}><Switch /></Form.Item>
          <Form.Item name="show_highlights" valuePropName="checked" label={label('显示生涯亮点', 'Show career highlights')}><Switch /></Form.Item>
        </div>
      </Card>

      <Card title={label('生涯亮点', 'Highlights')} extra={label('每项仅展示当前在线真人中的第一名。', 'Each metric shows only the top currently-online human player.')}>
        <div className={styles.threeColumns}>{[0, 1, 2].map(index => <Form.Item key={index} name={['highlight_metrics', index]} label={`${label('指标', 'Metric')} ${index + 1}`} rules={[{ required: true }]}><Select options={metrics} /></Form.Item>)}</div>
      </Card>

      <Card title={label('性能与缓存', 'Performance & cache')} extra={label('只允许安全预设，不能关闭缓存或填写任意秒数。', 'Only safe presets are available; cache cannot be disabled or customized.')}>
        <div className={styles.cacheGrid}>
          {(Object.keys(cachePresets) as Array<keyof typeof cachePresets>).map(key => {
            const titles = { home_cache_seconds: label('首页缓存', 'Home cache'), player_cache_seconds: label('玩家档案缓存', 'Player cache'), ranking_cache_seconds: label('排行榜缓存', 'Rankings cache'), content_cache_seconds: label('公告/文档缓存', 'Content cache') }
            return <Form.Item key={key} name={key} label={titles[key]} rules={[{ required: true }]}><Select options={cachePresets[key].map(value => ({ value, label: `${value} ${label('秒', 'seconds')}` }))} /></Form.Item>
          })}
        </div>
        <Alert type="info" showIcon title={label('A2S 刷新周期仍在“站点设置 → 服务”中配置。', 'The A2S refresh interval remains under Site settings → Services.')} />
      </Card>

      <Button type="primary" htmlType="submit" loading={save.isPending}>{label('保存', 'Save')}</Button>
    </Form>

    <Card className={styles.deployment} title={label('MOTD 部署帮助', 'MOTD deployment help')}>
      <Typography.Paragraph>{label('选择服务器后生成可复制的 motd.txt。Dashboard 和 Collector 不会写入游戏服务器文件；host.txt 只能使用普通文本。', 'Select a server to generate motd.txt. Dashboard and Collector never write game-server files; host.txt only supports plain text.')}</Typography.Paragraph>
      <Select className={styles.serverSelect} value={selectedServer} onChange={setSelectedServer} placeholder={label('选择服务器', 'Select a server')} options={servers.data?.filter(server => server.id).map(server => ({ value: server.id!, label: `${server.display_name} (${server.address})` }))} />
      {!publicOrigin && selectedServer && <Alert type="warning" showIcon title={label('请先在站点设置中配置公开地址 public_origin。', 'Configure public_origin in Site settings first.')} />}
      {publicOrigin && selectedServer && !deployment.isLoading && !serverKey && <Alert type="warning" showIcon title={label('尚未从该服务器的 A2S 规则中发现 sm_lps_server_key，请先确保 Collector 正常上报并刷新 A2S。', 'sm_lps_server_key has not been found in this server’s A2S rules. Ensure Collector is reporting and refresh A2S.')} />}
      {motd && <><pre className={styles.code}>{motd}</pre><Button icon={<CopyOutlined />} onClick={() => void copyMotd()}>{label('复制 motd.txt', 'Copy motd.txt')}</Button></>}
    </Card>
  </div>
}
