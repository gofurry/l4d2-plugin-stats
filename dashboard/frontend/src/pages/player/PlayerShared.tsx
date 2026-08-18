import { Alert } from 'antd'
import { Component, type ReactNode } from 'react'
import { numberFormat } from './playerFormat'
import styles from './PlayerPage.module.scss'

export type Metric = [ReactNode, number | string]

export function MetricList({ items }: { items: Metric[] }) {
  return <dl className={styles.metricList}>{items.map(([label, value], index) => <div key={index}><dt>{label}</dt><dd>{typeof value === 'number' ? numberFormat.format(value) : value}</dd></div>)}</dl>
}

export function Section({ title, children, wide = false }: { title: string; children: ReactNode; wide?: boolean }) {
  return <section className={`${styles.dataSection}${wide ? ` ${styles.wide}` : ''}`}><h3>{title}</h3>{children}</section>
}

export function PlayerTabPanel({ children, error, resetKey, zh }: { children: ReactNode; error?: boolean; resetKey: string; zh: boolean }) {
  if (error) return <Alert type="warning" showIcon title={zh ? '该栏目暂时无法读取，请稍后重试。' : 'This section is temporarily unavailable. Please try again later.'} />
  return <PlayerTabErrorBoundary resetKey={resetKey} message={zh ? '该栏目显示异常，请切换栏目或刷新页面后重试。' : 'This section could not be displayed. Switch sections or refresh the page.'}>{children}</PlayerTabErrorBoundary>
}

class PlayerTabErrorBoundary extends Component<{ children: ReactNode; resetKey: string; message: string }, { failed: boolean }> {
  state = { failed: false }

  static getDerivedStateFromError() {
    return { failed: true }
  }

  componentDidUpdate(previous: Readonly<{ resetKey: string }>) {
    if (this.state.failed && previous.resetKey !== this.props.resetKey) this.setState({ failed: false })
  }

  render() {
    if (this.state.failed) return <Alert type="error" showIcon title={this.props.message} />
    return this.props.children
  }
}
