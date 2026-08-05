import { FilterOutlined } from '@ant-design/icons'
import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { Alert, Button, Empty, Input, Layout, Modal, Select, Spin } from 'antd'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api'
import { FloatingNav } from '../components/FloatingNav'
import { FloatingToolbar } from '../components/FloatingToolbar'
import { MarkdownContent } from '../components/MarkdownContent'
import styles from './AnnouncementsPage.module.scss'

const formatDate = (value: number) => new Intl.DateTimeFormat(undefined, { dateStyle: 'long', timeStyle: 'short' }).format(new Date(value * 1000))

export function AnnouncementsPage() {
  const { t } = useTranslation()
  const [filterOpen, setFilterOpen] = useState(false)
  const [titleFilter, setTitleFilter] = useState('')
  const [yearFilter, setYearFilter] = useState<number | undefined>()
  const [draftTitle, setDraftTitle] = useState('')
  const [draftYear, setDraftYear] = useState<number | undefined>()
  const years = useQuery({ queryKey: ['announcement-years'], queryFn: api.announcementYears, staleTime: 5 * 60_000 })
  const query = useInfiniteQuery({
    queryKey: ['announcements', titleFilter, yearFilter],
    initialPageParam: 1,
    queryFn: ({ pageParam }) => api.announcements(pageParam, 5, titleFilter, yearFilter),
    getNextPageParam: page => page.page * page.limit < page.total ? page.page + 1 : undefined,
  })
  const items = query.data?.pages.flatMap(page => page.items) ?? []
  const filtered = titleFilter !== '' || yearFilter !== undefined
  const openFilter = () => { setDraftTitle(titleFilter); setDraftYear(yearFilter); setFilterOpen(true) }
  const applyFilter = () => { setTitleFilter(draftTitle.trim()); setYearFilter(draftYear); setFilterOpen(false) }
  const resetFilter = () => { setDraftTitle(''); setDraftYear(undefined); setTitleFilter(''); setYearFilter(undefined); setFilterOpen(false) }

  return <Layout className={styles.layout}>
    <FloatingNav />
    <FloatingToolbar ariaLabel={t('announcementFilter')} items={[{ key: 'filter', label: t('announcementFilter'), icon: <FilterOutlined />, active: filtered, onClick: openFilter }]} />
    <Layout.Content className={styles.content}>
      {query.isLoading && <div className={styles.statePanel}><Spin size="large" /></div>}
      {query.isError && <Alert className={styles.statePanel} type="error" showIcon title={t('announcementsUnavailable')} />}
      {!query.isLoading && !query.isError && items.length === 0 && <div className={styles.statePanel}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('noAnnouncements')} /></div>}
      <div className={styles.announcementList}>{items.map(item => <article className={styles.announcement} key={item.id}>
        <header><div><h2>{item.title}</h2><time dateTime={new Date(item.updated_at * 1000).toISOString()}>{formatDate(item.updated_at)}</time></div>{item.updated_at !== item.created_at && <span>{t('announcementUpdated')}</span>}</header>
        <MarkdownContent source={item.content_markdown} />
      </article>)}</div>
      {query.hasNextPage && <div className={styles.loadMore}><Button loading={query.isFetchingNextPage} onClick={() => void query.fetchNextPage()}>{t('loadMore')}</Button></div>}
    </Layout.Content>
    <Modal open={filterOpen} title={t('announcementFilter')} onCancel={() => setFilterOpen(false)}
      footer={<><Button onClick={resetFilter}>{t('resetFilter')}</Button><Button type="primary" onClick={applyFilter}>{t('applyFilter')}</Button></>} destroyOnHidden>
      <div className={styles.filterForm}>
        <label><span>{t('announcementTitleSearch')}</span><Input value={draftTitle} maxLength={120} placeholder={t('announcementTitleSearchHint')} allowClear onChange={event => setDraftTitle(event.target.value)} onPressEnter={applyFilter} /></label>
        <label><span>{t('announcementYear')}</span><Select value={draftYear} placeholder={t('allYears')} allowClear loading={years.isLoading}
          options={(years.data ?? []).map(year => ({ value: year, label: String(year) }))} onChange={setDraftYear} /></label>
      </div>
    </Modal>
  </Layout>
}
