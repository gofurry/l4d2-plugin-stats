export interface FooterLink { id?: string; label: string; url: string }
export type SiteLanguage = 'zh-CN' | 'en'
export type SiteTheme = 'light' | 'dark'
export type SiteDocumentKey = 'introduction' | 'commands' | 'resources'
export interface SiteDocument { key: SiteDocumentKey; enabled: boolean; content_markdown: string; updated_at: number }
export interface Site { language: SiteLanguage; browser_title: string; theme: SiteTheme; footer_enabled: boolean; background_image_url: string; footer_links: FooterLink[]; steam_openid_enabled: boolean; a2s_refresh_seconds: number; site_documents: SiteDocumentKey[]; configured: boolean }
export interface SiteSettings { language: SiteLanguage; browser_title: string; theme: SiteTheme; footer_enabled: boolean; background_image_url: string; public_origin: string; steam_openid_enabled: boolean; a2s_refresh_seconds: number; a2s_jitter_seconds: number; a2s_retry_count: number; seo_enabled: boolean; seo_description: string; seo_image_url: string; footer_links: FooterLink[] }
export interface CoreOverview { total_players: number; active_players_7d: number; total_active_play_seconds: number; completed_pve_runs: number; completed_versus_runs: number }
export interface PVEOverview { common_kills: number; special_kills: number; tank_kills: number; witch_kills: number; rescues: number }
export interface VersusOverview { completed_matches: number; completed_halves: number; human_controlled_infected_kills: number; human_survivor_controls: number }
export interface Overview { core: CoreOverview; pve: PVEOverview; versus: VersusOverview; generated_at: string }
export interface ServerPlayer { name: string; score: number; duration_seconds: number }
export interface ServerRule { name: string; value: string }
export interface ServerStatus { server_id: string; display_name: string; address: string; online: boolean; stale: boolean; name?: string; map?: string; players: number; max_players: number; bots: number; latency_ms?: number; last_success_at?: string; checked_at: string; player_list?: ServerPlayer[]; rules?: ServerRule[] }
export interface ServerA2SState { available: boolean; status?: ServerStatus }
export interface GameServer { id?: string; display_name: string; address: string; enabled: boolean; sort_order: number }
export interface GameServerInput { display_name: string; address: string }
export interface AdminIdentity { username:string; created_at:number; updated_at:number; password_changed_at:number; monitor_enabled:boolean }
export interface PlayerSummary { steam_id:string; last_name:string; first_seen_at:number; last_seen_at:number; session_count:number; connected_seconds:number; active_play_seconds:number }
export interface PlayerActivityPoint { day:number; session_count:number; connected_seconds:number; active_play_seconds:number }
export interface PlayerServerActivity { server_key:string; session_count:number; active_play_seconds:number }
export interface PlayerActivity { timeline:PlayerActivityPoint[]; servers:PlayerServerActivity[] }
export interface PVEInfectedClass { class_id:number; kills:number; damage:number; controls_received:number; controlled_seconds:number; saves:number }
export interface PVEEquipment { equipment_id:number; actions:number; common_kills:number; special_kills:number; tank_kills:number; witch_kills:number; headshot_kills:number; damage_to_special:number; damage_to_tank:number; damage_to_witch:number }
export interface PlayerPVE {
  common_kills:number; special_kills:number; tank_kills:number; witch_kills:number; damage_to_special:number; damage_to_tank:number; damage_to_witch:number; damage_taken_infected:number; friendly_fire:number; friendly_fire_taken:number;
  incapacitations:number; deaths:number; revives:number; incap_revives:number; ledge_rescues:number; defib_revives:number; rescues_received:number;
  medkits_used:number; medkits_used_self:number; medkits_used_on_others:number; healing:number; medkit_healing_self:number; medkit_healing_others:number; pills_used:number; adrenaline_used:number; temporary_health_received:number;
  chapter_participations:number; chapter_completions:number; chapter_completions_alive:number; chapter_completions_dead:number; campaign_completions:number;
  melee_tongue_self_cuts:number; tank_rocks_destroyed:number; witch_oneshots:number; witch_solo_kills:number; tank_encounters:number; tank_kill_participations:number; witch_encounters:number; witch_kill_participations:number;
  incendiary_packs_deployed:number; explosive_packs_deployed:number; objective_interactions:number; ammo_pile_uses:number; incapacitated_seconds:number; ledge_hanging_seconds:number; black_white_teammates_restored:number;
  infected_classes:PVEInfectedClass[]; equipment:PVEEquipment[];
}
export interface VersusSurvivorClass { class_id:number; human_controller_kills:number; bot_controller_kills:number; damage_to_human_controllers:number; damage_to_bot_controllers:number }
export interface VersusInfectedClass { class_id:number; spawns:number; damage_to_human_survivors:number; damage_to_bot_survivors:number; human_survivor_incaps:number; bot_survivor_incaps:number; human_survivor_kills:number; bot_survivor_kills:number; human_survivor_controls:number; bot_survivor_controls:number; human_survivor_control_seconds:number; bot_survivor_control_seconds:number; human_survivor_ability_hits:number; bot_survivor_ability_hits:number; human_survivor_ability_damage:number; bot_survivor_ability_damage:number }
export interface PlayerVersus {
  survivor_common_kills:number; human_special_kills:number; bot_special_kills:number; human_tank_kills:number; bot_tank_kills:number; survivor_damage:number; survivor_damage_taken:number; survivor_friendly_fire:number; survivor_friendly_fire_taken:number; survivor_incapacitations:number; survivor_deaths:number; survivor_revives:number; survivor_incap_revives:number; survivor_ledge_rescues:number; survivor_defib_revives:number; survivor_rescues_received:number;
  survivor_medkits_self:number; survivor_medkits_others:number; survivor_healing_self:number; survivor_healing_others:number; survivor_pills:number; survivor_adrenaline:number; survivor_temporary_health:number; survivor_witch_kills:number; survivor_witch_damage:number;
  molotovs_thrown:number; pipe_bombs_thrown:number; vomit_jars_thrown:number; survivor_incendiary_packs:number; survivor_explosive_packs:number; survivor_tongue_self_cuts:number; survivor_tank_rocks_destroyed:number; survivor_witch_oneshots:number; survivor_witch_solo_kills:number;
  infected_spawns:number; damage_to_human_survivors:number; damage_to_bot_survivors:number; human_survivor_incaps:number; bot_survivor_incaps:number; human_survivor_kills:number; bot_survivor_kills:number; human_survivor_controls:number; human_survivor_control_seconds:number;
  survivor_classes:VersusSurvivorClass[]; infected_classes:VersusInfectedClass[];
}
export interface RankingEntry { rank:number; steam_id:string; player_name:string; value:number; active_play_seconds:number }
export interface PlayerIdentity { steam_id:string; name:string }
export interface RankingPage { metric:string; mode:string; items:RankingEntry[]; total:number; self?:RankingEntry; generated_at:string }
export interface PlayerSession { server_key:string; player_name:string; started_at:number; ended_at?:number; connected_seconds:number; active_play_seconds:number; status:string; disconnect_reason:string }
export interface PlayerChapter { server_key:string; mode_family:string; game_mode:string; map_name:string; side:string; started_at:number; ended_at?:number; active_play_seconds:number; status:string }
export interface Page<T> { items:T[]; next_cursor?:string }
export interface Announcement { id:string; title:string; content_markdown:string; created_at:number; updated_at:number }
export interface AnnouncementPage { items:Announcement[]; total:number; page:number; limit:number }
export interface AnnouncementInput { title:string; content_markdown:string }

interface Envelope<T> { data: T; request_id: string }
interface ErrorEnvelope { error?: { code?: string; message?: string } }
export class APIError extends Error { constructor(public status:number,public code:string,message:string){super(message)} }

let csrfToken = ''
async function request<T>(path:string,init:RequestInit={}):Promise<T>{
  const headers=new Headers(init.headers);headers.set('Accept','application/json');if(init.body)headers.set('Content-Type','application/json')
  const response=await fetch(path,{...init,headers,credentials:'same-origin'});let payload:Envelope<T>&ErrorEnvelope
  try{payload=await response.json() as Envelope<T>&ErrorEnvelope}catch{throw new APIError(response.status,'invalid_response',`HTTP ${response.status}`)}
  if(!response.ok)throw new APIError(response.status,payload.error?.code??'request_failed',payload.error?.message??`HTTP ${response.status}`)
  return payload.data
}
async function csrf():Promise<string>{if(csrfToken)return csrfToken;const result=await request<{token:string}>('/api/v1/admin/auth/csrf');csrfToken=result.token;return csrfToken}
async function adminWrite<T>(path:string,method:string,body?:unknown):Promise<T>{
  const execute=async()=>request<T>(path,{method,headers:{'X-Csrf-Token':await csrf()},body:body===undefined?undefined:JSON.stringify(body)})
  try{return await execute()}catch(error){if(error instanceof APIError&&error.code==='csrf_invalid'){csrfToken='';return execute()}throw error}
}

export const api={
  site:()=>request<Site>('/api/v1/site'),overview:()=>request<Overview>('/api/v1/dashboard/overview'),serverStatuses:()=>request<ServerStatus[]>('/api/v1/servers/status'),siteDocument:(key:SiteDocumentKey)=>request<SiteDocument>(`/api/v1/site-documents/${key}`),
  announcements:(page=1,limit=20,title='',year?:number)=>request<AnnouncementPage>(`/api/v1/announcements?${new URLSearchParams({page:String(page),limit:String(limit),...(title?{title}:{}),...(year?{year:String(year)}:{})})}`),announcementYears:()=>request<number[]>('/api/v1/announcements/years'),
  setupStatus:()=>request<{required:boolean;expires_at?:string}>('/api/v1/setup/status'),setupAdmin:(body:{setup_token:string;username:string;password:string})=>request('/api/v1/setup/admin',{method:'POST',body:JSON.stringify(body)}),
  login:(username:string,password:string)=>request('/api/v1/admin/auth/login',{method:'POST',body:JSON.stringify({username,password})}),logout:()=>adminWrite('/api/v1/admin/auth/logout','POST'),adminMe:()=>request<AdminIdentity>('/api/v1/admin/auth/me'),
  adminSite:()=>request<SiteSettings>('/api/v1/admin/site'),saveSite:(site:SiteSettings)=>adminWrite<SiteSettings>('/api/v1/admin/site','PUT',site),
  adminSiteDocuments:()=>request<SiteDocument[]>('/api/v1/admin/site-documents'),saveSiteDocument:(document:SiteDocument)=>adminWrite<SiteDocument>(`/api/v1/admin/site-documents/${document.key}`,'PUT',document),
  adminAnnouncements:(page=1,limit=50)=>request<AnnouncementPage>(`/api/v1/admin/announcements?${new URLSearchParams({page:String(page),limit:String(limit)})}`),createAnnouncement:(value:AnnouncementInput)=>adminWrite<Announcement>('/api/v1/admin/announcements','POST',value),updateAnnouncement:(id:string,value:AnnouncementInput)=>adminWrite<Announcement>(`/api/v1/admin/announcements/${id}`,'PUT',value),deleteAnnouncement:(id:string)=>adminWrite(`/api/v1/admin/announcements/${id}`,'DELETE'),
  servers:()=>request<GameServer[]>('/api/v1/admin/servers'),createServer:(server:GameServerInput)=>adminWrite<GameServer>('/api/v1/admin/servers','POST',server),updateServer:(id:string,server:GameServerInput)=>adminWrite<GameServer>(`/api/v1/admin/servers/${id}`,'PUT',server),setServerEnabled:(id:string,enabled:boolean)=>adminWrite<{enabled:boolean}>(`/api/v1/admin/servers/${id}/enabled`,'PATCH',{enabled}),moveServer:(id:string,direction:'up'|'down')=>adminWrite<{moved:boolean}>(`/api/v1/admin/servers/${id}/move`,'POST',{direction}),serverA2S:(id:string)=>request<ServerA2SState>(`/api/v1/admin/servers/${id}/a2s`),refreshServerA2S:(id:string)=>adminWrite<ServerStatus>(`/api/v1/admin/servers/${id}/a2s`,'POST'),deleteServer:(id:string)=>adminWrite(`/api/v1/admin/servers/${id}`,'DELETE'),
  updateAccount:(username:string)=>adminWrite('/api/v1/admin/account','PUT',{username}),updatePassword:(current_password:string,new_password:string)=>adminWrite('/api/v1/admin/account/password','PUT',{current_password,new_password}),
  steamIdentity:()=>request<{steam_id:string}|null>('/api/v1/steam/identity'),
  playerSummary:(id:string)=>request<PlayerSummary>(`/api/v1/players/${id}/summary`),
  playerActivity:(id:string,range:string,server='')=>request<PlayerActivity>(`/api/v1/players/${id}/activity?${new URLSearchParams({range,...(server?{server}:{})})}`),
  playerPVE:(id:string,range:string,server='',mode='')=>request<PlayerPVE>(`/api/v1/players/${id}/pve?${new URLSearchParams({range,...(server?{server}:{}),...(mode?{mode}:{})})}`),
  playerVersus:(id:string,range:string,server='')=>request<PlayerVersus>(`/api/v1/players/${id}/versus?${new URLSearchParams({range,...(server?{server}:{})})}`),
  playerSessions:(id:string,cursor='')=>request<Page<PlayerSession>>(`/api/v1/players/${id}/sessions?limit=20${cursor?`&cursor=${encodeURIComponent(cursor)}`:''}`),playerChapters:(id:string,cursor='')=>request<Page<PlayerChapter>>(`/api/v1/players/${id}/chapters?limit=20${cursor?`&cursor=${encodeURIComponent(cursor)}`:''}`),
  rankings:(params:{mode:string;metric:string;range:string;server?:string;players?:string[];subject?:string;page:number;limit?:number})=>{const query=new URLSearchParams({mode:params.mode,metric:params.metric,range:params.range,page:String(params.page),limit:String(params.limit??20)});if(params.server)query.set('server',params.server);if(params.players?.length)query.set('players',params.players.join(','));if(params.subject)query.set('subject',params.subject);return request<RankingPage>(`/api/v1/rankings?${query}`)},
  rankingServers:()=>request<string[]>('/api/v1/rankings/servers'),
  rankingPlayers:(query='')=>request<PlayerIdentity[]>(`/api/v1/rankings/players?${new URLSearchParams({q:query})}`),
}
export function resetCSRF(){csrfToken=''}
