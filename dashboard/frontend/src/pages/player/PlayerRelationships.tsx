import { useQuery } from '@tanstack/react-query'
import { Alert, Drawer, Empty, Select, Spin, Table } from 'antd'
import type { TableProps } from 'antd'
import { useState } from 'react'
import { api, type PlayerRelationship, type PlayerRelationshipDirection, type PlayerRelationshipSummary } from '../../api'
import { duration, numberFormat } from './playerFormat'
import styles from './PlayerPage.module.scss'

type SortField = 'player_name' | 'shared_rounds' | 'shared_seconds' | 'outgoing_support' | 'incoming_support' | 'mutual_support' | 'outgoing_healing' | 'incoming_healing' | 'outgoing_friendly_fire' | 'incoming_friendly_fire'

export function PlayerRelationships({ steamID, range, server, enabled, zh }: { steamID: string; range: string; server: string; enabled: boolean; zh: boolean }) {
  const [mode, setMode] = useState('all')
  const [page, setPage] = useState(1)
  const [sort, setSort] = useState<SortField>('shared_rounds')
  const [order, setOrder] = useState<'asc' | 'desc'>('desc')
  const [selected, setSelected] = useState<PlayerRelationship>()
  const query = useQuery({
    queryKey: ['player-relationships', steamID, range, server, mode, page, sort, order],
    queryFn: () => api.playerRelationships(steamID, range, server, mode, page, 20, sort, order),
    enabled,
    placeholderData: previous => previous,
  })

  const sortable = (field: SortField) => ({ key: field, sorter: true, sortOrder: sort === field ? (order === 'asc' ? 'ascend' as const : 'descend' as const) : null })
  const columns: TableProps<PlayerRelationship>['columns'] = [
    { title: zh ? '玩家' : 'Player', ...sortable('player_name'), render: (_, row) => <div className={styles.relationshipPlayer}><strong>{row.peer_name || row.peer_steam_id}</strong><small>{row.peer_steam_id}</small></div> },
    { title: zh ? '并肩' : 'Together', ...sortable('shared_rounds'), render: (_, row) => `${numberFormat.format(row.shared_rounds)} ${zh ? '局' : 'rounds'} · ${duration(row.shared_seconds)}` },
    { title: zh ? '我支援 TA' : 'My support', ...sortable('outgoing_support'), render: (_, row) => numberFormat.format(row.outgoing.support_actions) },
    { title: zh ? 'TA 支援我' : 'Their support', ...sortable('incoming_support'), render: (_, row) => numberFormat.format(row.incoming.support_actions) },
    { title: zh ? '我治疗 TA' : 'My healing', ...sortable('outgoing_healing'), render: (_, row) => numberFormat.format(row.outgoing.medkit_healing) },
    { title: zh ? 'TA 治疗我' : 'Their healing', ...sortable('incoming_healing'), render: (_, row) => numberFormat.format(row.incoming.medkit_healing) },
    { title: zh ? '我友伤 TA' : 'My FF', ...sortable('outgoing_friendly_fire'), render: (_, row) => numberFormat.format(row.outgoing.friendly_fire_damage) },
    { title: zh ? 'TA 友伤我' : 'Their FF', ...sortable('incoming_friendly_fire'), render: (_, row) => numberFormat.format(row.incoming.friendly_fire_damage) },
  ]
  const onChange: TableProps<PlayerRelationship>['onChange'] = (pagination, _filters, sorter, extra) => {
    if (extra.action === 'sort') {
      const current = Array.isArray(sorter) ? sorter[0] : sorter
      if (current?.order) {
        setSort(String(current.columnKey) as SortField)
        setOrder(current.order === 'ascend' ? 'asc' : 'desc')
        setPage(1)
      }
    } else {
      setPage(pagination.current ?? 1)
    }
  }

  return <div className={styles.relationshipStack}>
    <Alert type="info" showIcon message={zh ? '定向互动关系统计仅覆盖服务器启用 Player Relationship Contract v1 后产生的数据。' : 'Directional interactions only cover data collected after Player Relationship Contract v1 was enabled.'} />
    <div className={styles.relationshipToolbar}><Select value={mode} onChange={value => { setMode(value); setPage(1) }} options={[{ value: 'all', label: zh ? 'PvE + 对抗' : 'PvE + Versus' }, { value: 'pve', label: 'PvE' }, { value: 'versus', label: zh ? '对抗' : 'Versus' }]} /></div>
    {query.isLoading ? <div className={styles.relationshipLoading}><Spin /></div> : query.data ? <>
      <div className={styles.relationshipSummaries}>
        <RelationshipSummary title={zh ? '最常并肩' : 'Most time together'} value={query.data.summaries.most_companion} kind="companion" zh={zh} />
        <RelationshipSummary title={zh ? '我最常支援' : 'Most supported by me'} value={query.data.summaries.most_supported} zh={zh} />
        <RelationshipSummary title={zh ? '最常支援我' : 'Most support for me'} value={query.data.summaries.most_supported_by} zh={zh} />
        <RelationshipSummary title={zh ? '互相支援最多' : 'Most mutual support'} value={query.data.summaries.most_mutual} zh={zh} />
      </div>
      <section className={styles.dataSection}>{query.data.items.length ? <Table<PlayerRelationship> className={styles.embeddedTable} columns={columns} dataSource={query.data.items} rowKey="peer_steam_id" loading={query.isFetching} pagination={{ current: page, pageSize: 20, total: query.data.total, showSizeChanger: false, showTotal: total => zh ? `共 ${numberFormat.format(total)} 位玩家` : `${numberFormat.format(total)} players` }} onChange={onChange} onRow={row => ({ onClick: () => setSelected(row) })} scroll={{ x: 1080 }} /> : <Empty description={zh ? '暂无玩家关系数据' : 'No relationship data'} />}</section>
    </> : <Alert type="warning" showIcon message={zh ? '玩家关系数据暂时不可用' : 'Relationship data is unavailable'} />}
    <Drawer width={620} title={selected?.peer_name || selected?.peer_steam_id} open={Boolean(selected)} onClose={() => setSelected(undefined)}>
      {selected && <div className={styles.relationshipDrawer}>
        <Direction title={zh ? `我 → ${selected.peer_name || 'TA'}` : `Me → ${selected.peer_name || 'them'}`} value={selected.outgoing} zh={zh} />
        <Direction title={zh ? `${selected.peer_name || 'TA'} → 我` : `${selected.peer_name || 'They'} → me`} value={selected.incoming} zh={zh} />
        <a className={styles.relationshipProfileLink} href={`/player?steam_id=${selected.peer_steam_id}`}>{zh ? '查看该玩家个人主页' : 'View player profile'}</a>
      </div>}
    </Drawer>
  </div>
}

function RelationshipSummary({ title, value, kind, zh }: { title: string; value?: PlayerRelationshipSummary; kind?: 'companion'; zh: boolean }) {
  return <div><span>{title}</span>{value ? <><strong>{value.peer_name || value.peer_steam_id}</strong><small>{kind === 'companion' ? `${numberFormat.format(value.shared_rounds)} ${zh ? '局' : 'rounds'} · ${duration(value.shared_seconds)}` : `${numberFormat.format(value.support_actions)} ${zh ? '次支援' : 'support actions'}`}</small></> : <strong>—</strong>}</div>
}

function Direction({ title, value, zh }: { title: string; value: PlayerRelationshipDirection; zh: boolean }) {
  const response = value.average_control_rescue_ms === undefined ? '—' : `${(value.average_control_rescue_ms / 1000).toFixed(2)}s`
  const items = [
    [zh ? '扶起' : 'Incap revives', value.incap_revives], [zh ? '挂边救援' : 'Ledge rescues', value.ledge_rescues], [zh ? '电击复活' : 'Defib revives', value.defib_revives],
    ['Smoker', value.smoker_rescues], ['Hunter', value.hunter_rescues], ['Jockey', value.jockey_rescues], ['Charger', value.charger_rescues], [zh ? '平均特感解救响应' : 'Average control rescue response', response],
    [zh ? '医疗包次数' : 'Medkits', value.medkits_used], [zh ? '实际治疗 HP' : 'Actual healing HP', value.medkit_healing], [zh ? '黑白救回' : 'Black-and-white restores', value.black_white_restores], [zh ? '友伤 HP' : 'Friendly-fire HP', value.friendly_fire_damage],
  ]
  return <section><h3>{title}</h3><dl>{items.map(([label, metric]) => <div key={String(label)}><dt>{label}</dt><dd>{typeof metric === 'number' ? numberFormat.format(metric) : metric}</dd></div>)}</dl></section>
}
