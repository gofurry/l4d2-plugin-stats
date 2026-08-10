import { Spin, Table } from 'antd'
import type { EChartsCoreOption } from 'echarts/core'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import type { PlayerPVE as PlayerPVEData, PVEEquipment, PVEInfectedClass } from '../../api'
import { EChart } from '../../components/EChart'
import { MetricList, Section } from './PlayerShared'
import { chartBase, duration, equipmentNames, infectedNames, numberFormat, palette, type PlayerCopy } from './playerFormat'
import styles from './PlayerPage.module.scss'

export function PlayerPVE({ data, loading, details, copy, zh }: { data?: PlayerPVEData; loading: boolean; details?: boolean; copy: PlayerCopy; zh: boolean }) {
  const { t } = useTranslation()
  const classOption = useMemo<EChartsCoreOption>(() => ({
    ...chartBase,
    legend: { top: 0, textStyle: { color: '#6f5b50' } },
    xAxis: { type: 'category', data: data?.infected_classes.map(item => infectedNames[item.class_id - 1]) ?? [], axisLabel: { color: '#806d62' }, axisLine: { lineStyle: { color: 'rgba(102,75,60,.22)' } } },
    yAxis: { type: 'value', axisLabel: { color: '#806d62' }, splitLine: { lineStyle: { color: 'rgba(102,75,60,.1)' } } },
    series: [
      { name: copy.kills, type: 'bar', data: data?.infected_classes.map(item => item.kills) ?? [], itemStyle: { color: palette[0], borderRadius: [5, 5, 0, 0] } },
      { name: copy.saves, type: 'bar', data: data?.infected_classes.map(item => item.saves) ?? [], itemStyle: { color: palette[1], borderRadius: [5, 5, 0, 0] } },
    ],
  }), [copy.kills, copy.saves, data])
  if (loading) return <Spin />
  if (!data) return null
  if (!details) return <div className={styles.sectionGrid}>
    <Section title={copy.combat}><MetricList items={[[t('commonKills'), data.common_kills], [t('specialKills'), data.special_kills], ['Tank', data.tank_kills], ['Witch', data.witch_kills], [t('damageSpecial'), data.damage_to_special], [zh ? '对 Tank 伤害' : 'Damage to Tank', data.damage_to_tank], [zh ? '对 Witch 伤害' : 'Damage to Witch', data.damage_to_witch]]} /></Section>
    <Section title={copy.survival}><MetricList items={[[t('deaths'), data.deaths], [t('incaps'), data.incapacitations], [copy.infectedDamage, data.damage_taken_infected], [copy.friendlyFireDealt, data.friendly_fire], [copy.friendlyFireTaken, data.friendly_fire_taken], [zh ? '倒地时长' : 'Incapacitated time', duration(data.incapacitated_seconds)], [zh ? '挂边时长' : 'Ledge time', duration(data.ledge_hanging_seconds)]]} /></Section>
    <Section title={copy.teamwork}><MetricList items={[[t('revives'), data.incap_revives], [copy.ledgeRescue, data.ledge_rescues], [copy.defib, data.defib_revives], [copy.received, data.rescues_received], [copy.medkitOthers, data.medkits_used_on_others], [copy.healingOthers, data.medkit_healing_others], [copy.blackWhite, data.black_white_teammates_restored]]} /></Section>
    <Section title={copy.supplies}><MetricList items={[[zh ? '对自己打包' : 'Medkits used on self', data.medkits_used_self], [zh ? '自我治疗量' : 'Self healing', data.medkit_healing_self], [zh ? '止痛药' : 'Pain pills', data.pills_used], [zh ? '肾上腺素' : 'Adrenaline', data.adrenaline_used], [zh ? '获得临时生命' : 'Temporary health received', data.temporary_health_received], [zh ? '燃烧弹药包' : 'Incendiary packs', data.incendiary_packs_deployed], [zh ? '高爆弹药包' : 'Explosive packs', data.explosive_packs_deployed], [copy.ammo, data.ammo_pile_uses]]} /></Section>
    <Section title={copy.progress}><MetricList items={[[t('chapters'), data.chapter_participations], [copy.survived, data.chapter_completions_alive], [copy.deadCompletion, data.chapter_completions_dead], [t('campaigns'), data.campaign_completions], [copy.ammo, data.ammo_pile_uses], [copy.objective, data.objective_interactions]]} /></Section>
    <Section title={copy.skills}><MetricList items={[[copy.tongue, data.melee_tongue_self_cuts], [copy.rocks, data.tank_rocks_destroyed], [copy.witchOneShot, data.witch_oneshots], [copy.witchSolo, data.witch_solo_kills], [zh ? 'Tank 遭遇' : 'Tank encounters', data.tank_encounters], [copy.tankParticipation, data.tank_kill_participations], [zh ? 'Witch 遭遇' : 'Witch encounters', data.witch_encounters], [copy.witchParticipation, data.witch_kill_participations], [copy.objective, data.objective_interactions]]} /></Section>
  </div>
  const equipmentColumns = [
    { title: copy.equipment, dataIndex: 'equipment_id', key: 'equipment', render: (id: number) => equipmentNames[id] ?? `#${id}` },
    { title: copy.actions, dataIndex: 'actions', key: 'actions', sorter: (a: PVEEquipment, b: PVEEquipment) => a.actions - b.actions },
    { title: t('commonKills'), dataIndex: 'common_kills', key: 'common', sorter: (a: PVEEquipment, b: PVEEquipment) => a.common_kills - b.common_kills },
    { title: t('specialKills'), dataIndex: 'special_kills', key: 'special', sorter: (a: PVEEquipment, b: PVEEquipment) => a.special_kills - b.special_kills },
    { title: copy.headshots, dataIndex: 'headshot_kills', key: 'headshots', sorter: (a: PVEEquipment, b: PVEEquipment) => a.headshot_kills - b.headshot_kills },
    { title: 'Tank', key: 'tank', render: (_: unknown, item: PVEEquipment) => `${numberFormat.format(item.tank_kills)} / ${numberFormat.format(item.damage_to_tank)}` },
    { title: 'Witch', key: 'witch', render: (_: unknown, item: PVEEquipment) => `${numberFormat.format(item.witch_kills)} / ${numberFormat.format(item.damage_to_witch)}` },
  ]
  const classColumns = [
    { title: zh ? '类型' : 'Class', key: 'class', render: (_: unknown, item: PVEInfectedClass) => infectedNames[item.class_id - 1] ?? `#${item.class_id}` },
    { title: copy.kills, dataIndex: 'kills', key: 'kills' },
    { title: copy.damage, dataIndex: 'damage', key: 'damage' },
    { title: copy.controlled, dataIndex: 'controls_received', key: 'controls' },
    { title: copy.controlTime, dataIndex: 'controlled_seconds', key: 'seconds', render: (value: number) => duration(value) },
    { title: copy.saves, dataIndex: 'saves', key: 'saves' },
  ]
  return <div className={styles.sectionGrid}>
    <Section title={copy.classes} wide><EChart ariaLabel={copy.classes} className={styles.chart} option={classOption} /><Table<PVEInfectedClass> className={styles.embeddedTable} columns={classColumns} dataSource={data.infected_classes} rowKey="class_id" pagination={false} size="small" scroll={{ x: 720 }} /></Section>
    <Section title={copy.equipment} wide><Table<PVEEquipment> className={styles.embeddedTable} columns={equipmentColumns} dataSource={data.equipment} rowKey="equipment_id" pagination={{ pageSize: 10, hideOnSinglePage: true }} size="small" scroll={{ x: 760 }} /></Section>
  </div>
}
