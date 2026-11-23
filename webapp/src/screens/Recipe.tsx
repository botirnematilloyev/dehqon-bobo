import { useEffect, useState } from 'react'
import { ApiError } from '../api'
import { useApp } from '../ctx'
import { pickName, warningText } from '../i18n'
import { haptic } from '../telegram'
import type { BatchDetail, Crop, Fertilizer, Recipe, RecipeRequest, Stage } from '../types'
import { Choice, Screen, fmt } from '../ui'
import { Icon, cropIcon, stageIcon } from '../icons'

const stages: Stage[] = ['seedling', 'vegetative', 'fruiting']
// Tavsiya etilgan tartib (Sonneveld & Voogt): avval fosfat kislota — P baribir kerak.
const preferredAcids = ['H3PO4_85', 'HNO3_38']

export function RecipeFlow() {
  const { api, t, lang, overview, back, go } = useApp()
  const [crops, setCrops] = useState<Crop[]>([])
  const [acidList, setAcidList] = useState<Fertilizer[]>([])
  const [crop, setCrop] = useState<string | null>(null)
  const [stage, setStage] = useState<Stage | null>(null)
  const [volume, setVolume] = useState(String(overview.plots[0]?.default_volume_l ?? 200))
  const [acids, setAcids] = useState<string[]>([])
  const [recipe, setRecipe] = useState<Recipe | null>(null)
  const [saved, setSaved] = useState<BatchDetail | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const direct = overview.farm.mixing_method === 'direct'

  useEffect(() => {
    api.crops().then(setCrops).catch((e) => setError(String(e?.message ?? e)))
    api.fertilizers().then((fs) => {
      const list = fs.filter((f) => f.kind === 'acid')
      setAcidList(list)
      // Suvda bikarbonat bo'lsa kislota oldindan tanlanadi.
      if ((overview.water?.hco3_mmol ?? 0) > 0.5) {
        setAcids(preferredAcids.filter((c) => list.some((f) => f.code === c)))
      }
    }).catch(() => undefined)
  }, [api, overview.water])

  const request = (): RecipeRequest => ({
    crop: crop!, stage: stage!, acids,
    ...(direct ? { method: 'direct' as const, volume_l: Number(volume) } : { method: 'ab' as const }),
    ...(overview.plots[0] ? { plot_id: overview.plots[0].id } : {}),
  })

  const run = async (fn: () => Promise<void>) => {
    setBusy(true)
    setError(null)
    try {
      await fn()
    } catch (e) {
      haptic('error')
      setError(e instanceof ApiError ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const calculate = () => run(async () => {
    const { plot_id: _plot, ...body } = request()
    setRecipe(await api.preview(overview.farm.id, body))
    haptic('success')
  })
  const save = () => run(async () => {
    setSaved(await api.saveRecipe(overview.farm.id, request()))
    haptic('success')
  })
  const mixed = () => run(async () => {
    setSaved(await api.markMixed(overview.farm.id, saved!.id))
    haptic('success')
  })

  // ---------- 2-bosqich: natija ----------
  if (recipe) {
    const bottom = saved
      ? saved.status === 'mixed'
        ? <button className="btn" onClick={() => go({ name: 'measure', batchId: saved.id })}>{t.measureNow}</button>
        : <button className="btn" disabled={busy} onClick={() => void mixed()}>{t.markMixed}</button>
      : <button className="btn" disabled={busy} onClick={() => void save()}>{t.saveToJournal}</button>
    return (
      <Screen title={pickName(recipe.profile.crop_name, lang)} onBack={saved ? back : () => setRecipe(null)} bottom={bottom}>
        {saved && <p className="pill" role="status">{saved.status === 'mixed' ? t.mixedDone : t.saved}</p>}
        <RecipeView recipe={recipe} checklist={!!saved && saved.status !== 'mixed'} />
        {error && <p className="error" role="alert">{error}</p>}
      </Screen>
    )
  }

  // ---------- 1-bosqich: tanlov ----------
  const ready = crop && stage && (!direct || Number(volume) > 0)
  return (
    <Screen title={t.mixSolution} onBack={back}
      bottom={<button className="btn" disabled={!ready || busy} onClick={() => void calculate()}>{t.calculate}</button>}>
      <Choice label={t.crop} value={crop} onChange={(v) => { setCrop(v); haptic('select') }}
        options={crops.map((c) => ({ value: c.code, label: pickName(c.name, lang), icon: cropIcon(c.code) }))} />
      {crop && (
        <Choice label={t.stage} value={stage} onChange={(v) => { setStage(v); haptic('select') }}
          options={stages.map((s) => ({ value: s, label: t[`st_${s}`], icon: stageIcon(s, crop) }))} />
      )}
      {crop && stage && direct && (
        <div className="field">
          <label htmlFor="volume">{t.tankVolume}</label>
          <input id="volume" className="input" inputMode="decimal" value={volume} onChange={(e) => setVolume(e.target.value.replace(',', '.'))} />
        </div>
      )}
      {crop && stage && acidList.length > 0 && (
        <div className="stack">
          <h3>{t.acids}</h3>
          <div className="choice-grid">
            {acidList.map((a) => {
              const on = acids.includes(a.code)
              return (
                <button key={a.code} className="choice" aria-pressed={on}
                  onClick={() => setAcids((s) => (on ? s.filter((x) => x !== a.code) : [...s, a.code]))}>
                  <span className="row" style={{ gap: 8 }}><Icon name="flask" size={18} />{pickName(a.name, lang)}</span>
                  {on && <span className="tick"><Icon name="check" size={16} strokeWidth={3} /></span>}
                </button>
              )
            })}
          </div>
        </div>
      )}
      {error && <p className="error" role="alert">{error}</p>}
    </Screen>
  )
}

/** Retsept: idishlar bo'yicha grammlar, kislota ml, EC/pH, ogohlantirishlar. */
export function RecipeView(props: { recipe: Recipe; checklist?: boolean }) {
  const { t, lang } = useApp()
  const r = props.recipe
  const [done, setDone] = useState<Record<string, boolean>>({})
  const direct = r.method === 'direct'

  const row = (key: string, label: string, amount: string) => (
    <label key={key} className="list-row" style={{ cursor: props.checklist ? 'pointer' : 'default' }}>
      <span className="row">
        {props.checklist && (
          <input type="checkbox" className="check" checked={!!done[key]}
            onChange={(e) => { setDone((d) => ({ ...d, [key]: e.target.checked })); haptic('select') }} />
        )}
        <span style={{ textDecoration: done[key] ? 'line-through' : undefined }}>{label}</span>
      </span>
      <span className="amount">{amount}</span>
    </label>
  )

  return (
    <>
      <div className="card soft">
        <div className="row" style={{ gap: 24 }}>
          <div className="stat"><span className="muted small">{t.expectedEC}</span><b>{fmt(r.ec.estimated, 2)}</b></div>
          <div className="stat"><span className="muted small">{t.phRange}</span><b>{fmt(r.ph.min)}–{fmt(r.ph.max)}</b></div>
          <div className="stat"><span className="muted small">L</span><b>{fmt(r.volume_l)}</b></div>
        </div>
        <span className={r.profile.is_reviewed ? 'pill' : 'pill warn'}>
          {r.profile.is_reviewed ? t.agronomReviewed : t.agronomNotReviewed}
        </span>
      </div>

      {direct && <p className="muted small">{t.oneTank}</p>}
      {r.tanks.filter((tk) => tk.items.length > 0).map((tk) => (
        <div className="card" key={tk.tank}>
          <h3 className="row"><span className="tank-badge">{direct ? (tk.tank === 'A' ? '1' : '2') : tk.tank}</span>{tk.tank === 'A' ? (direct ? t.group1 : t.tankA) : (direct ? t.group2 : t.tankB)}</h3>
          <div className="list">
            {tk.items.map((it) => row(`${tk.tank}-${it.code}`, pickName(it.name, lang) || it.code, `${fmt(it.grams, 3)} ${t.grams}`))}
          </div>
        </div>
      ))}

      {r.acids.length > 0 && (
        <div className="card">
          <h3 className="row"><span className="bubble sm"><Icon name="flask" size={20} /></span>{t.acidsTitle}</h3>
          <div className="list">
            {r.acids.map((a) => row(`acid-${a.code}`, pickName(a.name, lang) || a.code, `${fmt(a.ml)} ${t.ml}`))}
          </div>
        </div>
      )}

      {r.warnings.filter((w) => w.code !== 'profile_not_reviewed').map((w, i) => (
        <p key={i} className="note"><Icon name="alert" size={18} /><span>{warningText(lang, w.code)}{w.element ? ` (${w.element})` : ''}</span></p>
      ))}
    </>
  )
}
