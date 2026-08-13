import { BgColorsOutlined, CloudOutlined, DeleteOutlined, EditOutlined, HolderOutlined, PlusOutlined } from '@ant-design/icons'
import MDEditor from '@uiw/react-md-editor'
import '@uiw/react-md-editor/markdown-editor.css'
import '@uiw/react-markdown-preview/markdown.css'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Form, Input, Modal, Popconfirm, Select, Switch, Typography, message } from 'antd'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { api, type FooterLink, type SiteDocument, type SiteDocumentKey, type SiteSettings } from '../../api'
import { FloatingToolbar } from '../../components/FloatingToolbar'
import i18n from '../../i18n'
import styles from '../Portal.module.scss'

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

function validProxyURL(value?: string) {
  if (!value?.trim()) return true
  try {
    const candidate = value.includes('://') ? value : `http://${value}`
    const url = new URL(candidate)
    return ['http:', 'https:', 'socks5:', 'socks5h:'].includes(url.protocol) && !!url.hostname &&
      (url.pathname === '/' || url.pathname === '') && !url.search && !url.hash
  } catch { return false }
}

export function AdminSitePage() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()
  const activeSection = searchParams.get('section') === 'services' ? 'services' : 'appearance'
  const query = useQuery({ queryKey: ['admin-site'], queryFn: api.adminSite })
  const documents = useQuery({ queryKey: ['admin-site-documents'], queryFn: api.adminSiteDocuments })
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
  const [documentEditor, setDocumentEditor] = useState<SiteDocument | null>(null)
  const [documentMarkdown, setDocumentMarkdown] = useState('')
  const saveDocument = useMutation({
    mutationFn: api.saveSiteDocument,
    onSuccess: data => {
      client.setQueryData<SiteDocument[]>(['admin-site-documents'], current => current?.map(item => item.key === data.key ? data : item))
      void client.invalidateQueries({ queryKey: ['site'] })
      setDocumentEditor(null)
      void message.success(t('saved'))
    },
  })
  useEffect(() => {
    if (!query.data) return
    form.setFieldsValue(query.data)
  }, [form, query.data])
  const activeFooterLinks = footerLinks ?? query.data?.footer_links ?? []
  const origin = Form.useWatch('public_origin', form)
  const steamLoginEnabled = Form.useWatch('steam_openid_enabled', form)
  const footerEnabled = Form.useWatch('footer_enabled', form)
  const seoEnabled = Form.useWatch('seo_enabled', form)
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
  const documentLabel = (key: SiteDocumentKey) => t(key === 'introduction' ? 'serverIntroduction' : key === 'commands' ? 'serverCommands' : 'serverResources')
  const openDocumentEditor = (document: SiteDocument) => { setDocumentEditor(document); setDocumentMarkdown(document.content_markdown) }
  const toggleDocument = (document: SiteDocument, enabled: boolean) => {
    if (enabled && !document.content_markdown.trim()) { openDocumentEditor({ ...document, enabled: true }); return }
    saveDocument.mutate({ ...document, enabled })
  }
  const toolbarItems = [
    { key: 'appearance', label: t('siteAppearance'), icon: <BgColorsOutlined />, active: activeSection === 'appearance', onClick: () => setSearchParams({ section: 'appearance' }) },
    { key: 'services', label: t('siteServices'), icon: <CloudOutlined />, active: activeSection === 'services', onClick: () => setSearchParams({ section: 'services' }) },
  ]
  const saveCurrentSection = async () => {
    const fields: Array<keyof SiteSettings> = activeSection === 'appearance'
      ? ['language', 'theme', 'browser_title', 'background_image_url', 'footer_enabled']
      : ['a2s_refresh_seconds', 'a2s_jitter_seconds', 'a2s_retry_count', 'steam_openid_enabled', 'steam_openid_proxy_url', 'seo_enabled']
    if (activeSection === 'services' && (steamLoginEnabled || seoEnabled)) fields.push('public_origin')
    if (activeSection === 'services' && seoEnabled) fields.push('seo_description', 'seo_image_url')
    try {
      await form.validateFields(fields)
      save.mutate({ ...(form.getFieldsValue(true) as SiteSettings), footer_links: activeFooterLinks })
    } catch {
      // Ant Design already displays the validation messages for the current section.
    }
  }

  return <div className={`${styles.adminPage} ${styles.siteSettingsPage}`}>
    <FloatingToolbar ariaLabel={t('siteSettings')} items={toolbarItems} />
    <div className={styles.pageHeader}><Typography.Title level={2}>{t('siteSettings')}</Typography.Title></div>
    {save.isError && <Alert className={styles.inlineAlert} type="error" showIcon title={save.error.message} />}
    <Form form={form} layout="vertical" className={styles.settingsForm}>
      {activeSection === 'appearance' && <>
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
        <div className={styles.formSectionHeading}><strong>{t('homeTools')}</strong><span>{t('homeToolsHint')}</span></div>
        <div className={styles.siteDocumentList}>{documents.data?.map(document => <div className={styles.siteDocumentRow} key={document.key}>
          <div><strong>{documentLabel(document.key)}</strong><span>{document.content_markdown ? `${document.content_markdown.length} ${i18n.language === 'en' ? 'characters' : '字符'}` : '—'}</span></div>
          <div className={styles.sectionActions}>
            <Button type="text" icon={<EditOutlined />} aria-label={t('editContent')} title={t('editContent')} onClick={() => openDocumentEditor(document)} />
            <Switch checked={document.enabled} loading={saveDocument.isPending} onChange={checked => toggleDocument(document, checked)} />
          </div>
        </div>)}</div>
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
      </>}
      {activeSection === 'services' && <>
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
          <strong>{t('steamOpenID')}</strong>
          <Form.Item name="steam_openid_enabled" valuePropName="checked" noStyle><Switch /></Form.Item>
        </div>
        {(steamLoginEnabled || seoEnabled) && <>
          <Form.Item name="public_origin" label={t('publicOrigin')} extra={t('publicOriginHint')} rules={[{ required: true, message: t('requiredField') }, { validator: (_, value) => validPublicOrigin(value) ? Promise.resolve() : Promise.reject(new Error(t('invalidOrigin'))) }]}><Input placeholder={window.location.origin} /></Form.Item>
          {String(origin ?? '').startsWith('http://') && <Alert type="warning" showIcon title={t('insecureOrigin')} />}
        </>}
        {steamLoginEnabled && <Form.Item name="steam_openid_proxy_url" label={t('steamProxyURL')} extra={t('steamProxyURLHint')}
          rules={[{ max: 2048, message: t('invalidProxyURL') }, { validator: (_, value) => validProxyURL(value) ? Promise.resolve() : Promise.reject(new Error(t('invalidProxyURL'))) }]}>
          <Input placeholder="http://127.0.0.1:7890" />
        </Form.Item>}
      </section>
      <section className={styles.formSection}>
        <div className={styles.sectionTitleRow}>
          <div className={styles.formSectionHeading}><strong>{t('seoSettings')}</strong><span>{t('seoSettingsHint')}</span></div>
          <Form.Item name="seo_enabled" valuePropName="checked" noStyle><Switch /></Form.Item>
        </div>
        {seoEnabled && <>
          <Form.Item name="seo_description" label={t('seoDescription')} extra={t('seoDescriptionHint')} rules={[{ required: true, message: t('seoDescriptionRequired') }, { max: 200, message: t('seoDescriptionLength') }]}>
            <Input.TextArea rows={4} maxLength={200} showCount />
          </Form.Item>
          <Form.Item name="seo_image_url" label={t('seoImage')} extra={t('seoImageHint')} rules={[{ validator: (_, value) => validHTTPURL(value) ? Promise.resolve() : Promise.reject(new Error(t('invalidURL'))) }]}>
            <Input placeholder="https://example.com/share.jpg" />
          </Form.Item>
        </>}
      </section>
      </>}
      <Button type="primary" loading={save.isPending} onClick={() => void saveCurrentSection()}>{t('save')}</Button>
    </Form>
    <Modal open={footerEditor !== null} title={footerEditor?.index === null ? t('addLink') : t('editLink')}
      onCancel={() => setFooterEditor(null)} onOk={() => footerEditorForm.submit()} destroyOnHidden>
      <Form form={footerEditorForm} layout="vertical" onFinish={saveFooterLink}>
        <Form.Item name="label" label={t('label')} rules={[{ required: true, message: t('requiredField') }]}><Input /></Form.Item>
        <Form.Item name="url" label={t('linkAddress')} rules={[{ required: true, message: t('requiredField') }, { validator: (_, value) => validHTTPURL(value) ? Promise.resolve() : Promise.reject(new Error(t('invalidURL'))) }]}><Input placeholder="https://..." /></Form.Item>
      </Form>
    </Modal>
    <Modal className={styles.siteDocumentModal} width={960} open={documentEditor !== null} title={documentEditor ? documentLabel(documentEditor.key) : ''}
      onCancel={() => setDocumentEditor(null)} onOk={() => documentEditor && saveDocument.mutate({ ...documentEditor, content_markdown: documentMarkdown.trim() })}
      confirmLoading={saveDocument.isPending} destroyOnHidden>
      <div className={styles.editorWorkspace}>
        <MDEditor height={430} value={documentMarkdown} onChange={value => setDocumentMarkdown(value ?? '')} preview="live" visibleDragbar={false} />
      </div>
    </Modal>
  </div>
}
