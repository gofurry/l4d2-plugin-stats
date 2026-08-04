import { BarChart, HeatmapChart, LineChart, PieChart } from 'echarts/charts'
import {
  CalendarComponent,
  DatasetComponent,
  GridComponent,
  LegendComponent,
  TitleComponent,
  TooltipComponent,
  VisualMapComponent,
} from 'echarts/components'
import { init, use as registerECharts, type EChartsCoreOption } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { useEffect, useRef } from 'react'

registerECharts([
  BarChart,
  LineChart,
  PieChart,
  HeatmapChart,
  CalendarComponent,
  DatasetComponent,
  GridComponent,
  LegendComponent,
  TitleComponent,
  TooltipComponent,
  VisualMapComponent,
  CanvasRenderer,
])

interface EChartProps {
  className?: string
  option: EChartsCoreOption
  ariaLabel: string
}

export function EChart({ className, option, ariaLabel }: EChartProps) {
  const elementRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const element = elementRef.current
    if (!element) return
    const chart = init(element, undefined, { renderer: 'canvas' })
    chart.setOption(option, { notMerge: true })
    const resize = new ResizeObserver(() => chart.resize())
    resize.observe(element)
    return () => {
      resize.disconnect()
      chart.dispose()
    }
  }, [option])

  return <div aria-label={ariaLabel} className={className} ref={elementRef} role="img" />
}
