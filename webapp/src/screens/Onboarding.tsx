import { useEffect, useState } from 'react'
import { ApiError, type Api } from '../api'
import { pickName, strings, type Lang } from '../i18n'
import { haptic } from '../telegram'
import type { District, FarmOverview, Region } from '../types'
import { Choice, Screen } from '../ui'
import { Icon } from '../icons'

const sources = ['well', 'canal', 'tap', 'filter'] as const
const systems = ['greenhouse_soil', 'substrate', 'hydroponics', 'open_field'] as const

// Birinchi sozlash: viloyat, suv manbai, tizim. Suv tahlili server
// tomonida manba bo'yicha taxminiy yoziladi (keyin aniqlashtiriladi).
export function Onboarding(props: { api: Api; lang: Lang; onDone: (o: FarmOverview) => void }) {
  const t = strings[props.lang]
  const [regions, setRegions] = useState<Region[]>([])
  const [region, setRegion] = useState<number | null>(null)
  const [districts, setDistricts] = useState<District[]>([])
  const [district, setDistrict] = useState<number | null>(null)
  const [source, setSource] = useState<string | null>(null)
  const [system, setSystem] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    props.api.regions().then(setRegions).catch((e) => setError(String(e?.message ?? e)))
  }, [props.api])

  // Viloyat tanlanganda uning tuman va shaharlari (alifbo bo'yicha).
  useEffect(() => {
    setDistrict(null)
    setDistricts([])
    if (region == null) return
    let alive = true
    props.api.districts(region).then((list) => {
      if (!alive) return
      setDistricts([...list].sort((a, b) => pickName(a.name, props.lang).localeCompare(pickName(b.name, props.lang), props.lang)))
    }).catch((e) => setError(String(e?.message ?? e)))
    return () => { alive = false }
  }, [props.api, props.lang, region])

  const submit = async () => {
    if (region == null || district == null || !source || !system) return
    setBusy(true)
    setError(null)
    try {
      const overview = await props.api.onboard({ region_id: region, district_id: district, water_source: source, system_type: system })
      haptic('success')
      props.onDone(overview)
    } catch (e) {
      haptic('error')
      setError(e instanceof ApiError ? e.message : String(e))
      setBusy(false)
    }
  }

  return (
    <Screen
      bottom={
        <button className="btn" disabled={busy || region == null || district == null || !source || !system} onClick={() => void submit()}>
          {t.continue}
        </button>
      }
    >
      <div className="stack">
        <span className="bubble"><Icon name="sprout" size={26} /></span>
        <h1>{t.onbTitle}</h1>
        <p className="muted">{t.onbHint}</p>
      </div>
      <div className="field">
        <label htmlFor="region" className="row" style={{ gap: 6 }}><Icon name="pin" size={16} />{t.region}</label>
        <select id="region" className="input" value={region ?? ''} onChange={(e) => setRegion(Number(e.target.value))}>
          <option value="" disabled>—</option>
          {regions.map((r) => (
            <option key={r.id} value={r.id}>{pickName(r.name, props.lang)}</option>
          ))}
        </select>
      </div>
      <div className="field">
        <label htmlFor="district">{t.district}</label>
        <select id="district" className="input" value={district ?? ''} disabled={region == null || districts.length === 0}
          onChange={(e) => setDistrict(Number(e.target.value))}>
          <option value="" disabled>—</option>
          {districts.map((d) => (
            <option key={d.id} value={d.id}>{pickName(d.name, props.lang)}</option>
          ))}
        </select>
      </div>
      <Choice label={t.waterSource} value={source} onChange={setSource}
        options={sources.map((s) => ({ value: s, label: t[`ws_${s}`], icon: s }))} />
      <Choice label={t.system} value={system} onChange={setSystem}
        options={systems.map((s) => ({ value: s, label: t[`sys_${s}`], icon: s }))} />
      {error && <p className="error" role="alert">{error}</p>}
    </Screen>
  )
}
