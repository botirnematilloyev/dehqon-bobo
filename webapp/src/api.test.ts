import { describe, expect, it } from 'vitest'
import { Api, ApiError } from './api'
import { langFrom, pickName, warningText } from './i18n'

type Call = { url: string; init: RequestInit }

function fakeFetch(responses: Array<{ status: number; body?: unknown }>) {
  const calls: Call[] = []
  const fn = async (url: RequestInfo | URL, init?: RequestInit) => {
    calls.push({ url: String(url), init: init ?? {} })
    const r = responses.shift() ?? { status: 500 }
    return new Response(r.body === undefined ? null : JSON.stringify(r.body), { status: r.status })
  }
  return { fn: fn as typeof fetch, calls }
}

const auth = { status: 200, body: { access_token: 'tok-1', refresh_token: 'r', expires_at: 1, user: { id: 1 } } }

describe('Api', () => {
  it('logs in with initData, then sends the bearer token and language', async () => {
    const f = fakeFetch([auth, { status: 200, body: { farms: [] } }])
    const api = new Api(() => 'auth_date=1&hash=x', 'ru', f.fn)

    expect(await api.farms()).toEqual([])
    expect(f.calls[0].url).toBe('/api/v1/auth/telegram/webapp')
    expect(JSON.parse(String(f.calls[0].init.body))).toEqual({ init_data: 'auth_date=1&hash=x' })
    const h = f.calls[1].init.headers as Record<string, string>
    expect(h.Authorization).toBe('Bearer tok-1')
    expect(h['Accept-Language']).toBe('ru')
  })

  it('re-logs in once on 401', async () => {
    const f = fakeFetch([
      auth,
      { status: 401, body: { error: 'auth.invalid_token', message: 'x' } },
      { ...auth, body: { ...auth.body, access_token: 'tok-2' } },
      { status: 200, body: { farms: [{ id: 7 }] } },
    ])
    const api = new Api(() => 'd', 'uz', f.fn)
    expect(await api.farms()).toEqual([{ id: 7 }])
    expect((f.calls[3].init.headers as Record<string, string>).Authorization).toBe('Bearer tok-2')
  })

  it('surfaces the stable error code and localized message', async () => {
    const f = fakeFetch([auth, { status: 404, body: { error: 'recipe.profile_not_found', message: 'Рецепт не найден' } }])
    const api = new Api(() => 'd', 'ru', f.fn)
    const err = await api.preview(1, { crop: 'tomato', stage: 'seedling' }).catch((e) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err.code).toBe('recipe.profile_not_found')
    expect(err.message).toBe('Рецепт не найден')
  })
})

describe('i18n', () => {
  it('maps Telegram language codes', () => {
    expect(langFrom('ru-RU')).toBe('ru')
    expect(langFrom('en')).toBe('uz') // ingliz yo'q — o'zbekcha
    expect(langFrom('uz')).toBe('uz')
    expect(langFrom('de')).toBe('uz')
  })
  it('picks names with fallbacks', () => {
    expect(pickName({ uz: 'Pomidor', ru: 'Томат' }, 'ru')).toBe('Томат')
    expect(pickName({ uz: 'Pomidor' }, 'ru')).toBe('Pomidor')
  })
  it('knows every server warning code', () => {
    for (const code of ['profile_not_reviewed', 'water_is_estimate', 'water_salinity_above_crop_threshold', 'water_exceeds_target',
      'poor_ion_balance', 'no_acid_selected', 'bicarbonate_left', 'acid_over_target', 'infeasible_salts',
      'micro_fertilizer_missing', 'solubility_exceeded']) {
      for (const l of ['uz', 'ru'] as const) expect(warningText(l, code)).not.toBe(code)
    }
  })
})
