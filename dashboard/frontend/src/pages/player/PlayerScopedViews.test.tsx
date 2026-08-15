import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { PlayerPVEDetails, PlayerPVEOverview, PlayerVersusInfected, PlayerVersusInfectedDetails, PlayerVersusSurvivor, PlayerVersusSurvivorDetails } from '../../api'
import { PlayerPVE } from './PlayerPVE'
import { PlayerVersus } from './PlayerVersus'
import type { PlayerCopy } from './playerFormat'

vi.mock('../../components/EChart', () => ({ EChart: () => <div data-testid="chart" /> }))

const copy = new Proxy<Record<string, string>>({}, { get: (_target, key) => String(key) }) as PlayerCopy

describe('scoped player views', () => {
  it('renders the PvE overview without detail arrays', () => {
    expect(() => render(<PlayerPVE data={{} as PlayerPVEOverview} loading={false} copy={copy} zh />)).not.toThrow()
  })

  it('renders PvE details from the detail-only payload', () => {
    const data: PlayerPVEDetails = { infected_classes: [], equipment: [] }
    expect(() => render(<PlayerPVE data={data} loading={false} copy={copy} zh details />)).not.toThrow()
  })

  it('renders every Versus tab from its independently scoped payload', () => {
    const survivor = render(<PlayerVersus data={{} as PlayerVersusSurvivor} loading={false} view="survivor" copy={copy} zh />)
    survivor.unmount()
    const survivorDetails: PlayerVersusSurvivorDetails = { survivor_classes: [] }
    const details = render(<PlayerVersus data={survivorDetails} loading={false} view="survivor-details" copy={copy} zh />)
    details.unmount()
    const infected = render(<PlayerVersus data={{} as PlayerVersusInfected} loading={false} view="infected" copy={copy} zh />)
    infected.unmount()
    const infectedDetails: PlayerVersusInfectedDetails = { infected_classes: [] }
    expect(() => render(<PlayerVersus data={infectedDetails} loading={false} view="infected-details" copy={copy} zh />)).not.toThrow()
  })
})
