import type { ReactNode } from 'react'

// Ikonkalar prototipdan (Design canvas, Main.dc.html) — 24×24, chiziqli.
// Ekinlar rangli (to'ldirilgan), qolganlari `currentColor`.

const line = (children: ReactNode) => ({ children, fill: 'none' as const })

const paths: Record<string, { children: ReactNode; fill?: string; stroke?: string }> = {
  sprout: line(<><path d="M12 21V11" /><path d="M12 11c0-4 3-7 8-7 0 5-3 8-8 7z" /><path d="M12 14c0-3-2.5-5.5-7-5.5 0 4 2.5 6 7 5.5z" /></>),
  flask: line(<><path d="M9 3h6" /><path d="M10 3v6L5 18a2 2 0 0 0 1.8 3h10.4a2 2 0 0 0 1.8-3l-5-9V3" /><path d="M7.5 14h9" /></>),
  chart: line(<><path d="M4 19V5" /><path d="M4 19h16" /><path d="M8 15l3-4 3 2 5-6" /></>),
  gauge: line(<><path d="M4 17a8 8 0 1 1 16 0" /><path d="M12 17l4-5" /><circle cx="12" cy="17" r="1.2" /></>),
  drop: line(<path d="M12 3s-6 6.5-6 11a6 6 0 0 0 12 0c0-4.5-6-11-6-11z" />),
  alert: line(<><circle cx="12" cy="12" r="9" /><path d="M12 8v5M12 16h.01" /></>),
  check: line(<path d="M5 12l5 5 9-10" />),
  chevron: line(<path d="M9 18l6-6-6-6" />),
  back: line(<path d="M15 18l-6-6 6-6" />),
  pin: line(<><path d="M12 21s-7-6.2-7-11a7 7 0 0 1 14 0c0 4.8-7 11-7 11z" /><circle cx="12" cy="10" r="2.5" /></>),
  // suv manbalari
  well: line(<><path d="M4 10h16" /><path d="M6 10v10h12V10" /><path d="M8 10V6h8v4" /><path d="M12 6V3" /><path d="M6 15h12" /></>),
  canal: line(<><path d="M2 8c2 0 2-1.5 4-1.5S8 8 10 8s2-1.5 4-1.5S16 8 18 8s2-1.5 4-1.5" /><path d="M2 13c2 0 2-1.5 4-1.5S8 13 10 13s2-1.5 4-1.5S16 13 18 13s2-1.5 4-1.5" /><path d="M2 18c2 0 2-1.5 4-1.5S8 18 10 18s2-1.5 4-1.5S16 18 18 18s2-1.5 4-1.5" /></>),
  tap: line(<><path d="M4 9h9a3 3 0 0 1 3 3v2" /><path d="M4 6v6" /><path d="M9 6V4h-3" /><path d="M16 18c0 0-1.5 1.8-1.5 2.7a1.5 1.5 0 0 0 3 0C17.5 19.8 16 18 16 18z" /></>),
  filter: line(<path d="M4 4h16l-6 8v6l-4 2v-8z" />),
  // tizimlar
  greenhouse_soil: line(<><path d="M3 20V12a9 9 0 0 1 18 0v8z" /><path d="M12 3v17M7 6v14M17 6v14" /></>),
  substrate: line(<><path d="M6 13h12l-1.5 8h-9z" /><path d="M12 13V8" /><path d="M12 8c0-2.5 2-4 5-4 0 3-2 4-5 4z" /><path d="M12 10c0-2-1.6-3.5-4.5-3.5 0 2.5 1.6 3.5 4.5 3.5z" /></>),
  hydroponics: line(<><path d="M3 15h18" /><path d="M3 15v4h18v-4" /><path d="M8 15v-5M16 15v-5" /><path d="M8 10c0-2 1.4-3 3.5-3-.2 2-1.5 3-3.5 3zM16 10c0-2-1.4-3-3.5-3 .2 2 1.5 3 3.5 3z" /></>),
  open_field: line(<><circle cx="17" cy="6" r="2.5" /><path d="M2 20c4-3 8-3 10-3s6 0 10 3" /><path d="M4 16c3-2 6-2.5 8-2.5s5 .5 8 2.5" /></>),
  // ekinlar (rangli)
  tomato: { fill: 'none', stroke: '#B3261E', children: <><circle cx="12" cy="13.5" r="7.5" fill="#F26B5B" /><path d="M12 6V3.5" stroke="#2E6B3A" /><path d="M9 6.5l3 1.5 3-1.5" stroke="#2E6B3A" /></> },
  cucumber: { fill: 'none', stroke: '#2E6B3A', children: <><path d="M6 18c-2-2-1.5-5 1.5-8l3-3c3-3 6-3.5 8-1.5s1.5 5-1.5 8l-3 3c-3 3-6 3.5-8 1.5z" fill="#7FB685" /><path d="M10 11h.01M13 9h.01M12 14h.01M15 12h.01" /></> },
  pepper: { fill: 'none', stroke: '#C2410C', children: <><path d="M8 8c-2 1-3 4-2 8 1 3 3 5 6 5s5-2 6-5c1-4 0-7-2-8-1.5-.7-3 .3-4 .3S9.5 7.3 8 8z" fill="#F59E5B" /><path d="M12 8.3V5c0-1 1-2 2.5-2" stroke="#2E6B3A" /></> },
  strawberry: { fill: 'none', stroke: '#BE123C', children: <><path d="M12 21c-4-2-7-6-7-10 0-2 2-3.5 7-3.5s7 1.5 7 3.5c0 4-3 8-7 10z" fill="#F0607E" /><path d="M8 7.5 12 4l4 3.5" stroke="#2E6B3A" /><path d="M10 12h.01M14 12h.01M12 15h.01" stroke="#FFFFFF" /></> },
  lettuce: { fill: 'none', stroke: '#4D7C0F', children: <><path d="M12 20c-5 0-8-3-8-7 0-3 2-5 4-5 0-2 2-4 4-4s4 2 4 4c2 0 4 2 4 5 0 4-3 7-8 7z" fill="#A6CF6B" /><path d="M12 20V10" /><path d="M12 14l-3-2M12 16l3-2" /></> },
}

export type IconName = keyof typeof paths

export function Icon(props: { name: string; size?: number; strokeWidth?: number; className?: string }) {
  const p = paths[props.name] ?? paths.sprout
  return (
    <svg
      width={props.size ?? 22}
      height={props.size ?? 22}
      viewBox="0 0 24 24"
      fill={p.fill ?? 'none'}
      stroke={p.stroke ?? 'currentColor'}
      strokeWidth={props.strokeWidth ?? 2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      className={props.className}
    >
      {p.children}
    </svg>
  )
}

/** Ekin kodi uchun ikonka (yo'q bo'lsa — nihol). */
export function cropIcon(code: string): string {
  return code in paths ? code : 'sprout'
}

/** Bosqich ikonkasi: ko'chat → nihol, o'sish → tuvakdagi o'simlik, hosil → ekin mevasi. */
export function stageIcon(stage: string, crop?: string | null): string {
  if (stage === 'seedling') return 'sprout'
  if (stage === 'vegetative') return 'substrate'
  return crop ? cropIcon(crop) : 'tomato'
}
