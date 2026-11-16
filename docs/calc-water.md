# Suv va kislota qatlami (`internal/modules/calc/water`)

HPG yadrosi (`calc/hpg`) suvni toza deb hisoblaydi. Bu qatlam uning ustida
ishlaydi va HPG'ning o'ziga **tegmaydi**: suv toza va kislota yo'q bo'lsa,
natija HPG bilan bit-ma-bit bir xil (`TestPureWater_IsExactlyHPG`).

Versiya: `hpg@0a8eb86+water/1` → `solution_batches.calc_version`.

## Algoritm

```
maqsad ──HPG balans──▶ muvozanatlangan maqsad (S)
          − suv (Ca, Mg, K, NO3→N, SO4→S, Cl)        [ortiqchasi → ogohlantirish]
          − kislota ozuqasi (P / N / S)
          = tuzlar qismi ──hpg.DosePer10L──▶ grammlar
suv + kislota + tuzlar ──▶ erishilgan tarkib, maqsaddan farq, EC bahosi
```

### Kislota

- **Neytrallanadigan miqdor:** HCO₃ − 0.5 mmol/L. 0.5 mmol/L bufer
  sifatida qoldiriladi (Sonneveld & Voogt, 2009).
- **Ishlatish tartibi:** kislotalar foydalanuvchi bergan tartibda ishlatiladi.
  Tavsiya: avval **H₃PO₄**, chunki P baribir kerak; keyin **HNO₃**.
- **Chegaralar:** har bir kislota o'z ozuqasi maqsadi bilan chegaralanadi
  (P, NO₃-N yoki S). Ro'yxatdagi **oxirgi** kislota esa qolgan bikarbonatni
  chegaradan qat'i nazar yopadi, chunki pH ozuqa maqsadidan muhimroq. Bunda
  `acid_over_target` ogohlantirishi chiqadi.
- **Protonlar (pH 5–6 da):** H₃PO₄ → 1 H⁺ (pKa₁ 2.15, pKa₂ 7.2);
  HNO₃ → 1 H⁺; H₂SO₄ → 2 H⁺.
- **Hajm:** `ml / 1000 L = mmol/L × 1000 / (ρ × 1000 × w / M)`. Konsentratsiya
  (%) va zichlik **bazadan** olinadi: agronom standarti (`fertilizers`, kind = 3),
  dehqon esa o'z idishi yorlig'idagi qiymatni kiritadi (`farm_fertilizers`).
  Kodda kislota qiymati yo'q — `docs/calc-parameters.md`'ga qarang.

### Birliklar

Laboratoriya hisobotidagidek: ionlar **mg/L ion** ko'rinishida (NO₃⁻, SO₄²⁻),
HCO₃ — **mmol/L**. Ichkarida HPG birliklariga o'tkaziladi (NO₃-N, S).

## Ogohlantirishlar

| Kod | Ma'nosi |
|---|---|
| `water_exceeds_target` | Suvda element maqsaddan ko'p. Unga mos tuz berilmaydi, eritmada bu element maqsaddan yuqori bo'ladi. |
| `acid_over_target` | Bikarbonatni yopish uchun kislota ozuqa maqsadidan oshib ketdi. |
| `bicarbonate_left` | Kislota yo'q yoki yetmadi: pH ko'tariladi. |
| `infeasible_salts` | Manfiy gramm (HPG ma'nosida): tuz rejimini almashtirish kerak. |
| `poor_ion_balance` | Suv tahlilida kation va anionlar farqi 10% dan katta, ya'ni tahlil to'liq emas yoki xato. |

## Tekshiruvlar (`plan_test.go`)

- **HPG bilan moslik:** suv toza va kislota yo'q bo'lsa, natija HPG bilan aynan bir xil.
- **Qo'lda hisoblangan holatlar:**
  - suvdagi Ca va Mg'ni ayirish
  - avval H₃PO₄, keyin HNO₃
  - H₂SO₄ ikki protonli
  - oxirgi kislota maqsaddan oshishi
  - suvdagi element maqsaddan ko'p bo'lishi
  - Na'ning EC'ga ta'siri
- **Elektroneytrallik:** 200 ta tasodifiy muvozanatli suv olinadi (har xil
  kislota to'plamlari bilan). Har bir yechilgan holatda tayyor eritmadagi
  kation va anion yig'indilari ±0.03 meq/L ichida teng chiqishi kerak.
  Hozir 162/200 holat yechiladi.
- **Mutatsiya testi:** H₂SO₄ bir protonli deb olinganda, Mg ayirilmaganda yoki
  SO₄ to'g'ridan-to'g'ri S deb olinganda — testlar yiqiladi.

## Cheklovlar (agronom bilan aniqlashtirish kerak)

- **EC bahosi:** HPG formulasiga Na ham qo'shilgan. Bu o'lchov emas, taxmin.
- **Suvdagi NH₄, Fe va mikroelementlar** hisobga olinmaydi.
- **0.5 mmol/L bufer** — `calc_parameters.residual_hco3_mmol`. Agronom ham,
  xo'jalik ham o'zgartira oladi. `water.Input.ResidualHCO3` majburiy maydon,
  kodda yashirin standart qiymat yo'q.
- **Na va Cl** to'planishi (yopiq tizimlarda) hisobga olinmaydi. Bu korrektor
  va drenaj tahlili bilan keyingi bosqich.
