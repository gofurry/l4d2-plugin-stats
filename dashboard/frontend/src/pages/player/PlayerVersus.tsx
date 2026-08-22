import { Spin, Table, Tooltip } from 'antd'
import type { EChartsCoreOption } from 'echarts/core'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import type { PlayerVersusInfected, PlayerVersusInfectedDetails, PlayerVersusSurvivor, PlayerVersusSurvivorDetails, VersusInfectedClass, VersusSurvivorClass } from '../../api'
import { EChart } from '../../components/EChart'
import { MetricList, Section } from './PlayerShared'
import { chartBase, duration, infectedNames, numberFormat, palette, type PlayerCopy } from './playerFormat'
import styles from './PlayerPage.module.scss'

type VersusView = 'survivor' | 'survivor-details' | 'infected' | 'infected-details'

type PlayerVersusData = PlayerVersusSurvivor | PlayerVersusSurvivorDetails | PlayerVersusInfected | PlayerVersusInfectedDetails

export function PlayerVersus({ data, loading, view, copy, zh }: { data?: PlayerVersusData; loading: boolean; view: VersusView; copy: PlayerCopy; zh: boolean }) {
  const { t } = useTranslation()
  const classOption = useMemo<EChartsCoreOption>(() => {
    const infectedClasses = data && 'infected_classes' in data ? data.infected_classes : []
    return {
      ...chartBase,
      legend: { top: 0, textStyle: { color: '#6f5b50' } },
      xAxis: { type: 'category', data: infectedClasses.map(item => infectedNames[item.class_id - 1] ?? `#${item.class_id}`), axisLabel: { color: '#806d62' }, axisLine: { lineStyle: { color: 'rgba(102,75,60,.22)' } } },
      yAxis: { type: 'value', axisLabel: { color: '#806d62' }, splitLine: { lineStyle: { color: 'rgba(102,75,60,.1)' } } },
      series: [
        { name: copy.damage, type: 'bar', data: infectedClasses.map(item => item.damage_to_human_survivors), itemStyle: { color: palette[0] } },
        { name: t('incaps'), type: 'bar', data: infectedClasses.map(item => item.human_survivor_incaps), itemStyle: { color: palette[2] } },
        { name: t('controls'), type: 'bar', data: infectedClasses.map(item => item.human_survivor_controls), itemStyle: { color: palette[1] } },
      ],
    }
  }, [copy.damage, data, t])
  if (loading) return <Spin />
  if (!data) return null
  if (view === 'survivor') {
    const survivor = data as PlayerVersusSurvivor
    return <div className={styles.sectionGrid}>
      <Section title={copy.combat}><MetricList items={[[t('commonKills'), survivor.survivor_common_kills], [copy.humanSI, survivor.human_special_kills], [zh ? '助攻真人特感' : 'Human SI assists', nullable(survivor.human_special_assists, zh)], [copy.botSI, survivor.bot_special_kills], [zh ? '助攻 Bot 特感' : 'Bot SI assists', nullable(survivor.bot_special_assists, zh)], [copy.humanTank, survivor.human_tank_kills], [zh ? '助攻真人 Tank' : 'Human Tank assists', nullable(survivor.human_tank_assists, zh)], [copy.botTank, survivor.bot_tank_kills], [zh ? '助攻 Bot Tank' : 'Bot Tank assists', nullable(survivor.bot_tank_assists, zh)], [copy.damage, survivor.survivor_damage]]} /></Section>
      <Section title={copy.survival}><MetricList items={[[t('deaths'), survivor.survivor_deaths], [t('incaps'), survivor.survivor_incapacitations], [zh ? '挂边次数' : 'Ledge grabs', nullable(survivor.survivor_ledge_grabs, zh)], [copy.infectedDamage, survivor.survivor_damage_taken], [copy.friendlyFireDealt, survivor.survivor_friendly_fire], [copy.friendlyFireTaken, survivor.survivor_friendly_fire_taken], [copy.received, survivor.survivor_rescues_received]]} /></Section>
      <Section title={copy.teamwork}><MetricList items={[[<Tooltip title={zh ? '由游戏引擎的“保护队友”奖励判定。' : 'Awarded by the game engine Protect Teammate award.'}>{zh ? '保护队友' : 'Teammate protections'}</Tooltip>, nullable(survivor.survivor_teammate_protections, zh)], [t('revives'), survivor.survivor_incap_revives], [copy.ledgeRescue, survivor.survivor_ledge_rescues], [copy.defib, survivor.survivor_defib_revives], [copy.medkitOthers, survivor.survivor_medkits_others], [copy.healingOthers, survivor.survivor_healing_others], [copy.blackWhite, nullable(survivor.survivor_black_white_teammates_restored, zh)], [zh ? '自用医疗包 / 治疗量' : 'Self medkits / healing', `${numberFormat.format(survivor.survivor_medkits_self)} / ${numberFormat.format(survivor.survivor_healing_self)}`], [zh ? '止痛药 / 肾上腺素' : 'Pills / adrenaline', `${numberFormat.format(survivor.survivor_pills)} / ${numberFormat.format(survivor.survivor_adrenaline)}`], [zh ? '获得临时生命' : 'Temporary health received', survivor.survivor_temporary_health]]} /></Section>
      <Section title={copy.supplies}><MetricList items={[[zh ? '燃烧瓶' : 'Molotovs', survivor.molotovs_thrown], [zh ? '土制炸弹' : 'Pipe bombs', survivor.pipe_bombs_thrown], [zh ? '胆汁罐' : 'Vomit jars', survivor.vomit_jars_thrown], [zh ? '燃烧弹药包' : 'Incendiary packs', survivor.survivor_incendiary_packs], [zh ? '高爆弹药包' : 'Explosive packs', survivor.survivor_explosive_packs], [zh ? 'Witch 遭遇 / 击杀 / 助攻' : 'Witch encounters / kills / assists', `${nullable(survivor.survivor_witch_encounters, zh)} / ${numberFormat.format(survivor.survivor_witch_kills)} / ${nullable(survivor.survivor_witch_assists, zh)}`], [zh ? 'Witch 击杀参与' : 'Witch kill participations', nullable(survivor.survivor_witch_kill_participations, zh)], [zh ? 'Witch 伤害' : 'Witch damage', survivor.survivor_witch_damage]]} /></Section>
      <Section title={zh ? '互动与事件' : 'Interactions and incidents'}><MetricList items={[[copy.objective, survivor.survivor_objective_interactions], [zh ? '触发警报车' : 'Car alarms triggered', survivor.survivor_car_alarms_triggered]]} /></Section>
      <Section title={copy.skills}><MetricList items={[[zh ? 'Hunter 空中击杀' : 'Hunter Skeets', nullable(survivor.survivor_hunter_skeets, zh)], [zh ? '近战截停 Charger' : 'Charger Levels', nullable(survivor.survivor_charger_levels, zh)], [copy.rocks, survivor.survivor_tank_rocks_destroyed], [zh ? '被 Tank 石块命中' : 'Tank rock hits received', nullable(survivor.survivor_tank_rock_hits_received, zh)], [copy.tongue, survivor.survivor_tongue_self_cuts], [copy.witchOneShot, survivor.survivor_witch_oneshots], [copy.witchSolo, survivor.survivor_witch_solo_kills]]} /></Section>
    </div>
  }
  const survivorColumns = [
    { title: zh ? '类型' : 'Class', key: 'class', render: (_: unknown, item: VersusSurvivorClass) => infectedNames[item.class_id - 1] ?? `#${item.class_id}` },
    { title: copy.humanKills, dataIndex: 'human_controller_kills', key: 'humanKills' },
    { title: zh ? '助攻真人控制' : 'Human controller assists', dataIndex: 'human_controller_assists', key: 'humanAssists', render: (value: number | null) => nullable(value, zh) },
    { title: copy.botKills, dataIndex: 'bot_controller_kills', key: 'botKills' },
    { title: zh ? '助攻 Bot 控制' : 'Bot controller assists', dataIndex: 'bot_controller_assists', key: 'botAssists', render: (value: number | null) => nullable(value, zh) },
    { title: copy.humanDamage, dataIndex: 'damage_to_human_controllers', key: 'humanDamage' },
    { title: copy.botDamage, dataIndex: 'damage_to_bot_controllers', key: 'botDamage' },
  ]
  if (view === 'survivor-details') return <div className={styles.sectionGrid}>
    <Section title={copy.classes} wide><Table<VersusSurvivorClass> className={styles.embeddedTable} columns={survivorColumns} dataSource={(data as PlayerVersusSurvivorDetails).survivor_classes} rowKey="class_id" pagination={false} size="small" scroll={{ x: 720 }} /></Section>
  </div>
  if (view === 'infected') {
    const infected = data as PlayerVersusInfected
    return <div className={styles.sectionGrid}>
      <Section title={copy.overview}><MetricList items={[[t('infectedSpawns'), infected.infected_spawns], [copy.humanDamage, infected.damage_to_human_survivors], [copy.botDamage, infected.damage_to_bot_survivors], [copy.humanIncaps, infected.human_survivor_incaps], [copy.botIncaps, infected.bot_survivor_incaps], [copy.humanSurvivorKills, infected.human_survivor_kills], [copy.botSurvivorKills, infected.bot_survivor_kills], [t('controls'), infected.human_survivor_controls], [t('controlTime'), duration(infected.human_survivor_control_seconds)]]} /></Section>
    </div>
  }
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
    <Section title={copy.classes} wide><Table<VersusInfectedClass> className={styles.embeddedTable} columns={infectedColumns} dataSource={(data as PlayerVersusInfectedDetails).infected_classes} rowKey="class_id" pagination={false} size="small" scroll={{ x: 1200 }} /></Section>
  </div>
}

function nullable(value: number | null, zh: boolean) {
  return value === null ? (zh ? '未采集' : 'Not collected') : numberFormat.format(value)
}
