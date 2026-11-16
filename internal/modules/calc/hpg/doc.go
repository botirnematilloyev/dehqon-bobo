// Package hpg — WEGA-HPG (Hydroponic Profile Generator) hisob yadrosining
// Go'dagi aniq nusxasi.
//
// Manba: https://github.com/WEGA-project/wega-hpg (hpg.html, web-versiya).
// Pin qilingan commit va har bir funksiyaning asl nomi testdata/golden.json
// va har bir Go funksiyasining izohida ko'rsatilgan.
//
// Qoidalar:
//
//   - Formulalar HPG'dagi ifodalar bilan AYNAN bir xil tartibda yozilgan:
//     float64 amallari tartibi saqlansa, natija JS bilan bit-ma-bit mos
//     keladi. Shuning uchun molyar massalar `const` emas, `var` — Go
//     konstanta ifodalarini cheksiz aniqlikda hisoblaydi va natija JS'dan
//     farq qilib qolardi.
//   - Bu paketga HPG'da yo'q narsa QO'SHILMAYDI (suvni hisobga olish,
//     kislota, bosqich koeffitsientlari va h.k.). Ular alohida qatlamda,
//     o'z manbasi va testi bilan yashaydi.
//   - HPG'dan ataylab farq qiladigan joylar `HPG farqi:` deb belgilangan
//     va golden testda alohida tekshiriladi.
//
// Birliklar: makro — mg/L, mikro — µg/L (mkg/L), hajm — L, tuz tarkibi — %,
// konsentrat — g/L, zichlik — g/mL, EC — mS/sm.
//
// Litsenziya: HPG GPLv3 ostida. Bu paket uning formulalaridan olingan.
// Ilovani tarqatishdan oldin litsenziya masalasini hal qiling.
package hpg
