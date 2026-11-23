import type { I18nName } from './types'

// Ilova matnlari. Server xabarlari (xatolar) API'dan Accept-Language bilan
// keladi; bu yerda faqat interfeys matnlari va kodlar → matn (ogohlantirish,
// maslahat).

export type Lang = 'uz' | 'ru' | 'en'

export function langFrom(code: string): Lang {
  const c = code.toLowerCase()
  if (c.startsWith('ru')) return 'ru'
  if (c.startsWith('en')) return 'en'
  return 'uz'
}

/** Ko'p tilli nom: tanlangan til → o'zbekcha → birinchisi. */
export function pickName(name: I18nName | undefined, lang: Lang): string {
  if (!name) return ''
  return name[lang] ?? name.uz ?? Object.values(name)[0] ?? ''
}

const uz = {
  loading: 'Yuklanmoqda…',
  motto: 'Mehnatning tagi — rohat',
  openInTelegram: 'Ilovani Telegram orqali oching',
  openInTelegramHint: 'Botda /start yuboring va «Ilovani ochish» tugmasini bosing.',
  retry: 'Qayta urinish',
  back: 'Orqaga',
  save: 'Saqlash',

  onbTitle: 'Xo‘jaligingizni sozlaymiz',
  onbHint: 'Uch savol — retsept shunga qarab hisoblanadi.',
  region: 'Viloyat',
  waterSource: 'Suv manbai',
  system: 'Yetishtirish tizimi',
  continue: 'Davom etish',
  ws_well: 'Quduq', ws_canal: 'Ariq / kanal', ws_tap: 'Vodoprovod', ws_filter: 'Filtr / osmos',
  sys_greenhouse_soil: 'Issiqxona, tuproq', sys_substrate: 'Substrat yoki tuvak', sys_hydroponics: 'Gidroponika', sys_open_field: 'Ochiq dala',

  hello: 'Assalomu alaykum',
  mixSolution: 'Eritma tayyorlash',
  mixSolutionHint: 'Ekin va bosqichni tanlang — grammlarni hisoblaymiz',
  journal: 'Jurnal',
  measure: 'O‘lchash',
  water: 'Suv',
  waterEstimate: 'taxminiy',
  lastRecipes: 'Oxirgi eritmalar',
  noRecipes: 'Hali eritma tayyorlanmagan.',
  all: 'Hammasi',

  crop: 'Ekin',
  stage: 'Bosqich',
  st_seedling: 'Ko‘chat', st_vegetative: 'O‘sish', st_fruiting: 'Hosil',
  tankVolume: 'Bak hajmi, litr',
  acids: 'Kislota (suvdagi bikarbonat uchun)',
  calculate: 'Hisoblash',
  tankA: 'A bak — kalsiy va nitratlar',
  tankB: 'B bak — sulfat, fosfat, mikro',
  oneTank: 'Bitta bakka: avval 1-guruhni to‘liq eriting, keyin 2-guruhni.',
  group1: 'Kalsiy va nitratlar',
  group2: 'Sulfat, fosfat, mikro',
  acidsTitle: 'Kislota',
  expectedEC: 'Kutilgan EC',
  phRange: 'pH oralig‘i',
  grams: 'g',
  ml: 'ml',
  saveToJournal: 'Jurnalga saqlash',
  saved: 'Jurnalga saqlandi',
  markMixed: 'Tayyorladim',
  mixedDone: 'Tayyorlangan',
  planned: 'Tayyorlanmagan',
  measureNow: 'EC va pH ni o‘lchash',
  agronomNotReviewed: 'Agronom hali tasdiqlamagan',
  agronomReviewed: 'Agronom ko‘rdi ✓',

  ecIn: 'EC — bakdagi eritma',
  phIn: 'pH — bakdagi eritma',
  ecDrain: 'EC — drenaj',
  drainPct: 'Drenaj, %',
  optional: 'ixtiyoriy',
  result: 'Natija',
  st_low: 'past', st_ok: 'me’yorida', st_high: 'yuqori',
  addWater: (l: number) => `Bakka ${l} litr suv qo‘shing.`,
  addSalts: (pct: number) => `Retsept tuzlaridan yana ${pct}% qo‘shing (kislotasiz):`,
  injector: (n: number) => `Injektor nisbatini 1:${n} ga o‘zgartiring.`,
  allGood: 'Hammasi me’yorida',
} satisfies Record<string, unknown>

export type Strings = typeof uz

const ru: Strings = {
  loading: 'Загрузка…',
  motto: 'Терпение и труд всё перетрут',
  openInTelegram: 'Откройте приложение через Telegram',
  openInTelegramHint: 'Отправьте боту /start и нажмите «Открыть приложение».',
  retry: 'Повторить',
  back: 'Назад',
  save: 'Сохранить',

  onbTitle: 'Настроим ваше хозяйство',
  onbHint: 'Три вопроса — по ним считается рецепт.',
  region: 'Область',
  waterSource: 'Источник воды',
  system: 'Система выращивания',
  continue: 'Продолжить',
  ws_well: 'Скважина', ws_canal: 'Арык / канал', ws_tap: 'Водопровод', ws_filter: 'Фильтр / осмос',
  sys_greenhouse_soil: 'Теплица, грунт', sys_substrate: 'Субстрат или горшки', sys_hydroponics: 'Гидропоника', sys_open_field: 'Открытый грунт',

  hello: 'Здравствуйте',
  mixSolution: 'Приготовить раствор',
  mixSolutionHint: 'Выберите культуру и фазу — посчитаем граммы',
  journal: 'Журнал',
  measure: 'Измерить',
  water: 'Вода',
  waterEstimate: 'примерно',
  lastRecipes: 'Последние растворы',
  noRecipes: 'Растворов пока нет.',
  all: 'Все',

  crop: 'Культура',
  stage: 'Фаза',
  st_seedling: 'Рассада', st_vegetative: 'Рост', st_fruiting: 'Плодоношение',
  tankVolume: 'Объём бака, л',
  acids: 'Кислота (для бикарбонатов воды)',
  calculate: 'Рассчитать',
  tankA: 'Бак A — кальций и нитраты',
  tankB: 'Бак B — сульфаты, фосфаты, микро',
  oneTank: 'В один бак: сначала полностью растворите группу 1, затем группу 2.',
  group1: 'Кальций и нитраты',
  group2: 'Сульфаты, фосфаты, микро',
  acidsTitle: 'Кислота',
  expectedEC: 'Ожидаемая EC',
  phRange: 'Диапазон pH',
  grams: 'г',
  ml: 'мл',
  saveToJournal: 'Сохранить в журнал',
  saved: 'Сохранено в журнал',
  markMixed: 'Приготовил',
  mixedDone: 'Приготовлен',
  planned: 'Не приготовлен',
  measureNow: 'Измерить EC и pH',
  agronomNotReviewed: 'Агроном ещё не проверил',
  agronomReviewed: 'Проверено агрономом ✓',

  ecIn: 'EC — раствор в баке',
  phIn: 'pH — раствор в баке',
  ecDrain: 'EC — дренаж',
  drainPct: 'Дренаж, %',
  optional: 'необязательно',
  result: 'Результат',
  st_low: 'низкий', st_ok: 'в норме', st_high: 'высокий',
  addWater: (l: number) => `Добавьте в бак ${l} л воды.`,
  addSalts: (pct: number) => `Добавьте ещё ${pct}% солей рецепта (без кислоты):`,
  injector: (n: number) => `Измените пропорцию инжектора на 1:${n}.`,
  allGood: 'Всё в норме',
}

const en: Strings = {
  loading: 'Loading…',
  motto: 'Hard work pays off',
  openInTelegram: 'Open the app from Telegram',
  openInTelegramHint: 'Send /start to the bot and tap “Open the app”.',
  retry: 'Retry',
  back: 'Back',
  save: 'Save',

  onbTitle: 'Let’s set up your farm',
  onbHint: 'Three questions — the recipe is calculated from them.',
  region: 'Region',
  waterSource: 'Water source',
  system: 'Growing system',
  continue: 'Continue',
  ws_well: 'Well', ws_canal: 'Canal', ws_tap: 'Tap water', ws_filter: 'Filter / RO',
  sys_greenhouse_soil: 'Greenhouse, soil', sys_substrate: 'Substrate or pots', sys_hydroponics: 'Hydroponics', sys_open_field: 'Open field',

  hello: 'Hello',
  mixSolution: 'Mix a solution',
  mixSolutionHint: 'Pick a crop and stage — we’ll calculate the grams',
  journal: 'Journal',
  measure: 'Measure',
  water: 'Water',
  waterEstimate: 'estimate',
  lastRecipes: 'Recent solutions',
  noRecipes: 'No solutions yet.',
  all: 'All',

  crop: 'Crop',
  stage: 'Stage',
  st_seedling: 'Seedling', st_vegetative: 'Vegetative', st_fruiting: 'Fruiting',
  tankVolume: 'Tank volume, L',
  acids: 'Acid (for water bicarbonate)',
  calculate: 'Calculate',
  tankA: 'Tank A — calcium and nitrates',
  tankB: 'Tank B — sulfates, phosphates, micro',
  oneTank: 'One tank: fully dissolve group 1 first, then group 2.',
  group1: 'Calcium and nitrates',
  group2: 'Sulfates, phosphates, micro',
  acidsTitle: 'Acid',
  expectedEC: 'Expected EC',
  phRange: 'pH range',
  grams: 'g',
  ml: 'ml',
  saveToJournal: 'Save to journal',
  saved: 'Saved to journal',
  markMixed: 'I mixed it',
  mixedDone: 'Mixed',
  planned: 'Not mixed yet',
  measureNow: 'Measure EC and pH',
  agronomNotReviewed: 'Not yet reviewed by an agronomist',
  agronomReviewed: 'Reviewed by an agronomist ✓',

  ecIn: 'EC — tank solution',
  phIn: 'pH — tank solution',
  ecDrain: 'EC — drain',
  drainPct: 'Drain, %',
  optional: 'optional',
  result: 'Result',
  st_low: 'low', st_ok: 'OK', st_high: 'high',
  addWater: (l: number) => `Add ${l} L of water to the tank.`,
  addSalts: (pct: number) => `Add another ${pct}% of the recipe salts (no acid):`,
  injector: (n: number) => `Change the injector ratio to 1:${n}.`,
  allGood: 'All within range',
}

export const strings: Record<Lang, Strings> = { uz, ru, en }

// Hisob ogohlantirishlari (server `warnings[].code`).
const warnings: Record<Lang, Record<string, string>> = {
  uz: {
    profile_not_reviewed: 'Bu retseptni agronom hali tasdiqlamagan.',
    water_is_estimate: 'Suv tahlili taxminiy — aniq natija uchun laboratoriya tahlilini kiriting.',
    water_salinity_above_crop_threshold: 'Suvingiz bu ekin uchun sho‘r — hosil kamayishi mumkin.',
    water_exceeds_target: 'Suvda bu element kerakligidan ko‘p.',
    poor_ion_balance: 'Suv tahlili to‘liq emas yoki xato — natija taxminiy.',
    no_acid_selected: 'Suvda bikarbonat ko‘p — kislota tanlang.',
    bicarbonate_left: 'Kislota yetmadi — pH ko‘tarilishi mumkin.',
    acid_over_target: 'Kislota ozuqa me’yoridan ko‘proq berdi.',
    infeasible_salts: 'Bu tuzlar bilan retseptni yig‘ib bo‘lmaydi — agronomga murojaat qiling.',
    micro_fertilizer_missing: 'Mikroelement uchun o‘g‘it yo‘q.',
    solubility_exceeded: 'Konsentrat juda quyuq — tuz erimay qolishi mumkin. Idishni kattaroq oling.',
  },
  ru: {
    profile_not_reviewed: 'Этот рецепт ещё не проверен агрономом.',
    water_is_estimate: 'Анализ воды примерный — для точности внесите лабораторный анализ.',
    water_salinity_above_crop_threshold: 'Вода слишком солёная для этой культуры — урожай может снизиться.',
    water_exceeds_target: 'В воде этого элемента больше, чем нужно.',
    poor_ion_balance: 'Анализ воды неполный или ошибочный — результат примерный.',
    no_acid_selected: 'В воде много бикарбонатов — выберите кислоту.',
    bicarbonate_left: 'Кислоты не хватило — pH может вырасти.',
    acid_over_target: 'Кислота дала больше питания, чем нужно.',
    infeasible_salts: 'Этими солями рецепт не собрать — обратитесь к агроному.',
    micro_fertilizer_missing: 'Нет удобрения для микроэлемента.',
    solubility_exceeded: 'Концентрат слишком густой — соль может не раствориться. Возьмите ёмкость больше.',
  },
  en: {
    profile_not_reviewed: 'An agronomist has not reviewed this recipe yet.',
    water_is_estimate: 'The water analysis is an estimate — enter a lab analysis for accuracy.',
    water_salinity_above_crop_threshold: 'Your water is too saline for this crop — yield may drop.',
    water_exceeds_target: 'The water already has more of this element than needed.',
    poor_ion_balance: 'The water analysis is incomplete or wrong — the result is approximate.',
    no_acid_selected: 'The water has a lot of bicarbonate — pick an acid.',
    bicarbonate_left: 'Not enough acid — pH may rise.',
    acid_over_target: 'The acid added more nutrient than needed.',
    infeasible_salts: 'This recipe cannot be made with these salts — ask an agronomist.',
    micro_fertilizer_missing: 'No fertilizer for a micronutrient.',
    solubility_exceeded: 'The concentrate is too strong — salt may not dissolve. Use a bigger tank.',
  },
}

const advice: Record<Lang, Record<string, string>> = {
  uz: {
    ph_high_add_acid: 'pH yuqori — biroz kislota qo‘shing.',
    ph_low_check_acid: 'pH past — keyingi safar kislotani kamaytiring.',
    drain_ec_high: 'Drenaj EC yuqori — tuz to‘planmoqda, sug‘orishni ko‘paytiring.',
    drain_pct_low: 'Drenaj kam — sug‘orishni biroz ko‘paytiring.',
    drain_pct_high: 'Drenaj ko‘p — sug‘orishni kamaytiring.',
  },
  ru: {
    ph_high_add_acid: 'pH высокий — добавьте немного кислоты.',
    ph_low_check_acid: 'pH низкий — в следующий раз уменьшите кислоту.',
    drain_ec_high: 'EC дренажа высокая — соли накапливаются, увеличьте полив.',
    drain_pct_low: 'Дренажа мало — немного увеличьте полив.',
    drain_pct_high: 'Дренажа много — уменьшите полив.',
  },
  en: {
    ph_high_add_acid: 'pH is high — add a little acid.',
    ph_low_check_acid: 'pH is low — use less acid next time.',
    drain_ec_high: 'Drain EC is high — salts are building up, irrigate more.',
    drain_pct_low: 'Too little drain — irrigate a bit more.',
    drain_pct_high: 'Too much drain — irrigate less.',
  },
}

export function warningText(lang: Lang, code: string): string {
  return warnings[lang][code] ?? warnings.uz[code] ?? code
}

export function adviceText(lang: Lang, code: string): string | null {
  return advice[lang][code] ?? null
}
