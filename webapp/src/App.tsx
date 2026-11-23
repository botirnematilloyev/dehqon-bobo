import { useCallback, useEffect, useMemo, useState } from 'react'
import { Api, ApiError } from './api'
import { Ctx, type AppCtx, type Route } from './ctx'
import { langFrom, strings } from './i18n'
import { insideTelegram, telegramLanguage, tg } from './telegram'
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

  if (boot.state === 'loading') {
    return <Screen><div className="center"><p className="muted">{t.loading}</p></div></Screen>
  }
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
