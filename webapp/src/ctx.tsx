import { createContext, useContext } from 'react'
import type { Api } from './api'
import type { Lang, Strings } from './i18n'
import type { FarmOverview } from './types'

export type Route =
  | { name: 'home' }
  | { name: 'recipe' }
  | { name: 'journal' }
  | { name: 'batch'; id: number }
  | { name: 'measure'; batchId?: number }

export interface AppCtx {
  api: Api
  lang: Lang
  t: Strings
  overview: FarmOverview
  reloadFarm: () => Promise<void>
  go: (r: Route) => void
  back: () => void
}

export const Ctx = createContext<AppCtx | null>(null)

export function useApp(): AppCtx {
  const c = useContext(Ctx)
  if (!c) throw new Error('useApp outside provider')
  return c
}
