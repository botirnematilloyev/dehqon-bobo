import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Api, ApiError } from './api'
import { Ctx, type AppCtx, type Route } from './ctx'
import { langFrom, strings } from './i18n'
import { insideTelegram, setChrome, telegramLanguage, tg } from './telegram'
import { Splash, skyTopColor } from './Splash'
import { skyAt } from './sun'
import type { FarmOverview } from './types'
import { Screen } from './ui'
import { Onboarding } from './screens/Onboarding'
import { Home } from './screens/Home'
import { RecipeFlow } from './screens/Recipe'
import { Journal, BatchView } from './screens/Journal'
import { Measure } from './screens/Measure'

type Boot =
  | { state: 'loading' }
  | { state: 'outside' }
  | { state: 'error'; message: string }
  | { state: 'onboarding' }
  | { state: 'ready'; overview: FarmOverview }

export function App() {
  const lang = langFrom(telegramLanguage())
  const t = strings[lang]
  const api = useMemo(() => new Api(() => tg?.initData ?? '', lang), [lang])
  const [boot, setBoot] = useState<Boot>({ state: 'loading' })
  const [stack, setStack] = useState<Route[]>([{ name: 'home' }])
  // Splash kamida shuncha ko'rinadi (tez internetda ham "miltillab" o'tmasin).
  const SPLASH_MIN_MS = 1200
  const sky = useMemo(() => {
    const real = skyAt(new Date())
    // Faqat dev: ?sky=morning|day|evening|night&p=0.3 — sahnalarni ko'rish uchun.
    if (import.meta.env.DEV) {
      const q = new URLSearchParams(location.search)
      const phase = q.get('sky') as typeof real.phase | null
      if (phase) return { ...real, phase, sunProgress: q.has('p') ? Number(q.get('p')) : null }
    }
    return real
  }, [])
  const holdSplash = import.meta.env.DEV && new URLSearchParams(location.search).has('sky')
  const [splash, setSplash] = useState<'on' | 'leaving' | 'off'>('on')
  const shownAt = useRef(Date.now())

  useEffect(() => {
    if (splash === 'on') setChrome(skyTopColor(sky.phase))
  }, [splash, sky.phase])

  useEffect(() => {
    if (boot.state === 'loading' || splash !== 'on' || holdSplash) return
    const wait = Math.max(0, SPLASH_MIN_MS - (Date.now() - shownAt.current))
    const t1 = setTimeout(() => setSplash('leaving'), wait)
    const t2 = setTimeout(() => {
      setSplash('off')
      setChrome(tg?.colorScheme === 'dark' ? (tg.themeParams.secondary_bg_color ?? '#0F1613') : '#F3F6F1')
    }, wait + 300)
    return () => { clearTimeout(t1); clearTimeout(t2) }
  }, [boot.state, splash, holdSplash])

  const loadFarm = useCallback(async () => {
    const farms = await api.farms()
    if (farms.length === 0) {
      setBoot({ state: 'onboarding' })
      return
    }
    setBoot({ state: 'ready', overview: await api.farm(farms[0].id) })
  }, [api])

  const start = useCallback(async () => {
    if (!insideTelegram()) {
      setBoot({ state: 'outside' })
      return
    }
    setBoot({ state: 'loading' })
    try {
      await api.login()
      await loadFarm()
    } catch (e) {
      setBoot({ state: 'error', message: e instanceof ApiError ? e.message : String(e) })
    }
  }, [api, loadFarm])

  useEffect(() => {
    void start()
  }, [start])

  const splashEl = splash !== 'off' && (
    <div className={splash === 'leaving' ? 'splash-wrap leaving' : 'splash-wrap'}>
      <Splash sky={sky} motto={t.motto} label={t.loading} />
    </div>
  )

  if (boot.state === 'loading') return <>{splashEl}</>
  return <>{splashEl}{content()}</>

  // Asosiy ekran (splash ustida/ostida).
  function content() {
    if (boot.state === 'loading') return null
    if (boot.state === 'outside') {
      return (
        <Screen>
          <div className="center"><div className="stack">
            <h1>Dehqon Bobo</h1>
            <h2>{t.openInTelegram}</h2>
            <p className="muted">{t.openInTelegramHint}</p>
          </div></div>
        </Screen>
      )
    }
    if (boot.state === 'error') {
      return (
        <Screen bottom={<button className="btn" onClick={() => void start()}>{t.retry}</button>}>
          <div className="center"><p className="error" role="alert">{boot.message}</p></div>
        </Screen>
      )
    }
    if (boot.state === 'onboarding') {
      return <Onboarding api={api} lang={lang} onDone={(overview) => setBoot({ state: 'ready', overview })} />
    }

    const route = stack[stack.length - 1]
    const ctx: AppCtx = {
      api, lang, t,
      overview: boot.overview,
      reloadFarm: async () => setBoot({ state: 'ready', overview: await api.farm(boot.overview.farm.id) }),
      go: (r) => setStack((s) => [...s, r]),
      back: () => setStack((s) => (s.length > 1 ? s.slice(0, -1) : s)),
    }

    return (
      <Ctx.Provider value={ctx}>
        {route.name === 'home' && <Home />}
        {route.name === 'recipe' && <RecipeFlow />}
        {route.name === 'journal' && <Journal />}
        {route.name === 'batch' && <BatchView id={route.id} />}
        {route.name === 'measure' && <Measure batchId={route.batchId} />}
      </Ctx.Provider>
    )
  }
}
