import { useEffect, useState } from 'react'
import { ApiError } from '../api'
import { useApp } from '../ctx'
import { pickName } from '../i18n'
import { haptic } from '../telegram'
import type { BatchDetail, BatchSummary, Crop } from '../types'
import { Screen, dateOf, fmt } from '../ui'
import { Icon, cropIcon } from '../icons'
import { RecipeView } from './Recipe'

export function Journal() {
  const { api, t, lang, overview, go, back } = useApp()
  const [list, setList] = useState<BatchSummary[] | null>(null)
  const [crops, setCrops] = useState<Crop[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api.recipes(overview.farm.id, 50).then((r) => setList(r.recipes)).catch((e) => setError(String(e?.message ?? e)))
    api.crops().then(setCrops).catch(() => undefined)
  }, [api, overview.farm.id])

  const cropName = (code: string) => pickName(crops.find((c) => c.code === code)?.name, lang) || code

  return (
    <Screen title={t.journal} onBack={back}>
      {error && <p className="error" role="alert">{error}</p>}
      {list && list.length === 0 && <p className="muted">{t.noRecipes}</p>}
      {list && list.length > 0 && (
        <div className="card">
          <div className="list">
            {list.map((b) => (
              <button key={b.id} className="list-row" onClick={() => go({ name: 'batch', id: b.id })}>
                <span className="row-icon">
                  <span className="bubble sm"><Icon name={cropIcon(b.crop)} size={22} strokeWidth={1.7} /></span>
                  <span className="stack" style={{ gap: 2 }}>
                    <b>{cropName(b.crop)} · {t[`st_${b.stage}`]}</b>
                    <span className="muted small">{dateOf(b.created_at, lang)} · {fmt(b.volume_l)} L · EC {fmt(b.target_ec, 2)}</span>
                  </span>
                </span>
                <span className={b.status === 'mixed' ? 'pill' : 'pill warn'}>{b.status === 'mixed' ? t.mixedDone : t.planned}</span>
              </button>
            ))}
          </div>
        </div>
      )}
    </Screen>
  )
}

export function BatchView(props: { id: number }) {
  const { api, t, overview, back, go } = useApp()
  const [batch, setBatch] = useState<BatchDetail | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api.recipe(overview.farm.id, props.id).then(setBatch).catch((e) => setError(String(e?.message ?? e)))
  }, [api, overview.farm.id, props.id])

  const mixed = async () => {
    setBusy(true)
    try {
      setBatch(await api.markMixed(overview.farm.id, props.id))
      haptic('success')
    } catch (e) {
      setError(e instanceof ApiError ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const bottom = batch && (batch.status === 'mixed'
    ? <button className="btn" onClick={() => go({ name: 'measure', batchId: batch.id })}>{t.measureNow}</button>
    : <button className="btn" disabled={busy} onClick={() => void mixed()}>{t.markMixed}</button>)

  return (
    <Screen title={t.journal} onBack={back} bottom={bottom}>
      {error && <p className="error" role="alert">{error}</p>}
      {batch && (
        <>
          <span className={batch.status === 'mixed' ? 'pill' : 'pill warn'}>{batch.status === 'mixed' ? t.mixedDone : t.planned}</span>
          <RecipeView recipe={batch.recipe} checklist={batch.status !== 'mixed'} />
        </>
      )}
    </Screen>
  )
}
