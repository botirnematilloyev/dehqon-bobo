import { useEffect, type ReactNode } from 'react'
import { Icon } from './icons'
import { tg } from './telegram'

// Telegram mijozida (platform != unknown) tizim "Orqaga" tugmasi; brauzerda
// sahifadagi tugma.
const nativeBack = !!tg && tg.platform !== 'unknown'

export function Screen(props: { title?: string; onBack?: () => void; children: ReactNode; bottom?: ReactNode }) {
  const { onBack } = props
  useEffect(() => {
    const app = tg
    if (!nativeBack || !app) return
    if (!onBack) {
      app.BackButton.hide()
      return
    }
    app.BackButton.show()
    app.BackButton.onClick(onBack)
    return () => app.BackButton.offClick(onBack)
  }, [onBack])

  return (
    <main className="screen">
      {(props.title || (onBack && !nativeBack)) && (
        <div className="header">
          {onBack && !nativeBack && (
            <button className="icon-btn" onClick={onBack} aria-label="back">
              <Icon name="back" size={20} strokeWidth={2.2} />
            </button>
          )}
          {props.title && <h2>{props.title}</h2>}
        </div>
      )}
      {props.children}
      {props.bottom && (
        <div className="bottom"><div className="bottom-inner">{props.bottom}</div></div>
      )}
    </main>
  )
}

export function Choice<T extends string | number>(props: {
  options: { value: T; label: string; icon?: string }[]
  value: T | null
  onChange: (v: T) => void
  label: string
}) {
  return (
    <div className="stack" role="group" aria-label={props.label}>
      <h3>{props.label}</h3>
      <div className="choice-grid">
        {props.options.map((o) => (
          <button key={String(o.value)} className={o.icon ? 'choice with-icon' : 'choice'} aria-label={o.label} aria-pressed={props.value === o.value} onClick={() => props.onChange(o.value)}>
            {o.icon && <span className="bubble"><Icon name={o.icon} size={26} strokeWidth={1.8} /></span>}
            <span>{o.label}</span>
            {props.value === o.value && <span className="tick"><Icon name="check" size={16} strokeWidth={3} /></span>}
          </button>
        ))}
      </div>
    </div>
  )
}

export function fmt(n: number, digits = 1): string {
  return Number.isInteger(n) ? String(n) : n.toFixed(digits).replace(/\.?0+$/, '')
}

export function dateOf(unix: number, lang: string): string {
  return new Date(unix * 1000).toLocaleDateString(lang === 'uz' ? 'uz-Latn' : lang, { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
}
