import { DeleteOutlined, EditOutlined, PlusOutlined, RightOutlined } from '@ant-design/icons'
import MDEditor from '@uiw/react-md-editor'
import '@uiw/react-md-editor/markdown-editor.css'
import '@uiw/react-markdown-preview/markdown.css'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Button, Empty, Form, Input, Modal, Pagination, Popconfirm, Spin, Typography, message } from 'antd'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, type Announcement, type AnnouncementInput } from '../api'
import { MarkdownContent } from '../components/MarkdownContent'
import styles from './AdminAnnouncementsPage.module.scss'

const formatDate = (value: number) => new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value * 1000))
const emptyValue: AnnouncementInput = { title: '', content_markdown: '' }

export function AdminAnnouncementsPage() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [page, setPage] = useState(1)
  const [editing, setEditing] = useState<Announcement | null | 'new'>(null)
  const [expandedID, setExpandedID] = useState<string | null>(null)
  const [markdown, setMarkdown] = useState('')
  const [form] = Form.useForm<AnnouncementInput>()
  const query = useQuery({ queryKey: ['admin-announcements', page], queryFn: () => api.adminAnnouncements(page, 20) })
  const refresh = () => Promise.all([
    client.invalidateQueries({ queryKey: ['admin-announcements'] }),
    client.invalidateQueries({ queryKey: ['announcements'] }),
  ])
  const save = useMutation({
    mutationFn: (value: AnnouncementInput) => editing === 'new' ? api.createAnnouncement(value) : api.updateAnnouncement((editing as Announcement).id, value),
    onSuccess: () => { setEditing(null); void refresh(); void message.success(t('saved')) },
  })
  const remove = useMutation({ mutationFn: api.deleteAnnouncement, onSuccess: () => { void refresh(); void message.success(t('announcementDeleted')) } })
  const openCreate = () => { form.setFieldsValue(emptyValue); setMarkdown(''); setEditing('new') }
  const openEdit = (value: Announcement) => { form.setFieldsValue({ title: value.title, content_markdown: value.content_markdown }); setMarkdown(value.content_markdown); setEditing(value) }
  const submit = (value: AnnouncementInput) => {
    const content = markdown.trim()
    if (!content) { void message.error(t('announcementContentRequired')); return }
    save.mutate({ title: value.title.trim(), content_markdown: content })
  }

  return <div className={styles.page}>
    <div className={styles.toolbar}><div><Typography.Title level={2}>{t('announcementManagement')}</Typography.Title><Typography.Text type="secondary">{t('announcementManagementHint')}</Typography.Text></div><Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>{t('addAnnouncement')}</Button></div>
    {query.isLoading && <div className={styles.state}><Spin /></div>}
    {!query.isLoading && query.data?.items.length === 0 && <div className={styles.state}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('noAnnouncements')} /></div>}
    <div className={styles.list}>{query.data?.items.map(item => {
      const expanded = expandedID === item.id
      return <article className={`${styles.item} ${expanded ? styles.expanded : ''}`} key={item.id}>
      <div className={styles.itemRow}>
        <button className={styles.itemToggle} type="button" aria-expanded={expanded} onClick={() => setExpandedID(expanded ? null : item.id)}>
          <span><h3>{item.title}</h3><time>{formatDate(item.updated_at)}</time></span>
          <RightOutlined className={styles.expandIcon} aria-hidden="true" />
        </button>
        <div className={styles.actions}>
        <Button type="text" icon={<EditOutlined />} aria-label={t('editAnnouncement')} title={t('editAnnouncement')} onClick={() => openEdit(item)} />
        <Popconfirm title={t('confirmDeleteAnnouncement')} okText={t('deleteAnnouncement')} cancelText={t('cancel')} onConfirm={() => remove.mutate(item.id)}><Button danger type="text" icon={<DeleteOutlined />} aria-label={t('deleteAnnouncement')} title={t('deleteAnnouncement')} /></Popconfirm>
      </div></div>
      <div className={`${styles.previewRegion} ${expanded ? styles.open : ''}`}><div className={styles.previewInner}>
        {expanded && <div className={styles.preview}><MarkdownContent source={item.content_markdown} /></div>}
      </div></div>
    </article>})}</div>
    {(query.data?.total ?? 0) > 20 && <Pagination className={styles.pagination} current={page} pageSize={20} total={query.data?.total ?? 0} showSizeChanger={false} onChange={setPage} />}
    <Modal className={styles.editorModal} width={960} open={editing !== null} title={editing === 'new' ? t('addAnnouncement') : t('editAnnouncement')} onCancel={() => setEditing(null)} onOk={() => form.submit()} confirmLoading={save.isPending} destroyOnHidden>
      <Form form={form} layout="vertical" onFinish={submit}>
        <Form.Item name="title" label={t('announcementTitle')} rules={[{ required: true, message: t('announcementTitleRequired') }, { max: 120, message: t('announcementTitleLength') }]}><Input maxLength={120} showCount /></Form.Item>
        <Form.Item label={t('announcementContent')} required>
          <div className={styles.editorWorkspace}>
            <MDEditor className={styles.editor} height={430} value={markdown} onChange={value => setMarkdown(value ?? '')} preview="live" visibleDragbar={false} />
          </div>
        </Form.Item>
      </Form>
    </Modal>
  </div>
}
