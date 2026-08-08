import { DatabaseOutlined, DeleteOutlined, ReloadOutlined, SyncOutlined, UserOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Spin, Typography, message } from 'antd'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, type DataMaintenanceSettings } from '../api'
import { PlayerPreviewModal } from '../components/PlayerPreviewModal'
import styles from './Portal.module.scss'

const intervals = [15, 30, 60, 180, 300, 720, 1440]

function bytes(value = 0) {
  if (value < 1024) return `${value} B`
  if (value < 1024 ** 2) return `${(value / 1024).toFixed(1)} KiB`
  if (value < 1024 ** 3) return `${(value / 1024 ** 2).toFixed(1)} MiB`
  return `${(value / 1024 ** 3).toFixed(2)} GiB`
}

function dateTime(value = 0) {
  return value > 0 ? new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'medium' }).format(new Date(value * 1000)) : '—'
}

export function AdminDataPage() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [form] = Form.useForm<DataMaintenanceSettings>()
  const [previewPromptOpen, setPreviewPromptOpen] = useState(false)
  const [previewInput, setPreviewInput] = useState('')
  const [previewSteamID, setPreviewSteamID] = useState('')
  const status = useQuery({ queryKey: ['admin-data-status'], queryFn: api.dataStatus })
  useEffect(() => { if (status.data?.settings) form.setFieldsValue(status.data.settings) }, [form, status.data?.settings])
  const refresh = () => void client.invalidateQueries({ queryKey: ['admin-data-status'] })
  const save = useMutation({ mutationFn: api.saveDataSettings, onSuccess: () => { void message.success(t('saved')); refresh() } })
  const aggregate = useMutation({ mutationFn: api.aggregateNow, onSuccess: () => { void message.success(t('aggregateCompleted')); refresh() }, onError: () => void message.error(t('operationFailed')) })
  const cleanup = useMutation({ mutationFn: (planID: string) => api.applyRetention(planID), onSuccess: result => { void message.success(t('cleanupCompleted', { count: result.equipment_rows + result.versus_class_rows + result.session_rows + result.versus_round_result_rows + result.versus_run_result_rows })); refresh() }, onError: () => { void message.error(t('cleanupPreviewChanged')); refresh() } })
  const openPlayerPreview = () => {
    const steamID = previewInput.trim()
    if (!/^7656119\d{10}$/.test(steamID)) return
    setPreviewPromptOpen(false)
    setPreviewSteamID(steamID)
  }

  if (status.isLoading) return <div className={styles.adminPage}><Spin /></div>
  if (!status.data) return <div className={styles.adminPage}><Alert type="error" message={t('dataStatusUnavailable')} action={<Button onClick={() => status.refetch()}>{t('retry')}</Button>} /></div>
  const data = status.data
  const plan = data.retention_plan
  const cleanupRows = plan.equipment_rows_eligible + plan.versus_class_rows_eligible + plan.session_rows_eligible + plan.versus_round_results_eligible + plan.versus_run_results_eligible

  return <div className={styles.adminPage}>
    <div className={styles.toolbar}>
      <div><Typography.Title level={2}>{t('dataMaintenance')}</Typography.Title><Typography.Text type="secondary">{t('dataMaintenanceHint')}</Typography.Text></div>
      <Space>
        <Button icon={<UserOutlined />} onClick={() => setPreviewPromptOpen(true)}>{t('previewPlayerCard')}</Button>
        <Button icon={<ReloadOutlined />} onClick={() => status.refetch()}>{t('refresh')}</Button>
      </Space>
    </div>

    <section className={styles.dataUsageGrid}>
      <div><DatabaseOutlined /><span>{t('statsDatabase')}</span><strong>{bytes(data.stats_database.bytes + (data.stats_database.wal_bytes ?? 0))}</strong></div>
      <div><DatabaseOutlined /><span>{t('dashboardDatabase')}</span><strong>{bytes(data.dashboard_database.bytes + (data.dashboard_database.wal_bytes ?? 0))}</strong></div>
      <div><DatabaseOutlined /><span>{t('rotatedLogs')}</span><strong>{bytes(data.log_bytes)}</strong></div>
      <div><SyncOutlined /><span>{t('aggregateRows')}</span><strong>{data.aggregate.aggregate_rows.toLocaleString()}</strong></div>
    </section>

    <section className={styles.dataSection}>
      <div className={styles.sectionTitleRow}><div className={styles.formSectionHeading}><strong>{t('incrementalAggregate')}</strong><span>{t('incrementalAggregateHint')}</span></div><Button icon={<SyncOutlined />} loading={aggregate.isPending} onClick={() => aggregate.mutate()}>{t('aggregateNow')}</Button></div>
      <div className={styles.dataStatusRows}>
        <span>{t('aggregateState')}</span><strong>{data.aggregate.state}</strong>
        <span>{t('lastAggregate')}</span><strong>{dateTime(data.aggregate.last_finished_at)}</strong>
        <span>{t('aggregateDuration')}</span><strong>{data.aggregate.last_duration_ms.toLocaleString()} ms</strong>
        <span>{t('changedDays')}</span><strong>{data.aggregate.last_changed_days}</strong>
      </div>
      {data.aggregate.last_error && <Alert type="warning" showIcon message={data.aggregate.last_error} />}
    </section>

    <Form form={form} layout="vertical" onFinish={value => save.mutate(value)}>
      <section className={styles.dataSection}>
        <div className={styles.formSectionHeading}><strong>{t('maintenancePolicy')}</strong><span>{t('maintenancePolicyHint')}</span></div>
        <div className={styles.dataSettingsGrid}>
          <Form.Item name="aggregate_interval_minutes" label={t('aggregateInterval')}><Select options={intervals.map(value => ({ value, label: t('minutesValue', { value }) }))} /></Form.Item>
          <Form.Item name="detail_retention_days" label={t('detailRetention')} rules={[{ required: true }]}><InputNumber min={30} max={3650} addonAfter={t('days')} /></Form.Item>
          <Form.Item name="session_retention_days" label={t('sessionRetention')} rules={[{ required: true }]}><InputNumber min={30} max={3650} addonAfter={t('days')} /></Form.Item>
          <Form.Item name="result_retention_days" label={t('resultRetention')} rules={[{ required: true }]}><InputNumber min={30} max={3650} addonAfter={t('days')} /></Form.Item>
        </div>
        <Button type="primary" htmlType="submit" loading={save.isPending}>{t('save')}</Button>
      </section>
    </Form>

    <section className={styles.dataSection}>
      <div className={styles.sectionTitleRow}><div className={styles.formSectionHeading}><strong>{t('rawDataCleanup')}</strong><span>{t('rawDataCleanupHint')}</span></div></div>
      <div className={styles.dataStatusRows}>
        <span>{t('equipmentDetailRows')}</span><strong>{plan.equipment_rows_eligible.toLocaleString()}</strong>
        <span>{t('classDetailRows')}</span><strong>{plan.versus_class_rows_eligible.toLocaleString()}</strong>
        <span>{t('closedSessionRows')}</span><strong>{plan.session_rows_eligible.toLocaleString()}</strong>
        <span>{t('matchResultRows')}</span><strong>{(plan.versus_round_results_eligible + plan.versus_run_results_eligible).toLocaleString()}</strong>
      </div>
      {!plan.aggregate_coverage_ready && <Alert type="warning" showIcon message={t('aggregateCoverageNotReady')} />}
      <Popconfirm title={t('confirmRawCleanup')} description={t('confirmRawCleanupHint', { count: cleanupRows })} okButtonProps={{ danger: true }} okText={t('confirmCleanup')} cancelText={t('cancel')} onConfirm={() => cleanup.mutate(plan.plan_id)} disabled={!plan.deletion_enabled || cleanupRows === 0}>
        <Button danger icon={<DeleteOutlined />} loading={cleanup.isPending} disabled={!plan.deletion_enabled || cleanupRows === 0}>{t('cleanupEligibleRows', { count: cleanupRows })}</Button>
      </Popconfirm>
      <Typography.Text type="secondary" className={styles.cleanupAudit}>{t('cleanupRuns', { count: data.retention_runs })}</Typography.Text>
    </section>

    <Modal title={t('previewPlayerCard')} open={previewPromptOpen} okText={t('preview')} cancelText={t('cancel')}
      okButtonProps={{ disabled: !/^7656119\d{10}$/.test(previewInput.trim()) }}
      onOk={openPlayerPreview} onCancel={() => setPreviewPromptOpen(false)} destroyOnHidden>
      <Typography.Paragraph type="secondary">{t('previewPlayerCardHint')}</Typography.Paragraph>
      <Input autoFocus value={previewInput} maxLength={17} placeholder="SteamID64" onChange={event => setPreviewInput(event.target.value)} onPressEnter={openPlayerPreview} />
    </Modal>
    <PlayerPreviewModal open={previewSteamID !== ''} steamID={previewSteamID} contextLabel={t('statsDatabase')}
      onClose={() => setPreviewSteamID('')} />
  </div>
}
