// API javoblari (docs/openapi.yaml dan, ilovada ishlatiladiganlari).

export type I18nName = Record<string, string>
export type Stage = 'seedling' | 'vegetative' | 'fruiting'
export type Method = 'direct' | 'ab'

export interface AuthResponse {
  access_token: string
  refresh_token: string
  expires_at: number
  user: { id: number; username: string; full_name: string | null }
}

export interface Profile {
  id: number
  full_name: string | null
  locale: string
  region_id: number | null
  onboarded: boolean
}

export interface Region { id: number; name: I18nName }

export interface Farm {
  id: number
  name: string | null
  system_type: string
  has_drain: boolean
  mixing_method: Method
  injector_ratio: number | null
  stock_tank_l: number | null
}

export interface Plot { id: number; name: string; area_m2: number | null; default_volume_l: number | null }

export interface WaterAnalysis {
  id: number
  source: string
  is_estimate: boolean
  ec: number
  hco3_mmol: number
}

export interface FarmOverview { farm: Farm; water: WaterAnalysis | null; plots: Plot[] }

export interface Crop { code: string; name: I18nName; salinity_ecw_threshold: number | null }

export interface Fertilizer { code: string; name: I18nName; kind: 'macro' | 'micro' | 'acid'; formula: string }

export interface Nutrients { no3_n: number; nh4_n: number; p: number; k: number; ca: number; mg: number; s: number; cl: number }

export interface RecipeItem { code: string; name: I18nName; grams: number }

export interface Warning { code: string; element?: string; value?: number }

export interface Recipe {
  profile: { id: number; crop: string; crop_name: I18nName; stage: Stage; is_reviewed: boolean; source: string | null }
  method: Method
  volume_l: number
  stock_tank_l: number | null
  injector_ratio: number | null
  tanks: { tank: 'A' | 'B'; items: RecipeItem[]; total_grams: number }[]
  acids: { code: string; name: I18nName; tank: string; ml: number; element: string }[]
  target_mg_l: Nutrients
  achieved_mg_l: Nutrients
  deviation_mg_l: Nutrients
  ec: { target: number | null; estimated: number }
  ph: { min: number; max: number }
  hco3_left_mmol: number
  warnings: Warning[]
  calc_version: string
}

export interface BatchSummary {
  id: number
  crop: string
  stage: Stage
  plot_id: number | null
  method: Method
  volume_l: number
  target_ec: number
  status: 'planned' | 'mixed'
  created_at: number
  mixed_at: number | null
}

export interface BatchDetail extends BatchSummary {
  items: { code: string; tank: string; grams?: number; ml?: number }[]
  recipe: Recipe
}

export interface Assessment {
  ec?: { value: number; expected: number; deviation_pct: number; status: 'low' | 'ok' | 'high' }
  ph?: { value: number; min: number; max: number; status: 'low' | 'ok' | 'high' }
  drain?: { ec_excess?: number; ec_status?: string; pct?: number; pct_status?: string }
  correction?: { action: 'add_water' | 'add_salts' | 'adjust_injector'; water_l?: number; fraction?: number; items?: RecipeItem[]; injector_ratio?: number }
  advice: string[]
}

export interface RecipeRequest {
  crop: string
  stage: Stage
  method?: Method
  volume_l?: number
  acids?: string[]
  plot_id?: number
}
