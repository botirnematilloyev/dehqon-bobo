import { describe, expect, it } from 'vitest'
import { skyAt, sunTimes } from './sun'

// O'zbekiston vaqti (UTC+5) → Date.
const uz = (iso: string) => new Date(iso + '+05:00')
const hm = (d: Date) => {
  const t = new Date(d.getTime() + 5 * 3600_000)
  return t.getUTCHours() * 60 + t.getUTCMinutes()
}
const at = (h: number, m: number) => h * 60 + m

describe('sunTimes (Toshkent)', () => {
  // Mustaqil hisob: US Naval Observatory "Almanac for Computers" algoritmi
  // (boshqa formula, Python'da), Toshkent 41.2995°N 69.2401°E, UTC+5.
  // Farq ≤ 3 daqiqa.
  const cases: [string, number, number][] = [
    ['2026-06-21T12:00:00', at(4, 50), at(20, 0)],
    ['2026-12-21T12:00:00', at(7, 45), at(16, 57)],
    ['2026-03-20T12:00:00', at(6, 28), at(18, 35)],
    ['2026-09-22T12:00:00', at(6, 10), at(18, 21)],
    ['2026-11-23T12:00:00', at(7, 20), at(16, 58)],
  ]
  for (const [day, rise, set] of cases) {
    it(day.slice(0, 10), () => {
      const t = sunTimes(uz(day))
      expect(Math.abs(hm(t.sunrise) - rise)).toBeLessThanOrEqual(3)
      expect(Math.abs(hm(t.sunset) - set)).toBeLessThanOrEqual(3)
    })
  }

  it('does not depend on the phone time zone or time of day', () => {
    const a = sunTimes(uz('2026-11-23T00:30:00'))
    const b = sunTimes(uz('2026-11-23T23:30:00'))
    expect(a.sunrise.getTime()).toBe(b.sunrise.getTime())
  })
})

describe('skyAt', () => {
  it('splits the day into four phases around sunrise and sunset', () => {
    // 23-noyabr: chiqish 07:20, botish 16:59.
    expect(skyAt(uz('2026-11-23T03:00:00')).phase).toBe('night')
    expect(skyAt(uz('2026-11-23T08:00:00')).phase).toBe('morning')
    expect(skyAt(uz('2026-11-23T12:30:00')).phase).toBe('day')
    expect(skyAt(uz('2026-11-23T16:30:00')).phase).toBe('evening')
    expect(skyAt(uz('2026-11-23T21:00:00')).phase).toBe('night')
  })

  it('reports how far the sun has travelled', () => {
    const noon = skyAt(uz('2026-11-23T12:17:00')).sunProgress!
    expect(noon).toBeGreaterThan(0.45)
    expect(noon).toBeLessThan(0.55)
    expect(skyAt(uz('2026-11-23T23:00:00')).sunProgress).toBeNull()
  })
})
