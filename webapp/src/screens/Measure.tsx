import { useState } from 'react'
import { ApiError } from '../api'
import { useApp } from '../ctx'
import { adviceText, pickName } from '../i18n'
import { haptic } from '../telegram'
import type { Assessment } from '../types'
import { Screen, fmt } from '../ui'
import { Icon } from '../icons'

const num = (s: string): number | undefined => {
  const v = Number(s.replace(',', '.'))
  return s.trim() === '' || Number.isNaN(v) ? undefined : v
}

/** O'lchov: EC, pH (substrat/gidroponikada drenaj ham) → baholash va tuzatish. */
export function Measure(props: { batchId?: number }) {
  const { api, t, lang, overview, back } = useApp()
  const [ecIn, setEcIn] = useState('')
  const [phIn, setPhIn] = useState('')
  const [ecDrain, setEcDrain] = useState('')
  const [drainPct, setDrainPct] = useState('')
  const [result, setResult] = useState<Assessment | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const drain = overview.farm.has_drain

  const submit = async () => {
    setBusy(true)
    setError(null)
    try {
      const body: Record<string, unknown> = {
        ec_in: num(ecIn), ph_in: num(phIn),
        ...(drain ? { ec_drain: num(ecDrain), drain_pct: num(drainPct) } : {}),
        ...(props.batchId ? { batch_id: props.batchId } : {}),
      }
      const res = await api.measure(overview.farm.id, body)
      setResult(res.assessment)
      haptic(res.assessment.advice.length ? 'warning' : 'success')
    } catch (e) {
      haptic('error')
      setError(e instanceof ApiError ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const field = (id: string, label: string, value: string, set: (v: string) => void, hint?: string) => (
    <div className="field">
      <label htmlFor={id}>{label}{hint ? ` · ${hint}` : ''}</label>
      <input id={id} className="input" inputMode="decimal" value={value} onChange={(e) => set(e.target.value)} />
    </div>
  )

  if (result) {
    const c = result.correction
    return (
      <Screen title={t.result} onBack={back} bottom={<button className="btn" onClick={back}>OK</button>}>
        <div className="card soft">
          <div className="row" style={{ gap: 24 }}>
            {result.ec && <div className="stat"><span className="muted small">EC · {t[`st_${result.ec.status}`]}</span><b>{fmt(result.ec.value, 2)}</b><span className="muted small">/ {fmt(result.ec.expected, 2)}</span></div>}
            {result.ph && <div className="stat"><span className="muted small">pH · {t[`st_${result.ph.status}`]}</span><b>{fmt(result.ph.value)}</b><span className="muted small">{fmt(result.ph.min)}–{fmt(result.ph.max)}</span></div>}
          </div>
        </div>
        {c && (
          <div className="card">
            <span className="bubble sm"><Icon name={c.action === 'add_water' ? 'drop' : c.action === 'add_salts' ? 'flask' : 'gauge'} size={20} /></span>
            {c.action === 'add_water' && <p>{t.addWater(c.water_l ?? 0)}</p>}
            {c.action === 'adjust_injector' && <p>{t.injector(c.injector_ratio ?? 0)}</p>}
            {c.action === 'add_salts' && (
              <>
                <p>{t.addSalts(Math.round((c.fraction ?? 0) * 100))}</p>
                <div className="list">
                  {c.items?.map((it) => (
                    <div key={it.code} className="list-row"><span>{pickName(it.name, lang) || it.code}</span><span className="amount">{fmt(it.grams, 3)} {t.grams}</span></div>
                  ))}
                </div>
              </>
            )}
          </div>
        )}
        {result.advice.map((a) => adviceText(lang, a)).filter(Boolean).map((text) => (
          <p key={text} className="note"><Icon name="alert" size={18} /><span>{text}</span></p>
        ))}
        {!c && result.advice.length === 0 && <p className="result-ok"><Icon name="check" size={22} strokeWidth={2.6} />{t.allGood}</p>}
      </Screen>
    )
  }

  const ready = num(ecIn) !== undefined || num(phIn) !== undefined || num(ecDrain) !== undefined
  return (
    <Screen title={t.measure} onBack={back}
      bottom={<button className="btn" disabled={!ready || busy} onClick={() => void submit()}>{t.save}</button>}>
      {field('ec_in', t.ecIn, ecIn, setEcIn)}
      {field('ph_in', t.phIn, phIn, setPhIn)}
      {drain && field('ec_drain', t.ecDrain, ecDrain, setEcDrain, t.optional)}
      {drain && field('drain_pct', t.drainPct, drainPct, setDrainPct, t.optional)}
      {error && <p className="error" role="alert">{error}</p>}
    </Screen>
  )
}
