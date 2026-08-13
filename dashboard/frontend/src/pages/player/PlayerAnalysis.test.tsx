import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { PlayerAnalysis as PlayerAnalysisData } from '../../api'
import { PlayerAnalysis } from './PlayerAnalysis'

const data: PlayerAnalysisData = {
  view: 'versus_infected',
  active_play_seconds: 7200,
  metrics: {},
  samples: {},
  recent_incidents: {
    earliest_incident_at: 1_786_623_421,
    latest_incident_at: 1_786_626_233,
    controls_received: 12,
    average_control_seconds: 5.4,
    incaps: 3,
    deaths: 1,
    teammates_rescued: 2,
    rescued_by_teammates: 4,
    control_classes: [{ infected_class: 5, controls: 4, average_duration_seconds: 9.3 }],
    top_rescuers: [{ player_name: '队友甲', rescues: 3 }],
    two_cap_episodes: 2,
    three_cap_episodes: 1,
    four_cap_episodes: 0,
  },
}

describe('PlayerAnalysis', () => {
  it('explains incident coverage and control perspectives in Chinese', () => {
    render(<PlayerAnalysis data={data} loading={false} view="versus_infected" onView={() => undefined} zh />)
    expect(screen.getByText('战局事件统计')).toBeInTheDocument()
    expect(screen.getByText('数据覆盖时间')).toBeInTheDocument()
    expect(screen.getByText('作为感染者参与多人同时控制')).toBeInTheDocument()
    expect(screen.getByText('被 Jockey 控制')).toBeInTheDocument()
    expect(screen.getByText('4 次 · 平均 9.3 秒')).toBeInTheDocument()
    expect(screen.getByText('救起你 3 次')).toBeInTheDocument()
    expect(screen.queryByText(/—/)).not.toBeInTheDocument()
  })
})
