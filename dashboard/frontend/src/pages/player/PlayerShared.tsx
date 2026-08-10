import type { ReactNode } from 'react'
import { numberFormat } from './playerFormat'
import styles from './PlayerPage.module.scss'

export type Metric = [string, number | string]

export function MetricList({ items }: { items: Metric[] }) {
  return <dl className={styles.metricList}>{items.map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{typeof value === 'number' ? numberFormat.format(value) : value}</dd></div>)}</dl>
}

export function Section({ title, children, wide = false }: { title: string; children: ReactNode; wide?: boolean }) {
  return <section className={`${styles.dataSection}${wide ? ` ${styles.wide}` : ''}`}><h3>{title}</h3>{children}</section>
}
