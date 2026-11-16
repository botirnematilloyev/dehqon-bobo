// Package water — HPG yadrosi ustidagi qatlam: sug'orish suvining tarkibini
// hisobga olish va bikarbonatni kislota bilan neytrallash.
//
// HPG o'zi suvni "toza" deb hisoblaydi. O'zbekistondagi quduq va kanal
// suvlarida Ca, Mg, SO4 va HCO3 ko'p, shuning uchun bu qatlam shart.
//
// Usul (Sonneveld & Voogt, "Plant Nutrition of Greenhouse Crops", 2009,
// 11-bob "Nutrient solutions" va standart issiqxona amaliyoti):
//
//  1. Maqsad profil HPG kabi muvozanatlanadi (S ion balansidan).
//  2. Suvdagi ozuqa ionlari (Ca, Mg, K, NO3, SO4, Cl) maqsaddan ayiriladi.
//     Na ozuqa emas — ayirilmaydi, lekin EC bahosiga va erishilgan
//     profilga kiradi.
//  3. Bikarbonat kislota bilan neytrallanadi: HCO3⁻ + H⁺ → CO2 + H2O.
//     Bufer uchun 0.5 mmol/L qoldiriladi (pH taxminan 5.5–6 da barqaror).
//     Kislotaning anioni ozuqa: H3PO4 → H2PO4⁻ (P), HNO3 → NO3⁻ (N),
//     H2SO4 → SO4²⁻ (S); ular ham maqsaddan ayiriladi.
//     pH 5–6 oralig'ida H3PO4 bitta proton beradi (pKa1 = 2.15, pKa2 = 7.2),
//     H2SO4 — ikkita.
//  4. Qolgan "tuzlar qismi" HPG yechuvchisiga (hpg.DosePer10L) beriladi.
//  5. Suv + kislota + tuzlardan haqiqatda erishilgan profil qayta yig'iladi
//     va maqsaddan farqi ko'rsatiladi. Masalan, suvdagi Na⁻ ga mos anion
//     tufayli NO3 maqsaddan biroz past chiqadi — bu xato emas, kimyo.
//
// Bu qatlam HPG'da YO'Q. Uning natijasi CalcVersion bilan belgilanadi
// (solution_batches.calc_version) va agronom tasdig'igacha ilovada
// "suv hisobga olingan (taxminiy)" deb ko'rsatiladi.
package water
