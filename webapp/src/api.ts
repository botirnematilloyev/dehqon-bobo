import type {
  Assessment, AuthResponse, BatchDetail, BatchSummary, Crop, Farm, FarmOverview, Fertilizer,
  District, Profile, Recipe, RecipeRequest, Region,
} from './types'

// Barcha so'rovlar nisbiy: prod'da ilova va API bitta domenda, dev'da Vite proxy.
const BASE = '/api/v1'

/** Server xatosi: `code` — barqaror kod (mantiq uchun), `message` — tarjima. */
export class ApiError extends Error {
  constructor(public status: number, public code: string, message: string) {
    super(message)
  }
}

type Fetch = typeof fetch

export class Api {
  private token: string | null = null

  constructor(
    private initData: () => string,
    private language: string,
    private fetchFn: Fetch = (...a) => fetch(...a),
  ) {}

  /** Mini App ichidan kirish: Telegram imzolagan initData → JWT. */
  async login(): Promise<AuthResponse> {
    const res = await this.raw<AuthResponse>('POST', '/auth/telegram/webapp', { init_data: this.initData() }, false)
    this.token = res.access_token
    return res
  }

  /**
   * Token muddati tugasa (401) bir marta qayta kiradi: initData Telegram
   * oynasi ochiq turgan vaqtda amal qiladi, refresh token saqlash shart emas.
   */
  private async call<T>(method: string, path: string, body?: unknown): Promise<T> {
    if (!this.token) await this.login()
    try {
      return await this.raw<T>(method, path, body, true)
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) {
        await this.login()
        return this.raw<T>(method, path, body, true)
      }
      throw e
    }
  }

  private async raw<T>(method: string, path: string, body: unknown, auth: boolean): Promise<T> {
    const headers: Record<string, string> = { 'Accept-Language': this.language }
    if (body !== undefined) headers['Content-Type'] = 'application/json'
    if (auth && this.token) headers.Authorization = `Bearer ${this.token}`
    const res = await this.fetchFn(BASE + path, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    })
    if (res.status === 204) return undefined as T
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      throw new ApiError(res.status, data.error ?? 'common.internal_error', data.message ?? res.statusText)
    }
    return data as T
  }

  me = () => this.call<Profile>('GET', '/me')
  regions = () => this.call<{ regions: Region[] }>('GET', '/regions').then((r) => r.regions)
  districts = (regionId: number) =>
    this.call<{ districts: District[] }>('GET', `/districts?region_id=${regionId}`).then((r) => r.districts)
  onboard = (body: { region_id: number; district_id?: number; water_source: string; system_type: string }) =>
    this.call<FarmOverview>('POST', '/me/onboarding', body)
  farms = () => this.call<{ farms: Farm[] }>('GET', '/farms').then((r) => r.farms)
  farm = (id: number) => this.call<FarmOverview>('GET', `/farms/${id}`)
  crops = () => this.call<{ crops: Crop[] }>('GET', '/catalog/crops').then((r) => r.crops)
  fertilizers = () => this.call<{ fertilizers: Fertilizer[] }>('GET', '/catalog/fertilizers').then((r) => r.fertilizers)

  preview = (farmId: number, body: RecipeRequest) => this.call<Recipe>('POST', `/farms/${farmId}/recipes/preview`, body)
  saveRecipe = (farmId: number, body: RecipeRequest) => this.call<BatchDetail>('POST', `/farms/${farmId}/recipes`, body)
  recipes = (farmId: number, limit = 20) =>
    this.call<{ recipes: BatchSummary[]; next_before_id: number | null }>('GET', `/farms/${farmId}/recipes?limit=${limit}`)
  recipe = (farmId: number, id: number) => this.call<BatchDetail>('GET', `/farms/${farmId}/recipes/${id}`)
  markMixed = (farmId: number, id: number) => this.call<BatchDetail>('POST', `/farms/${farmId}/recipes/${id}/mixed`)

  measure = (farmId: number, body: Record<string, unknown>) =>
    this.call<{ measurement: { id: number }; assessment: Assessment }>('POST', `/farms/${farmId}/measurements`, body)
}
