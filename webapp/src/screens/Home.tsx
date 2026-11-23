import { useEffect, useState } from 'react'
import { useApp } from '../ctx'
import { tg } from '../telegram'
import type { BatchSummary, Crop } from '../types'
import { pickName } from '../i18n'
import { Screen, dateOf, fmt } from '../ui'
import { Icon, cropIcon } from '../icons'

export function Home() {
  const { api, t, lang, overview, go } = useApp()
  const [recent, setRecent] = useState<BatchSummary[] | null>(null)
  const [crops, setCrops] = useState<Crop[]>([])
  const name = tg?.initDataUnsafe.user?.first_name

  useEffect(() => {
    api.recipes(overview.farm.id, 3).then((r) => setRecent(r.recipes)).catch(() => setRecent([]))
    api.crops().then(setCrops).catch(() => undefined)
  }, [api, overview.farm.id])

  const cropName = (code: string) => pickName(crops.find((c) => c.code === code)?.name, lang) || code
  const water = overview.water

  return (
    <Screen>
      <div className="stack">
        <p className="muted small">{t.hello}{name ? `, ${name}` : ''}</p>
        <h1>Dehqon Bobo</h1>
      </div>

      <button className="hero" onClick={() => go({ name: 'recipe' })}>
        <span className="bubble light"><Icon name="flask" size={26} /></span>
        <span className="text">
          <h2>{t.mixSolution}</h2>
          <span className="sub">{t.mixSolutionHint}</span>
        </span>
        <Icon name="chevron" size={22} />
      </button>

      <div className="tiles">
        <button className="tile" onClick={() => go({ name: 'measure' })}>
          <span className="bubble sm"><Icon name="gauge" size={20} /></span>
          <span>{t.measure}</span>
          <span className="muted small">EC · pH</span>
        </button>
        <button className="tile" onClick={() => go({ name: 'journal' })}>
          <span className="bubble sm"><Icon name="chart" size={20} /></span>
          <span>{t.journal}</span>
          <span className="muted small">{t.lastRecipes}</span>
        </button>
      </div>

      {water && (
        <div className="card soft">
          <div className="row between">
            <h3 className="row" style={{ gap: 8 }}><Icon name="drop" size={20} />{t.water}</h3>
            {water.is_estimate && <span className="pill warn">{t.waterEstimate}</span>}
          </div>
          <div className="row" style={{ gap: 24 }}>
            <div className="stat"><span className="muted small">EC</span><b>{fmt(water.ec, 2)}</b></div>
            <div className="stat"><span className="muted small">HCO₃, mmol/L</span><b>{fmt(water.hco3_mmol, 1)}</b></div>
          </div>
        </div>
      )}

      <div className="card">
        <div className="row between">
          <h3>{t.lastRecipes}</h3>
          {recent && recent.length > 0 && (
            <button className="pill" style={{ border: 0, cursor: 'pointer' }} onClick={() => go({ name: 'journal' })}>{t.all}</button>
          )}
        </div>
        {recent && recent.length === 0 && <p className="muted small">{t.noRecipes}</p>}
        <div className="list">
          {recent?.map((b) => (
            <button key={b.id} className="list-row" onClick={() => go({ name: 'batch', id: b.id })}>
              <span className="row-icon">
                <span className="bubble sm"><Icon name={cropIcon(b.crop)} size={22} strokeWidth={1.7} /></span>
                <span className="stack" style={{ gap: 2 }}>
                  <b>{cropName(b.crop)} · {t[`st_${b.stage}`]}</b>
                  <span className="muted small">{dateOf(b.created_at, lang)} · {fmt(b.volume_l)} L</span>
                </span>
              </span>
              <span className={b.status === 'mixed' ? 'pill' : 'pill warn'}>{b.status === 'mixed' ? t.mixedDone : t.planned}</span>
            </button>
          ))}
        </div>
      </div>
    </Screen>
  )
}
