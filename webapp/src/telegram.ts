// Telegram Mini App SDK ustidan yupqa qatlam (telegram-web-app.js).
// https://core.telegram.org/bots/webapps

type ThemeParams = Record<string, string | undefined>

interface BottomButton {
  setText(text: string): BottomButton
  show(): BottomButton
  hide(): BottomButton
  enable(): BottomButton
  disable(): BottomButton
  showProgress(leaveActive?: boolean): BottomButton
  hideProgress(): BottomButton
  onClick(cb: () => void): BottomButton
  offClick(cb: () => void): BottomButton
}

interface WebApp {
  initData: string
  initDataUnsafe: { user?: { id: number; first_name?: string; language_code?: string } }
  colorScheme: 'light' | 'dark'
  themeParams: ThemeParams
  platform: string
  version: string
  ready(): void
  expand(): void
  close(): void
  MainButton: BottomButton
  BackButton: { show(): void; hide(): void; onClick(cb: () => void): void; offClick(cb: () => void): void }
  HapticFeedback?: { notificationOccurred(t: 'success' | 'error' | 'warning'): void; selectionChanged(): void }
  onEvent(event: string, cb: () => void): void
}

declare global {
  interface Window {
    Telegram?: { WebApp?: WebApp }
  }
}

export const tg: WebApp | undefined = typeof window !== 'undefined' ? window.Telegram?.WebApp : undefined

/** Telegram ichida ochilganmi (imzolangan initData bor). */
export function insideTelegram(): boolean {
  return !!tg?.initData
}

export function initTelegram(): void {
  if (!tg) return
  tg.ready()
  tg.expand()
}

export function haptic(kind: 'success' | 'error' | 'warning' | 'select'): void {
  const h = tg?.HapticFeedback
  if (!h) return
  if (kind === 'select') h.selectionChanged()
  else h.notificationOccurred(kind)
}

/** Ilova tili — Telegram'dagi tildan: ru → ru, en → en, qolgani o'zbekcha. */
export function telegramLanguage(): string {
  return tg?.initDataUnsafe.user?.language_code ?? navigator.language ?? 'uz'
}
