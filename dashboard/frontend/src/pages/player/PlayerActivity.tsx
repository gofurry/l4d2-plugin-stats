import { Empty, Spin } from 'antd'
import type { EChartsCoreOption } from 'echarts/core'
import { useEffect, useMemo, useState } from 'react'
import type { PlayerActivity as PlayerActivityData } from '../../api'
import { EChart } from '../../components/EChart'
import { Section } from './PlayerShared'
import { chartBase, palette, type PlayerCopy } from './playerFormat'
import styles from './PlayerPage.module.scss'

function currentChartTextColor() {
  return getComputedStyle(document.documentElement).getPropertyValue('--text-primary').trim() || '#4e3c32'
}

function formatActivityDay(day: number) {
  return new Date(day * 86400_000).toLocaleDateString(undefined, { timeZone: 'UTC' })
}

function useChartTextColor() {
  const [color, setColor] = useState(currentChartTextColor)
  useEffect(() => {
    const update = () => setColor(currentChartTextColor())
    update()
    const observer = new MutationObserver(update)
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })
    return () => observer.disconnect()
  }, [])
  return color
}

export function PlayerActivity({ data, loading, copy }: { data?: PlayerActivityData; loading: boolean; copy: PlayerCopy }) {
  const chartTextColor = useChartTextColor()
  const showSymbols = (data?.timeline.length ?? 0) <= 31
  const activityOption = useMemo<EChartsCoreOption>(() => ({
    ...chartBase,
    legend: { top: 0, textStyle: { color: chartTextColor } },
    xAxis: { type: 'category', data: data?.timeline.map(point => formatActivityDay(point.day)) ?? [], axisLabel: { color: chartTextColor, hideOverlap: true }, axisLine: { lineStyle: { color: chartTextColor, opacity: 0.32 } } },
    yAxis: { type: 'value', axisLabel: { color: chartTextColor, formatter: '{value}h' }, splitLine: { lineStyle: { color: chartTextColor, opacity: 0.14 } } },
    series: [
      { name: copy.activeTime, type: 'line', smooth: true, showSymbol: showSymbols, symbolSize: 6, data: data?.timeline.map(point => +(point.active_play_seconds / 3600).toFixed(2)) ?? [], lineStyle: { width: 3, color: palette[0] }, areaStyle: { color: 'rgba(198,111,59,.12)' } },
      { name: copy.connected, type: 'line', smooth: true, showSymbol: showSymbols, symbolSize: 6, data: data?.timeline.map(point => +(point.connected_seconds / 3600).toFixed(2)) ?? [], lineStyle: { width: 2, color: palette[1] } },
    ],
  }), [chartTextColor, copy.activeTime, copy.connected, data, showSymbols])
  const serverOption = useMemo<EChartsCoreOption>(() => ({
    ...chartBase,
    tooltip: { trigger: 'item', backgroundColor: 'rgba(67,48,38,.94)', borderWidth: 0, textStyle: { color: '#fff7ed' } },
    legend: { bottom: 0, type: 'scroll', textStyle: { color: chartTextColor } },
    series: [{
      type: 'pie',
      radius: ['45%', '70%'],
      center: ['50%', '44%'],
      padAngle: 2,
      itemStyle: { borderRadius: 5 },
      label: { color: chartTextColor, textBorderColor: 'transparent', textBorderWidth: 0, textShadowBlur: 0 },
      labelLine: { lineStyle: { color: chartTextColor, opacity: 0.62 } },
      data: data?.servers.slice(0, 10).map((item, index) => ({ name: item.server_key, value: +(item.active_play_seconds / 3600).toFixed(2), itemStyle: { color: palette[index % palette.length] } })) ?? [],
    }],
  }), [chartTextColor, data])
  return <div className={styles.chartGrid}>
    <Section title={copy.activity}>{loading ? <Spin /> : data?.timeline.length ? <EChart ariaLabel={copy.activity} className={styles.chart} option={activityOption} /> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={copy.noActivity} />}</Section>
    <Section title={copy.serverShare}>{loading ? <Spin /> : <EChart ariaLabel={copy.serverShare} className={styles.chart} option={serverOption} />}</Section>
  </div>
}
