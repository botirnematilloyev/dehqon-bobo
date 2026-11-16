package hpg

// RatioElements — HPG nisbatlar matritsasidagi elementlar va tartib.
var RatioElements = []string{"N", "P", "K", "Ca", "Mg", "S"}

// Ratios — `A_B` kalitli massaviy nisbatlar (masalan "K_Ca" = K / Ca).
// HPG: calcRatios. Biror element (yoki NH4/NO3) 0 yoki manfiy bo'lsa HPG
// matritsani umuman yangilamaydi — bu yerda ok=false.
func Ratios(m Macro) (map[string]float64, bool) {
	N := m.N()
	if N <= 0 || m.P <= 0 || m.K <= 0 || m.Ca <= 0 || m.Mg <= 0 || m.S <= 0 || m.NH4 <= 0 || m.NO3 <= 0 {
		return nil, false
	}
	values := map[string]float64{"N": N, "P": m.P, "K": m.K, "Ca": m.Ca, "Mg": m.Mg, "S": m.S}

	out := make(map[string]float64, 30)
	for _, a := range RatioElements {
		for _, b := range RatioElements {
			if a != b {
				out[a+"_"+b] = values[a] / values[b]
			}
		}
	}
	return out, true
}

// NH4Ratio — NH4 : NO3. pH barqarorligi uchun asosiy dastak (HPG wiki).
func NH4Ratio(m Macro) (float64, bool) {
	if m.N() <= 0 || m.P <= 0 || m.K <= 0 || m.Ca <= 0 || m.Mg <= 0 || m.S <= 0 || m.NH4 <= 0 || m.NO3 <= 0 {
		return 0, false
	}
	if m.NO3 == 0 {
		return 0, false
	}
	return m.NH4 / m.NO3, true
}
