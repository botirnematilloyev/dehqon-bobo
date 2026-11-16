package hpg

// Density — monoeritma zichligi (g/mL) uning konsentratsiyasidan (g/L).
// Empirik kvadrat formulalar; ular sof tuzning g/L iga moslangan, shuning
// uchun avval o'g'itdagi asosiy element ulushiga qarab qayta hisoblanadi.
// HPG: calcDensityCaNO3 ... calcDensityCaCl2. Mikro tuzlar uchun 1.0.
func Density(s Salt, gl float64, c Salts) float64 {
	switch s {
	case CaNO3:
		kmol := gl / (24.4247 / c.CaNO3Ca)
		return 0.999 + float64(0.000732*kmol) - float64(0.000000113*kmol*kmol)
	case KNO3:
		kmol := gl / (38.6717 / c.KNO3K)
		return 0.998 + float64(0.00062*kmol) - float64(0.000000114*kmol*kmol)
	case NH4NO3:
		kmol := gl / ((34.9978 / 2) / c.NH4NO3NO3)
		return 0.999 + float64(0.000397*kmol) - float64(0.0000000422*kmol*kmol)
	case MgNO3:
		kmol := gl / (16.3874 / c.MgNO3Mg)
		return 0.998 + float64(0.000736*kmol) - float64(0.000000121*kmol*kmol)
	case MgSO4:
		kmol := gl / (20.1923 / c.MgSO4Mg)
		return 0.999 + float64(0.00097*kmol) - float64(0.000000268*kmol*kmol)
	case K2SO4:
		kmol := gl / (44.8737 / c.K2SO4K)
		return 0.998 + float64(0.000814*kmol) - float64(0.00000039*kmol*kmol)
	case KH2PO4:
		kmol := gl / (28.7307 / c.KH2PO4K)
		return 0.998 + float64(0.000716*kmol) - float64(0.000000399*kmol*kmol)
	case CaCl2:
		kmol := gl / (36.1115 / c.CaCl2Ca)
		return 0.999 + float64(0.000794*kmol) - float64(0.000000151*kmol*kmol)
	}
	return 1.0
}
