import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import styles from './MarkdownContent.module.scss'

export function MarkdownContent({ source }: { source: string }) {
  return <div className={styles.markdown}>
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={{
        a: ({ children, href }) => <a href={href} rel="noopener noreferrer" target="_blank">{children}</a>,
      }}
    >{source}</ReactMarkdown>
  </div>
}
