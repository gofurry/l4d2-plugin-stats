import { useInfiniteQuery } from '@tanstack/react-query'
import { Alert, Button, Empty, Layout, Spin } from 'antd'
import { useTranslation } from 'react-i18next'
import { api } from '../api'
import { FloatingNav } from '../components/FloatingNav'
import { MarkdownContent } from '../components/MarkdownContent'
import styles from './AnnouncementsPage.module.scss'

const formatDate = (value: number) => new Intl.DateTimeFormat(undefined, { dateStyle: 'long', timeStyle: 'short' }).format(new Date(value * 1000))

export function AnnouncementsPage() {
  const { t } = useTranslation()
  const query = useInfiniteQuery({
    queryKey: ['announcements'],
    initialPageParam: 1,
    queryFn: ({ pageParam }) => api.announcements(pageParam, 5),
    getNextPageParam: page => page.page * page.limit < page.total ? page.page + 1 : undefined,
  })
  const items = query.data?.pages.flatMap(page => page.items) ?? []

  return <Layout className={styles.layout}>
    <FloatingNav />
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
  </Layout>
}
