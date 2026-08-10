import { Empty, Spin } from 'antd'
import type { EChartsCoreOption } from 'echarts/core'
import { useMemo } from 'react'
import type { PlayerActivity as PlayerActivityData } from '../../api'
import { EChart } from '../../components/EChart'
import { Section } from './PlayerShared'
import { chartBase, palette, type PlayerCopy } from './playerFormat'
import styles from './PlayerPage.module.scss'

export function PlayerActivity({ data, loading, copy }: { data?: PlayerActivityData; loading: boolean; copy: PlayerCopy }) {
  const activityOption = useMemo<EChartsCoreOption>(() => ({
    ...chartBase,
    legend: { top: 0, textStyle: { color: '#6f5b50' } },
    xAxis: { type: 'category', data: data?.timeline.map(point => new Date(point.day * 86400_000).toLocaleDateString()) ?? [], axisLabel: { color: '#806d62', hideOverlap: true }, axisLine: { lineStyle: { color: 'rgba(102,75,60,.22)' } } },
    yAxis: { type: 'value', axisLabel: { color: '#806d62', formatter: '{value}h' }, splitLine: { lineStyle: { color: 'rgba(102,75,60,.1)' } } },
    series: [
      { name: copy.activeTime, type: 'line', smooth: true, showSymbol: false, data: data?.timeline.map(point => +(point.active_play_seconds / 3600).toFixed(2)) ?? [], lineStyle: { width: 3, color: palette[0] }, areaStyle: { color: 'rgba(198,111,59,.12)' } },
      { name: copy.connected, type: 'line', smooth: true, showSymbol: false, data: data?.timeline.map(point => +(point.connected_seconds / 3600).toFixed(2)) ?? [], lineStyle: { width: 2, color: palette[1] } },
    ],
  }), [copy.activeTime, copy.connected, data])
  const serverOption = useMemo<EChartsCoreOption>(() => ({
    ...chartBase,
    tooltip: { trigger: 'item', backgroundColor: 'rgba(67,48,38,.94)', borderWidth: 0, textStyle: { color: '#fff7ed' } },
    legend: { bottom: 0, type: 'scroll', textStyle: { color: '#6f5b50' } },
    series: [{ type: 'pie', radius: ['45%', '70%'], center: ['50%', '44%'], padAngle: 2, itemStyle: { borderRadius: 5 }, data: data?.servers.slice(0, 10).map((item, index) => ({ name: item.server_key, value: +(item.active_play_seconds / 3600).toFixed(2), itemStyle: { color: palette[index % palette.length] } })) ?? [] }],
  }), [data])
  return <div className={styles.chartGrid}>
    <Section title={copy.activity}>{loading ? <Spin /> : data?.timeline.length ? <EChart ariaLabel={copy.activity} className={styles.chart} option={activityOption} /> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={copy.noActivity} />}</Section>
    <Section title={copy.serverShare}>{loading ? <Spin /> : <EChart ariaLabel={copy.serverShare} className={styles.chart} option={serverOption} />}</Section>
  </div>
}
