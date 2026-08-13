import { Empty, Segmented, Skeleton } from 'antd'
import type { PlayerAnalysis as PlayerAnalysisData } from '../../api'
import { date } from './playerFormat'
import styles from './PlayerPage.module.scss'

const number = new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 })
const classNames: Record<number,string> = { 1:'Smoker',3:'Hunter',5:'Jockey',6:'Charger' }

export function PlayerAnalysis({data,loading,view,onView,zh}:{data?:PlayerAnalysisData;loading:boolean;view:string;onView:(value:string)=>void;zh:boolean}) {
  if (loading) return <section className={`${styles.dataSection} ${styles.wide}`}><Skeleton active/></section>
  if (!data) return <section className={`${styles.dataSection} ${styles.wide}`}><Empty/></section>
  const labels:Record<string,[string,string,string]> = {
    special_kills_per_hour:['特感击杀 / 小时','Special kills / hour','rate'], rescues_per_hour:['团队救援 / 小时','Team rescues / hour','rate'], incaps_per_hour:['倒地 / 小时','Incaps / hour','rate'], deaths_per_hour:['死亡 / 小时','Deaths / hour','rate'], friendly_fire_per_hour:['友伤 / 小时','Friendly fire / hour','rate'], tank_participation_rate:['Tank 参与率','Tank participation','percent'], witch_participation_rate:['Witch 参与率','Witch participation','percent'], human_si_tank_kills_per_hour:['真人特感/Tank 击杀 / 小时','Human SI/Tank kills / hour','rate'], damage_per_hour:['伤害 / 小时','Damage / hour','rate'], incaps_per_spawn:['击倒真人 / 复活','Human incaps / spawn','rate'], controls_per_spawn:['控制真人 / 复活','Human controls / spawn','rate'], kills_per_spawn:['击杀真人 / 复活','Human kills / spawn','rate'], average_control_seconds:['平均控制时长','Average control duration','seconds'],
  }
  const metrics = Object.entries(data.metrics).map(([key,value])=>[labels[key]?.[zh?0:1]??key, value===null?'—':labels[key]?.[2]==='percent'?`${(value*100).toFixed(1)}%`:labels[key]?.[2]==='seconds'?`${value.toFixed(1)}s`:number.format(value)]) as [string,string][]
  const incidents=data.recent_incidents
  return <div className={styles.analysisStack}>
    <section className={`${styles.dataSection} ${styles.wide}`}><div className={styles.analysisHeader}><div><h3>{zh?'标准化生涯指标':'Normalized career metrics'}</h3><p>{zh?'除零显示为不可用；Boss 与控制时长遵循最低样本门槛。':'Division by zero is unavailable; boss and control-duration metrics use sample gates.'}</p></div><Segmented value={view} onChange={value=>onView(String(value))} options={[{value:'pve',label:'PvE'},{value:'versus_survivor',label:zh?'对抗幸存者':'Versus survivor'},{value:'versus_infected',label:zh?'对抗感染者':'Versus infected'}]}/></div><dl className={styles.metricList}>{metrics.map(([label,value])=><div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl></section>
    <section className={styles.dataSection}><h3>{zh?'最近 Incident':'Recent incidents'}</h3><p className={styles.analysisHint}>{zh?'可用窗口':'Availability'}: {incidents.earliest_incident_at?date(incidents.earliest_incident_at):'—'} — {incidents.latest_incident_at?date(incidents.latest_incident_at):'—'}</p><dl className={styles.metricList}><div><dt>{zh?'受控次数':'Controls received'}</dt><dd>{incidents.controls_received}</dd></div><div><dt>{zh?'平均受控时长':'Avg control duration'}</dt><dd>{incidents.average_control_seconds===undefined?'—':`${incidents.average_control_seconds.toFixed(1)}s`}</dd></div><div><dt>{zh?'倒地 / 死亡':'Incaps / deaths'}</dt><dd>{incidents.incaps} / {incidents.deaths}</dd></div><div><dt>{zh?'救援队友':'Teammates rescued'}</dt><dd>{incidents.teammates_rescued}</dd></div><div><dt>{zh?'被队友救援':'Rescued by teammates'}</dt><dd>{incidents.rescued_by_teammates}</dd></div></dl></section>
    <section className={styles.dataSection}><h3>{zh?'控制与救援事实':'Control and rescue facts'}</h3><div className={styles.analysisRows}><div><strong>{zh?'同步控制参与':'Synchronized control participation'}</strong><span>2-cap {incidents.two_cap_episodes} · 3-cap {incidents.three_cap_episodes} · 4-cap {incidents.four_cap_episodes}</span></div>{incidents.control_classes.map(item=><div key={item.infected_class}><strong>{classNames[item.infected_class]??`Class ${item.infected_class}`}</strong><span>{item.controls} · {item.average_duration_seconds.toFixed(1)}s</span></div>)}{incidents.top_rescuers.map(item=><div key={item.player_name}><strong>{item.player_name}</strong><span>{zh?'可靠救援':'verified rescues'} · {item.rescues}</span></div>)}</div></section>
  </div>
}
