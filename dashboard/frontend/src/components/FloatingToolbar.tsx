import { Tooltip } from 'antd'
import type { ReactNode } from 'react'

export interface FloatingToolbarItem {
  key: string
  label: string
  icon: ReactNode
  onClick: () => void
  disabled?: boolean
  danger?: boolean
  active?: boolean
}

export function FloatingToolbar({ ariaLabel, items }: { ariaLabel: string; items: FloatingToolbarItem[] }) {
  return <div aria-label={ariaLabel} className="floating-toolbar" role="toolbar">
    {items.map(item => <Tooltip key={item.key} placement="left" title={item.label}>
      <button aria-label={item.label} aria-pressed={item.active} className={`floating-toolbar-button${item.active ? ' active' : ''}${item.danger ? ' danger' : ''}`} disabled={item.disabled} onClick={item.onClick} type="button">
        <span className="floating-toolbar-icon">{item.icon}</span>
      </button>
    </Tooltip>)}
  </div>
}
