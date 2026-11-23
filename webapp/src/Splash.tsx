import { useMemo } from 'react'
import type { DayPhase, SkyState } from './sun'

// Kirish ekrani — prototipdagi sahna (osmon, quyosh, issiqxona, dala, nihol).
// Ranglar kunning qismiga qarab: tong, kunduz, kech, tun. Kunduzi quyosh
// haqiqiy yo'li bo'yicha (chiqishdan botishgacha) siljiydi.

interface Palette {
  sky: string[]
  sun: string
  glow: string
  cloud: number
  hills: [string, string, string, string, string]
  dots: [string, string]
  house: string
  houseLine: string
  door: string
  leaf: [string, string, string]
  title: string
  sub: string
}

const palettes: Record<DayPhase, Palette> = {
  morning: {
    sky: ['#F4A97F', '#F8CDA4', '#F6E4C9', '#D5E6BE', '#A3CC8A'],
    sun: '#FDB54E', glow: '#FFE1B0', cloud: 0.85,
    hills: ['#C6DFAA', '#9CCB84', '#84BD6E', '#6EAE5B', '#5E9F4D'],
    dots: ['#549D49', '#438D3C'],
    house: '#FFFAF2', houseLine: '#E8D3BE', door: '#F1E3D2',
    leaf: ['#2E6B3A', '#5BA25E', '#2E8B47'],
    title: '#12301A', sub: '#2B3F2C',
  },
  day: {
    sky: ['#7DBCE8', '#B5D9F0', '#DCEEF3', '#CFE6C2', '#9CCB86'],
    sun: '#FBBF24', glow: '#FDE68A', cloud: 0.85,
    hills: ['#BEDDA8', '#96C880', '#7FBA6B', '#6AAB59', '#5A9D4C'],
    dots: ['#4F9A47', '#3F8A3A'],
    house: '#FFFFFF', houseLine: '#B9D3E3', door: '#DCEAF2',
    leaf: ['#2E6B3A', '#5BA25E', '#2E8B47'],
    title: '#12301A', sub: '#1F3A24',
  },
  evening: {
    sky: ['#D9694F', '#EE9A6A', '#F4C995', '#CBD39C', '#8BB471'],
    sun: '#F97316', glow: '#FDBA74', cloud: 0.7,
    hills: ['#B1C98F', '#88B26C', '#71A25C', '#5D914D', '#4D8241'],
    dots: ['#45803C', '#376F31'],
    house: '#FFF1E4', houseLine: '#E2BFA3', door: '#F0D6C0',
    leaf: ['#2A5E34', '#4F9152', '#277A3E'],
    title: '#1A2A17', sub: '#33301F',
  },
  night: {
    sky: ['#0A1730', '#13284A', '#1D3A5C', '#22404A', '#1B3528'],
    sun: '#F4F1D0', glow: '#9FB3D9', cloud: 0.12,
    hills: ['#2E4A3A', '#284335', '#223B2F', '#1C3429', '#172D23'],
    dots: ['#1E3D2B', '#183424'],
    house: '#2A3E52', houseLine: '#4A6178', door: '#FFD27A',
    leaf: ['#3F7A4A', '#4D8A56', '#3D7A48'],
    title: '#F3F6F1', sub: '#C9D6CF',
  },
}

const HORIZON = 545

function sunPosition(sky: SkyState): { x: number; y: number } {
  const p = sky.sunProgress
  // Eng baland nuqta (peshin) — y≈335: sarlavha matniga tegmaydi.
  if (p != null) return { x: 30 + p * 330, y: HORIZON - Math.sin(Math.PI * p) * 210 }
  if (sky.phase === 'morning') return { x: 40, y: HORIZON + 8 } // hali chiqmagan — ufq ortida
  if (sky.phase === 'evening') return { x: 350, y: HORIZON + 8 } // botgan
  return { x: 300, y: 330 } // oy
}

// Yulduzlar — har safar bir xil joyda (tasodifiy emas, miltillamasin).
const stars = Array.from({ length: 34 }, (_, i) => ({
  x: (i * 97 + 23) % 390,
  y: 20 + ((i * 53) % 470),
  r: i % 5 === 0 ? 1.6 : 1,
  o: 0.4 + ((i * 37) % 60) / 100,
}))

export function Splash(props: { sky: SkyState; motto: string; label: string }) {
  const { phase } = props.sky
  const c = palettes[phase]
  const sun = useMemo(() => sunPosition(props.sky), [props.sky])
  const night = phase === 'night'
  const g = `sky-${phase}`

  return (
    <div className={`splash splash-${phase}`} role="status" aria-label={props.label}>
      <svg className="splash-scene" viewBox="0 0 390 844" preserveAspectRatio="xMidYMax slice" aria-hidden="true">
        <defs>
          <linearGradient id={g} x1="0" y1="0" x2="0" y2="1">
            {c.sky.map((col, i) => (
              <stop key={i} offset={[0, 0.38, 0.56, 0.7, 1][i]} stopColor={col} />
            ))}
          </linearGradient>
          <radialGradient id={`${g}-win`}>
            <stop offset="0" stopColor="#FFD27A" stopOpacity="0.55" />
            <stop offset="1" stopColor="#FFD27A" stopOpacity="0" />
          </radialGradient>
        </defs>
        <rect width="390" height="844" fill={`url(#${g})`} />

        {night && stars.map((s, i) => <circle key={i} cx={s.x} cy={s.y} r={s.r} fill="#FFFFFF" opacity={s.o} />)}

        {/* quyosh yoki oy (tepaliklardan oldin — botayotganda ortida qoladi) */}
        <g className="splash-sun">
          <circle cx={sun.x} cy={sun.y} r={66} fill={c.glow} opacity={night ? 0.18 : 0.35} />
          <circle cx={sun.x} cy={sun.y} r={night ? 30 : 40} fill={c.sun} />
          {night && <circle cx={sun.x + 13} cy={sun.y - 9} r={27} fill={c.sky[1]} />}
        </g>

        <g opacity={c.cloud}>
          <ellipse cx="70" cy="380" rx="58" ry="14" fill="#FFFFFF" opacity="0.95" />
          <ellipse cx="104" cy="368" rx="34" ry="14" fill="#FFFFFF" />
          <ellipse cx="236" cy="430" rx="52" ry="11" fill="#FFFFFF" opacity="0.8" />
        </g>

        <path d="M0 560 C 70 532, 150 546, 214 536 S 334 520, 390 532 L 390 844 L 0 844 Z" fill={c.hills[0]} />
        {/* issiqxona */}
        {night && <circle cx="195" cy="570" r="120" fill={`url(#${g}-win)`} />}
        <path d="M88 604 V532 Q 195 432 302 532 V604 Z" fill={c.house} opacity={night ? 0.9 : 0.85} />
        <path d="M88 532 Q 195 432 302 532" fill="none" stroke={c.houseLine} strokeWidth="3" />
        <path d="M124 604 V508 M159 604 V484 M195 604 V474 M231 604 V484 M266 604 V508" stroke={c.houseLine} strokeWidth="2.5" opacity="0.7" />
        <path d="M88 556 H302" stroke={c.houseLine} strokeWidth="2.5" opacity="0.7" />
        <rect x="180" y="566" width="30" height="38" rx="3" fill={c.door} />

        <path d="M0 618 C 90 594, 300 600, 390 616 L 390 844 L 0 844 Z" fill={c.hills[1]} />
        <path d="M0 660 C 110 636, 280 642, 390 658 L 390 844 L 0 844 Z" fill={c.hills[2]} />
        <path d="M0 712 C 120 690, 270 694, 390 710 L 390 844 L 0 844 Z" fill={c.hills[3]} />
        <path d="M0 772 C 120 754, 270 756, 390 770 L 390 844 L 0 844 Z" fill={c.hills[4]} />
        <g fill={c.dots[0]}>
          <ellipse cx="34" cy="644" rx="9" ry="5" /><ellipse cx="86" cy="638" rx="9" ry="5" /><ellipse cx="304" cy="638" rx="9" ry="5" /><ellipse cx="356" cy="644" rx="9" ry="5" />
          <ellipse cx="26" cy="690" rx="11" ry="6" /><ellipse cx="92" cy="682" rx="11" ry="6" /><ellipse cx="298" cy="682" rx="11" ry="6" /><ellipse cx="364" cy="690" rx="11" ry="6" />
        </g>
        <g fill={c.dots[1]}>
          <ellipse cx="20" cy="744" rx="13" ry="7" /><ellipse cx="96" cy="736" rx="13" ry="7" /><ellipse cx="294" cy="736" rx="13" ry="7" /><ellipse cx="370" cy="744" rx="13" ry="7" />
        </g>

        {/* nihol */}
        <g className="splash-plant">
          <path d="M195 790 V700" stroke={c.leaf[0]} strokeWidth="6" strokeLinecap="round" />
          <path d="M195 724 C 152 724, 128 700, 122 664 C 160 664, 192 684, 195 724 Z" fill={c.leaf[1]} />
          <path d="M195 706 C 236 704, 264 678, 270 640 C 230 640, 198 664, 195 706 Z" fill={c.leaf[2]} />
        </g>
        {!night && <path className="splash-drop" d="M150 626 c 0 0 -11 13 -11 21 a 11 11 0 0 0 22 0 c 0 -8 -11 -21 -11 -21 z" fill="#5AA9E6" />}
      </svg>

      <div className="splash-title">
        <div className="splash-logo">
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#FFFFFF" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <path d="M12 21V11" /><path d="M12 11c0-4 3-7 8-7 0 5-3 8-8 7z" /><path d="M12 14c0-3-2.5-5.5-7-5.5 0 4 2.5 6 7 5.5z" />
          </svg>
        </div>
        <div className="splash-name" style={{ color: c.title }}>Dehqon Bobo</div>
        <div className="splash-motto" style={{ color: c.sub }}>{props.motto}</div>
      </div>
    </div>
  )
}

/** Telegram sarlavha rangi — splash osmoniga mos (keyin ilova foniga qaytadi). */
export function skyTopColor(phase: DayPhase): string {
  return palettes[phase].sky[0]
}
