// Quyosh chiqishi / botishi — NOAA "General Solar Position Calculations"
// (fractional year, equation of time, declination; zenith 90.833° —
// atmosfera sinishi va quyosh diski radiusi hisobga olingan).
// https://gml.noaa.gov/grad/solcalc/solareqns.PDF
//
// Hisob absolyut vaqtda (UTC), shuning uchun telefonning soat mintaqasi
// ahamiyatsiz: natija O'zbekiston uchun to'g'ri, telefon qayerda bo'lmasin.

export interface Place { lat: number; lon: number }

/** Toshkent — standart joy (O'zbekiston vaqti UTC+5, yozgi vaqt yo'q). */
export const TASHKENT: Place = { lat: 41.2995, lon: 69.2401 }

const rad = Math.PI / 180

export interface SunTimes { sunrise: Date; sunset: Date }

export function sunTimes(day: Date, place: Place = TASHKENT): SunTimes {
  // Kun — O'zbekiston sanasi bo'yicha (UTC+5), UTC yarim tunidan.
  const uz = new Date(day.getTime() + 5 * 3600_000)
  const y = uz.getUTCFullYear(), m = uz.getUTCMonth(), d = uz.getUTCDate()
  const startOfYear = Date.UTC(y, 0, 1)
  const n = Math.floor((Date.UTC(y, m, d) - startOfYear) / 86400_000) + 1
  const daysInYear = (y % 4 === 0 && y % 100 !== 0) || y % 400 === 0 ? 366 : 365
  const g = (2 * Math.PI / daysInYear) * (n - 1) // peshin atrofida, aniqlik yetarli

  const eqtime = 229.18 * (0.000075 + 0.001868 * Math.cos(g) - 0.032077 * Math.sin(g)
    - 0.014615 * Math.cos(2 * g) - 0.040849 * Math.sin(2 * g))
  const decl = 0.006918 - 0.399912 * Math.cos(g) + 0.070257 * Math.sin(g)
    - 0.006758 * Math.cos(2 * g) + 0.000907 * Math.sin(2 * g)
    - 0.002697 * Math.cos(3 * g) + 0.00148 * Math.sin(3 * g)

  const lat = place.lat * rad
  const cosHa = Math.cos(90.833 * rad) / (Math.cos(lat) * Math.cos(decl)) - Math.tan(lat) * Math.tan(decl)
  const ha = Math.acos(Math.min(1, Math.max(-1, cosHa))) / rad

  const midnightUTC = Date.UTC(y, m, d)
  const minutes = (sign: 1 | -1) => 720 - 4 * (place.lon + sign * ha) - eqtime
  return {
    sunrise: new Date(midnightUTC + minutes(1) * 60_000),
    sunset: new Date(midnightUTC + minutes(-1) * 60_000),
  }
}

export type DayPhase = 'morning' | 'day' | 'evening' | 'night'

// Chegaralar (daqiqa): tong — chiqishdan 40 daq. oldin … 2.5 soat keyin;
// kech — botishdan 2 soat oldin … 40 daq. keyin; orasi kunduz, qolgani tun.
const MORNING_BEFORE = 40, MORNING_AFTER = 150, EVENING_BEFORE = 120, EVENING_AFTER = 40

export interface SkyState {
  phase: DayPhase
  /** Kunduzgi yo'lning ulushi: 0 — chiqish, 1 — botish (tunda null). */
  sunProgress: number | null
  times: SunTimes
}

export function skyAt(now: Date, place: Place = TASHKENT): SkyState {
  const times = sunTimes(now, place)
  const t = now.getTime()
  const rise = times.sunrise.getTime(), set = times.sunset.getTime()
  const min = 60_000

  let phase: DayPhase = 'night'
  if (t >= rise - MORNING_BEFORE * min && t < rise + MORNING_AFTER * min) phase = 'morning'
  else if (t >= rise + MORNING_AFTER * min && t < set - EVENING_BEFORE * min) phase = 'day'
  else if (t >= set - EVENING_BEFORE * min && t < set + EVENING_AFTER * min) phase = 'evening'

  const p = (t - rise) / (set - rise)
  return { phase, sunProgress: p >= 0 && p <= 1 ? p : null, times }
}
