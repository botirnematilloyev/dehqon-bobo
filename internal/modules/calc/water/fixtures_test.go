package water

// Test kislotalari. Ishlab chiqarishda bu qiymatlar bazadan keladi
// (migratsiya 000011 seed'i: CRC Handbook, 20 °C, taxminiy).
var (
	Nitric38     = Acid{Kind: Nitric, Percent: 38, Density: 1.23}
	Nitric60     = Acid{Kind: Nitric, Percent: 60, Density: 1.37}
	Phosphoric75 = Acid{Kind: Phosphoric, Percent: 75, Density: 1.57}
	Phosphoric85 = Acid{Kind: Phosphoric, Percent: 85, Density: 1.69}
	Sulfuric96   = Acid{Kind: Sulfuric, Percent: 96, Density: 1.84}
	Sulfuric37   = Acid{Kind: Sulfuric, Percent: 37, Density: 1.28}
)
