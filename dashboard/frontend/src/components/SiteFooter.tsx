import type { Site } from '../api'
import styles from './SiteFooter.module.scss'

export function SiteFooter({ site }: { site?: Site }) {
  if (!site?.footer_text && !site?.footer_links.length) return null
  return <footer className={styles.footer}>
    {site.footer_text && <div className={styles.text}>{site.footer_text}</div>}
    {site.footer_links.length > 0 && <nav className={`${styles.links} flex flex-wrap justify-center gap-x-[18px] gap-y-2`} aria-label="footer links">
      {site.footer_links.map((link) => <a key={`${link.label}-${link.url}`} href={link.url}
        target={link.open_new_tab ? '_blank' : undefined}
        rel={link.open_new_tab ? 'noopener noreferrer' : undefined}>{link.label}</a>)}
    </nav>}
  </footer>
}
