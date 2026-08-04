import type { Site } from '../api'
import styles from './SiteFooter.module.scss'

export function SiteFooter({ site }: { site?: Site }) {
  if (!site?.footer_enabled || !site.footer_links.length) return null
  return <footer className={styles.footer}>
    <nav className={styles.links} aria-label="footer links">
      {site.footer_links.map((link, index) => <span className={styles.linkItem} key={`${link.label}-${link.url}`}>
        {index > 0 && <span className={styles.separator} aria-hidden="true">·</span>}
        <a href={link.url} target="_blank" rel="noopener noreferrer">{link.label}</a>
      </span>)}
    </nav>
  </footer>
}
