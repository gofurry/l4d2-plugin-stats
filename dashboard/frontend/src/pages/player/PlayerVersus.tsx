import { Spin, Table } from 'antd'
import type { EChartsCoreOption } from 'echarts/core'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import type { PlayerVersus as PlayerVersusData, VersusInfectedClass, VersusSurvivorClass } from '../../api'
import { EChart } from '../../components/EChart'
import { MetricList, Section } from './PlayerShared'
import { chartBase, duration, infectedNames, numberFormat, palette, type PlayerCopy } from './playerFormat'
import styles from './PlayerPage.module.scss'

type VersusView = 'survivor' | 'survivor-details' | 'infected' | 'infected-details'

export function PlayerVersus({ data, loading, view, copy, zh }: { data?: PlayerVersusData; loading: boolean; view: VersusView; copy: PlayerCopy; zh: boolean }) {
  const { t } = useTranslation()
  const classOption = useMemo<EChartsCoreOption>(() => ({
    ...chartBase,
    legend: { top: 0, textStyle: { color: '#6f5b50' } },
    xAxis: { type: 'category', data: data?.infected_classes.map(item => infectedNames[item.class_id - 1] ?? `#${item.class_id}`) ?? [], axisLabel: { color: '#806d62' }, axisLine: { lineStyle: { color: 'rgba(102,75,60,.22)' } } },
    yAxis: { type: 'value', axisLabel: { color: '#806d62' }, splitLine: { lineStyle: { color: 'rgba(102,75,60,.1)' } } },
    series: [
      { name: copy.damage, type: 'bar', data: data?.infected_classes.map(item => item.damage_to_human_survivors) ?? [], itemStyle: { color: palette[0] } },
      { name: t('incaps'), type: 'bar', data: data?.infected_classes.map(item => item.human_survivor_incaps) ?? [], itemStyle: { color: palette[2] } },
      { name: t('controls'), type: 'bar', data: data?.infected_classes.map(item => item.human_survivor_controls) ?? [], itemStyle: { color: palette[1] } },
    ],
  }), [copy.damage, data, t])
  if (loading) return <Spin />
  if (!data) return null
  if (view === 'survivor') return <div className={styles.sectionGrid}>
    <Section title={copy.combat}><MetricList items={[[t('commonKills'), data.survivor_common_kills], [copy.humanSI, data.human_special_kills], [copy.botSI, data.bot_special_kills], [copy.humanTank, data.human_tank_kills], [copy.botTank, data.bot_tank_kills], [copy.damage, data.survivor_damage]]} /></Section>
    <Section title={copy.survival}><MetricList items={[[t('deaths'), data.survivor_deaths], [t('incaps'), data.survivor_incapacitations], [copy.infectedDamage, data.survivor_damage_taken], [copy.friendlyFireDealt, data.survivor_friendly_fire], [copy.friendlyFireTaken, data.survivor_friendly_fire_taken], [copy.received, data.survivor_rescues_received]]} /></Section>
    <Section title={copy.teamwork}><MetricList items={[[t('revives'), data.survivor_incap_revives], [copy.ledgeRescue, data.survivor_ledge_rescues], [copy.defib, data.survivor_defib_revives], [copy.medkitOthers, data.survivor_medkits_others], [copy.healingOthers, data.survivor_healing_others], [zh ? '自用医疗包 / 治疗量' : 'Self medkits / healing', `${numberFormat.format(data.survivor_medkits_self)} / ${numberFormat.format(data.survivor_healing_self)}`], [zh ? '止痛药 / 肾上腺素' : 'Pills / adrenaline', `${numberFormat.format(data.survivor_pills)} / ${numberFormat.format(data.survivor_adrenaline)}`], [zh ? '获得临时生命' : 'Temporary health received', data.survivor_temporary_health]]} /></Section>
    <Section title={copy.supplies}><MetricList items={[[zh ? '燃烧瓶' : 'Molotovs', data.molotovs_thrown], [zh ? '土制炸弹' : 'Pipe bombs', data.pipe_bombs_thrown], [zh ? '胆汁罐' : 'Vomit jars', data.vomit_jars_thrown], [zh ? '燃烧弹药包' : 'Incendiary packs', data.survivor_incendiary_packs], [zh ? '高爆弹药包' : 'Explosive packs', data.survivor_explosive_packs], [zh ? 'Witch 击杀 / 伤害' : 'Witch kills / damage', `${numberFormat.format(data.survivor_witch_kills)} / ${numberFormat.format(data.survivor_witch_damage)}`]]} /></Section>
    <Section title={zh ? '互动与事件' : 'Interactions and incidents'}><MetricList items={[[copy.objective, data.survivor_objective_interactions], [zh ? '触发警报车' : 'Car alarms triggered', data.survivor_car_alarms_triggered]]} /></Section>
    <Section title={copy.skills}><MetricList items={[[copy.tongue, data.survivor_tongue_self_cuts], [copy.rocks, data.survivor_tank_rocks_destroyed], [copy.witchOneShot, data.survivor_witch_oneshots], [copy.witchSolo, data.survivor_witch_solo_kills]]} /></Section>
  </div>
  const survivorColumns = [
    { title: zh ? '类型' : 'Class', key: 'class', render: (_: unknown, item: VersusSurvivorClass) => infectedNames[item.class_id - 1] ?? `#${item.class_id}` },
    { title: copy.humanKills, dataIndex: 'human_controller_kills', key: 'humanKills' },
    { title: copy.botKills, dataIndex: 'bot_controller_kills', key: 'botKills' },
    { title: copy.humanDamage, dataIndex: 'damage_to_human_controllers', key: 'humanDamage' },
    { title: copy.botDamage, dataIndex: 'damage_to_bot_controllers', key: 'botDamage' },
  ]
  if (view === 'survivor-details') return <div className={styles.sectionGrid}>
    <Section title={copy.classes} wide><Table<VersusSurvivorClass> className={styles.embeddedTable} columns={survivorColumns} dataSource={data.survivor_classes} rowKey="class_id" pagination={false} size="small" scroll={{ x: 720 }} /></Section>
  </div>
  if (view === 'infected') return <div className={styles.sectionGrid}>
    <Section title={copy.overview}><MetricList items={[[t('infectedSpawns'), data.infected_spawns], [copy.humanDamage, data.damage_to_human_survivors], [copy.botDamage, data.damage_to_bot_survivors], [copy.humanIncaps, data.human_survivor_incaps], [copy.botIncaps, data.bot_survivor_incaps], [copy.humanSurvivorKills, data.human_survivor_kills], [copy.botSurvivorKills, data.bot_survivor_kills], [t('controls'), data.human_survivor_controls], [t('controlTime'), duration(data.human_survivor_control_seconds)]]} /></Section>
  </div>
  const infectedColumns = [
    { title: zh ? '类型' : 'Class', key: 'class', render: (_: unknown, item: VersusInfectedClass) => infectedNames[item.class_id - 1] ?? `#${item.class_id}` },
    { title: copy.spawns, dataIndex: 'spawns', key: 'spawns' },
    { title: copy.humanDamage, dataIndex: 'damage_to_human_survivors', key: 'humanDamage' },
    { title: copy.botDamage, dataIndex: 'damage_to_bot_survivors', key: 'botDamage' },
    { title: copy.humanIncaps, dataIndex: 'human_survivor_incaps', key: 'humanIncaps' },
    { title: copy.botIncaps, dataIndex: 'bot_survivor_incaps', key: 'botIncaps' },
    { title: copy.humanSurvivorKills, dataIndex: 'human_survivor_kills', key: 'humanKills' },
    { title: copy.botSurvivorKills, dataIndex: 'bot_survivor_kills', key: 'botKills' },
    { title: copy.humanControls, dataIndex: 'human_survivor_controls', key: 'humanControls' },
    { title: copy.botControls, dataIndex: 'bot_survivor_controls', key: 'botControls' },
    { title: copy.abilityHits, key: 'abilityHits', render: (_: unknown, item: VersusInfectedClass) => numberFormat.format(item.human_survivor_ability_hits + item.bot_survivor_ability_hits) },
    { title: copy.abilityDamage, key: 'abilityDamage', render: (_: unknown, item: VersusInfectedClass) => numberFormat.format(item.human_survivor_ability_damage + item.bot_survivor_ability_damage) },
  ]
  return <div className={styles.sectionGrid}>
    <Section title={copy.classes} wide><EChart ariaLabel={copy.classes} className={styles.largeChart} option={classOption} /></Section>
    <Section title={copy.classes} wide><Table<VersusInfectedClass> className={styles.embeddedTable} columns={infectedColumns} dataSource={data.infected_classes} rowKey="class_id" pagination={false} size="small" scroll={{ x: 1200 }} /></Section>
  </div>
}
