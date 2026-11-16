// HPG golden ma'lumot generatori.
//
// WEGA-HPG web-kalkulyatorining (hpg.html) hisob funksiyalarini MATNINI
// O'ZGARTIRMASDAN ajratib oladi, soxta DOM ichida node'da ishga tushiradi
// va har bir misol profil (.hpg) uchun natijani JSON'ga yozadi. Go'dagi
// `internal/modules/calc/hpg` paketi shu JSON bilan solishtiriladi.
//
// Ishlatish:
//   node scripts/hpg-golden/generate.mjs <hpg.html> <examples-dir> <out.json> <commit-sha>
//
// hpg.html va misollar repoga KIRITILMAYDI (GPLv3, 744 KB). Ularni
// `fetch.sh` pin qilingan commit'dan yuklaydi; repoda faqat natija sonlari.

import fs from 'node:fs';
import path from 'node:path';
import vm from 'node:vm';

const [, , htmlPath, examplesDir, outPath, sha] = process.argv;
if (!htmlPath || !examplesDir || !outPath || !sha) {
  console.error('usage: generate.mjs <hpg.html> <examples-dir> <out.json> <commit-sha>');
  process.exit(2);
}

const html = fs.readFileSync(htmlPath, 'utf8');

// ---------- manbadan funksiyalarni ajratish ----------

function extractFunction(name) {
  const re = new RegExp(`function\\s+${name}\\s*\\(`);
  const m = re.exec(html);
  if (!m) throw new Error(`function ${name} not found in hpg.html`);
  let i = html.indexOf('{', m.index + m[0].length);
  let depth = 0;
  for (let j = i; j < html.length; j++) {
    const c = html[j];
    if (c === '{') depth++;
    else if (c === '}') {
      depth--;
      if (depth === 0) return html.slice(m.index, j + 1);
    }
  }
  throw new Error(`unbalanced braces in ${name}`);
}

function extractConst(name) {
  const re = new RegExp(`const\\s+${name}\\s*=`);
  const m = re.exec(html);
  if (!m) throw new Error(`const ${name} not found`);
  // `const x = (...) => { ... };` — oxirgi `};` gacha qavslarni sanab.
  let depth = 0;
  let started = false;
  for (let j = m.index; j < html.length; j++) {
    const c = html[j];
    if (c === '{') { depth++; started = true; }
    else if (c === '}') depth--;
    if (started && depth === 0 && html[j + 1] === ';') return html.slice(m.index, j + 2);
  }
  throw new Error(`cannot extract const ${name}`);
}

function extractMolarBlock() {
  const start = html.indexOf('const molN = 14.0067;');
  const end = html.indexOf('const molCl = 35.453;', start);
  if (start < 0 || end < 0) throw new Error('molar mass block not found');
  return html.slice(start, end + 'const molCl = 35.453;'.length);
}

const FUNCTIONS = [
  'calculateS', 'calculateCa', 'calcEC', 'calcWeight', 'calcRatios',
  'microToWeght', 'toMicrocomplex', 'calcConcentrates', 'getSaltComposition',
  'calcDensityCaNO3', 'calcDensityKNO3', 'calcDensityNH4NO3', 'calcDensityMgNO3',
  'calcDensityMgSO4', 'calcDensityK2SO4', 'calcDensityKH2PO4', 'calcDensityCaCl2',
  'korrection', 'calculateCorrectorS', 'calculateCorrectorEC', 'updateCorrectionProtocol',
];

const hpgSource = [
  extractMolarBlock(),
  extractConst('roundTo'),
  extractConst('setInputValue'),
  extractConst('parseInput'),
  ...FUNCTIONS.map(extractFunction),
].join('\n\n');

// ---------- soxta DOM ----------

function makeDocument() {
  const els = new Map();
  const get = (id) => {
    // Haqiqiy DOM'dagidek `value` doim satr: HPG ba'zan son yozadi
    // (`input.value = Math.round(...)`), keyin parseFloat bilan o'qiydi.
    if (!els.has(id)) {
      let v = '';
      els.set(id, {
        id, checked: false, textContent: '', step: '',
        get value() { return v; },
        set value(x) { v = String(x); },
      });
    }
    return els.get(id);
  };
  return {
    els,
    document: { activeElement: null, getElementById: get },
  };
}

// HPG ichidagi UI yangilovchi funksiyalar — no-op, lekin kerakli
// argumentlarini yozib olamiz (konsentrat jadvali shu orqali chiqadi).
const UI_STUBS = `
  var __captured = {};
  function updateConcentrateTable(which, data) { __captured['table_' + which] = data; }
  function updatePreparationTab(data) { __captured.preparation = data; }
  function updateSaltNames() {}
  function updateMobileConcentrateInfo() {}
  function updateMicroDilutionInfo() {}
  function updateInfo() {}
  function updateProfileString() {}
  function autoSaveState() {}
  function genNH4NO3event() {}
  var currentProfile = {};
  var isUpdating = false;
  var useK2SO4 = false;
  var useMgNO3 = false;
  var saltCompositions = {};
  var agFe = 20.1, agMn = 36.4, agB = 17.5, agZn = 22.7, agCu = 25.5, agMo = 54.3, agCo = 13.0, agSi = 7.0;
  var gmSUM = 0;
`;

// ---------- .hpg fayl ----------

function parseHpg(text) {
  const kv = {};
  for (const raw of text.split('\n')) {
    const line = raw.trim();
    if (!line || line.startsWith('#') || line.startsWith('date=') || !line.includes('=')) continue;
    const idx = line.indexOf('=');
    kv[line.slice(0, idx)] = line.slice(idx + 1).trim();
  }
  return kv;
}

const num = (kv, key, fallback) => {
  const v = parseFloat(kv[key]);
  return Number.isFinite(v) ? v : fallback;
};
const bool = (kv, key) => ['true', '1', 'yes'].includes(String(kv[key] ?? '').toLowerCase());

// HPG'dagi boshlang'ich qiymatlar (saltCompositions va currentProfile).
const DEFAULT_SALTS = {
  CaNO3: { Ca: 16.972, NO3: 11.863, NH4: 0.0 },
  KNO3: { K: 38.672, NO3: 13.854 },
  NH4NO3: { NH4: 17.499, NO3: 17.499 },
  MgSO4: { Mg: 9.861, S: 13.01 },
  KH2PO4: { K: 28.731, P: 22.761 },
  K2SO4: { K: 44.874, S: 18.401 },
  MgNO3: { Mg: 9.479, NO3: 10.925 },
  CaCl2: { Ca: 18.294, Cl: 32.366 },
};
const SALT_NAMES = Object.keys(DEFAULT_SALTS);
const MICRO = ['Fe', 'Mn', 'B', 'Zn', 'Cu', 'Mo', 'Co', 'Si'];

function buildInput(kv) {
  const salts = JSON.parse(JSON.stringify(DEFAULT_SALTS));
  for (const salt of SALT_NAMES) {
    for (const el of Object.keys(salts[salt])) {
      salts[salt][el] = num(kv, `${salt}_${el}`, salts[salt][el]);
    }
  }
  const input = {
    macro: {
      NO3: num(kv, 'NO3', 200), NH4: num(kv, 'NH4', 20), P: num(kv, 'P', 40), K: num(kv, 'K', 180),
      Ca: num(kv, 'Ca', 200), Mg: num(kv, 'Mg', 50), S: num(kv, 'S', 73.049), Cl: num(kv, 'Cl', 0),
    },
    micro: Object.fromEntries(MICRO.map((m) => [m, num(kv, m, 0)])),
    microPercent: Object.fromEntries(MICRO.map((m) => [m, num(kv, 'd' + m, 0)])),
    salts,
    useK2SO4: bool(kv, 'chK2SO4'),
    useMgNO3: bool(kv, 'chMgNO3'),
    complex: bool(kv, 'chkComplex'),
    volume: num(kv, 'V', 10),
    gl: {}, gml: {},
    tankA: num(kv, 'tAml', 500),
    tankB: num(kv, 'tBml', 500),
  };
  for (const s of [...SALT_NAMES, ...MICRO, 'Cmplx']) {
    if (kv['gl' + s] !== undefined) input.gl[s] = num(kv, 'gl' + s, 0);
    if (kv['gml' + s] !== undefined) input.gml[s] = num(kv, 'gml' + s, 0);
  }
  return input;
}

// ---------- bitta profilni HPG'da ishga tushirish ----------

function runProfile(input) {
  const { els, document } = makeDocument();
  const ctx = vm.createContext({ document, console: { log() {}, warn() {}, error: console.error }, Math, Number, String, JSON, parseFloat, parseInt, isNaN, Object });
  vm.runInContext(UI_STUBS + '\n' + hpgSource, ctx);

  ctx.saltCompositions = JSON.parse(JSON.stringify(input.salts));
  vm.runInContext('saltCompositions = this.saltCompositions;', ctx);
  vm.runInContext(`useK2SO4 = ${input.useK2SO4}; useMgNO3 = ${input.useMgNO3};`, ctx);

  const set = (id, v) => { document.getElementById(id).value = String(v); };
  set('volume', input.volume);
  for (const m of MICRO) {
    set(m, input.micro[m]);
    set(`micro_${m}_percent`, input.microPercent[m]);
  }
  document.getElementById('use_complex').checked = input.complex;
  for (const s of SALT_NAMES) {
    for (const el of Object.keys(input.salts[s])) set(`salt_${s}_${el}`, input.salts[s][el]);
  }
  for (const [k, v] of Object.entries(input.gl)) set('conc_gl' + k, v);
  for (const [k, v] of Object.entries(input.gml)) set('conc_gml' + k, v);
  set('tank_A', input.tankA);
  set('tank_B', input.tankB);

  // calcAll() tartibi: S balansdan, EC, tuzlar, nisbatlar.
  const data = { ...input.macro, volume: input.volume };
  data.N = data.NO3 + data.NH4;
  ctx.__data = data;
  vm.runInContext('calculateS(__data); calcEC(__data); calcWeight(__data); calcRatios(__data);', ctx);

  // Ca balansi — alohida (yashil maydon P/Cl o'zgarganda HPG shuni chaqiradi).
  ctx.__caData = { ...data };
  vm.runInContext('calculateCa(__caData);', ctx);

  const val = (id) => els.get(id)?.value ?? null;
  const out = {
    S: data.S,
    EC: data.EC,
    CaFromBalance: ctx.__caData.Ca,
    grams: Object.fromEntries(SALT_NAMES.map((s) => [s, val('g_' + s)])),
    totalSalts: val('total-salts'),
    ratios: {},
    nh4Ratio: val('NH4_ratio'),
  };
  for (const a of ['N', 'P', 'K', 'Ca', 'Mg', 'S']) {
    for (const b of ['N', 'P', 'K', 'Ca', 'Mg', 'S']) {
      if (a !== b && els.has(`m_${a}_${b}`)) out.ratios[`${a}_${b}`] = val(`m_${a}_${b}`);
    }
  }

  // Mikro + konsentratlar.
  vm.runInContext('microToWeght(); calcConcentrates();', ctx);
  out.microGrams = Object.fromEntries(MICRO.map((m) => [m, val('g_' + m)]));
  out.microAfter = Object.fromEntries(MICRO.map((m) => [m, val(m)]));
  out.totalMicroGrams = val('total_micro_grams');
  out.complexComposition = vm.runInContext('({Fe: agFe, Mn: agMn, B: agB, Zn: agZn, Cu: agCu, Mo: agMo, Co: agCo, Si: agSi})', ctx);
  const cap = vm.runInContext('__captured', ctx);
  out.concentrateA = cap.table_A;
  out.concentrateB = cap.table_B;
  out.preparation = cap.preparation;
  out.sumA = els.get('sumA')?.textContent ?? null;
  out.sumB = els.get('sumB')?.textContent ?? null;
  out.tankA = els.get('lVolA')?.textContent ?? null;
  out.tankB = els.get('lVolB')?.textContent ?? null;

  // Zichlik formulalari — fayldagi g/L uchun.
  out.density = {};
  for (const s of SALT_NAMES) {
    const gl = input.gl[s];
    if (gl !== undefined) out.density[s] = vm.runInContext(`calcDensity${s}(${gl})`, ctx);
  }

  // Korrektor: boshlang'ich = profil, EC 25% oshgan (bug'lanish),
  // maqsad = profil, 10 L dan 30 L gacha.
  for (const el of ['NO3', 'NH4', 'P', 'K', 'Ca', 'Mg', 'Cl']) {
    set(`corr_init_${el}`, input.macro[el]);
    set(`corr_final_${el}`, input.macro[el]);
  }
  set('corr_init_vol', 10);
  set('corr_final_vol', 30);
  vm.runInContext("calculateCorrectorEC('init');", ctx);
  const ec0 = parseFloat(val('corr_init_EC'));
  set('corr_curr_EC', (ec0 * 1.25).toFixed(3));
  vm.runInContext('korrection();', ctx);
  out.corrector = {};
  for (const col of ['init', 'curr', 'corr', 'final']) {
    out.corrector[col] = {};
    for (const el of ['NO3', 'NH4', 'P', 'K', 'Ca', 'Mg', 'Cl', 'S', 'EC', 'N']) {
      out.corrector[col][el] = val(`corr_${col}_${el}`);
    }
  }
  out.corrector.corrVolume = val('corr_corr_vol');
  out.corrector.currEC = (ec0 * 1.25).toFixed(3);
  out.corrector.protocol = val('corrector_protocol');

  return out;
}

// ---------- hammasi ----------

const files = fs.readdirSync(examplesDir).filter((f) => f.endsWith('.hpg')).sort();
const cases = [];
for (const file of files) {
  const kv = parseHpg(fs.readFileSync(path.join(examplesDir, file), 'utf8'));
  const input = buildInput(kv);
  cases.push({ name: file, input, output: runProfile(input) });
}

// Sintetik holatlar: uchala tuz rejimi, Cl va kompleks bilan — misollarda
// har bir tarmoq qamralishi kafolatlanmagan.
const base = buildInput({});
base.micro = { Fe: 2000, Mn: 550, B: 500, Zn: 330, Cu: 63, Mo: 63, Co: 0, Si: 0 };
base.microPercent = { Fe: 13, Mn: 32.5, B: 17.5, Zn: 22.7, Cu: 25.5, Mo: 54.3, Co: 21, Si: 13.2 };
base.gl = { CaNO3: 600, KNO3: 250, NH4NO3: 100, MgNO3: 500, CaCl2: 100, MgSO4: 600, KH2PO4: 150, K2SO4: 100, Fe: 10, Mn: 10, B: 10, Zn: 10, Cu: 10, Mo: 10, Co: 10, Si: 10, Cmplx: 10 };
for (const [name, patch] of [
  ['synthetic-plain', {}],
  ['synthetic-k2so4', { useK2SO4: true }],
  ['synthetic-mgno3', { useMgNO3: true, macro: { ...base.macro, S: 30 } }],
  ['synthetic-both-flags', { useK2SO4: true, useMgNO3: true }],
  ['synthetic-chloride', { useK2SO4: true, macro: { ...base.macro, Cl: 35 } }],
  ['synthetic-complex', { complex: true }],
  ['synthetic-volume-250', { useK2SO4: true, volume: 250 }],
  ['synthetic-volume-2000', { useK2SO4: true, volume: 2000 }],
]) {
  const input = { ...JSON.parse(JSON.stringify(base)), ...patch };
  cases.push({ name, input, output: runProfile(input) });
}

const doc = {
  source: 'https://github.com/WEGA-project/wega-hpg/blob/' + sha + '/hpg.html',
  commit: sha,
  generator: 'scripts/hpg-golden/generate.mjs',
  note: 'Values are produced by running HPG functions unmodified. Do not edit by hand.',
  cases,
};
fs.writeFileSync(outPath, JSON.stringify(doc, null, 1) + '\n');
console.log(`wrote ${cases.length} cases to ${outPath}`);
