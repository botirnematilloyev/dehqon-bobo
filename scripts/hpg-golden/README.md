# HPG golden testlari

`internal/modules/calc/hpg` — WEGA-HPG hisob yadrosining Go nusxasi. Uning
to'g'riligi **HPG'ning o'zi** bilan tekshiriladi:

1. `generate.mjs` `hpg.html` ichidan hisob funksiyalarini (calculateS,
   calculateCa, calcEC, calcWeight, calcRatios, microToWeght, toMicrocomplex,
   calcConcentrates, calcDensity*, korrection, calculateCorrector*,
   updateCorrectionProtocol) **matnini o'zgartirmasdan** ajratib oladi.
2. Ularni soxta DOM ichida node'da ishga tushiradi: HPG repodagi barcha
   `.hpg` misol profillar + 8 ta sintetik holat (K2SO4 / Mg(NO3)2 / ikkalasi
   ham, Cl, kompleks, 250 L va 2000 L).
3. Natija `internal/modules/calc/hpg/testdata/golden.json` ga yoziladi.
   Fayl HPG commit SHA'si bilan pin qilingan.
4. `golden_test.go` Go natijasini shu fayl bilan solishtiradi: S, EC, Ca —
   bit-ma-bit; grammlar, nisbatlar, korrektor — HPG ko'rsatgan satrlar bilan
   aynan; konsentrat ml/gramm — aynan.

`go test` uchun node kerak emas: golden.json repoda.

## Qayta yaratish / HPG yangilanganda

```bash
scripts/hpg-golden/fetch.sh            # joriy pin
scripts/hpg-golden/fetch.sh <yangi-sha> # HPG'ning yangi versiyasiga o'tish
go test ./internal/modules/calc/...
```

Yangi SHA bilan test yiqilsa — HPG formulasini o'zgartirgan: farqni o'qing,
Go portini moslang yoki `HPG farqi:` sifatida hujjatlang.

## Ataylab qilingan farqlar (`HPG farqi:`)

| Joy | HPG | Bizda | Sabab |
|---|---|---|---|
| B idishi massasi (alohida mikro tuzlar) | mis kuporosi massaga qo'shilmaydi, hajmga qo'shiladi | qo'shiladi | xato ko'rinadi; test farq = Cu grammi ekanini tekshiradi |

## HPG xatti-harakati — saqlangan, lekin bilish kerak

- **Manfiy gramm**: HPG kesmaydi. Manfiy = profilni tanlangan tuzlar bilan
  qurib bo'lmaydi. `SaltGrams.Feasible()` shuni bildiradi.
- **Konsentrat ml** tarozida tortilgan (yaxlitlangan) grammdan hisoblanadi.
- **Korrektor**: web-versiyada joriy hajm doim 10 L; oraliq qiymatlar maydonga
  yozilib (toFixed) qayta o'qiladi — Go'da ham shunday.
- **`|| standart`**: HPG 0 ni "bo'sh" deb standartga almashtiradi (g/L, %,
  zichlik). Go API buni qilmaydi — chaqiruvchi aniq qiymat beradi; golden test
  HPG standartlarini o'zi qo'llaydi.

## Litsenziya

HPG — GPLv3. `hpg.html` va misollar repoga kiritilmaydi; `golden.json`
faqat sonlar. Go porti esa HPG formulalaridan olingan — ilovani tarqatishdan
oldin litsenziya masalasini yurist bilan hal qiling.
