package hpg

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// Golden test HPG bilan mosligini tekshiradi; bu yerda — JS raqam
// semantikasi, chegaraviy holatlar va formulalarning fizik xossalari.

func TestJSNumberSemantics(t *testing.T) {
	t.Run("Math.round rounds half towards +infinity", func(t *testing.T) {
		require.Equal(t, 3.0, jsRound(2.5))
		require.Equal(t, -2.0, jsRound(-2.5), "Go math.Round would give -3")
		require.Equal(t, -3.0, jsRound(-2.6))
	})

	t.Run("toFixed uses the exact double and breaks ties upwards", func(t *testing.T) {
		require.Equal(t, "0.13", ToFixed(0.125, 2), "0.125 is exact; JS picks the larger")
		require.Equal(t, "1.00", ToFixed(1.005, 2), "1.005 is really 1.00499999...")
		require.Equal(t, "-1.50", ToFixed(-1.5, 2))
		require.Equal(t, "-0.00", ToFixed(-0.0001, 2), "JS keeps the sign")
		require.Equal(t, "3", ToFixed(2.5, 0))
		require.Equal(t, "0.00050", ToFixed(0.0005, 5))
		require.Equal(t, "NaN", ToFixed(math.NaN(), 2))
	})

	t.Run("Display = roundTo then toFixed", func(t *testing.T) {
		require.Equal(t, "2.310", Display(2.3095, 3))
		require.Equal(t, "7.51", Display(7.505000001, 2))
	})

	t.Run("|| default treats zero and NaN alike", func(t *testing.T) {
		require.Equal(t, 600.0, orDefault(0, 600))
		require.Equal(t, 600.0, orDefault(math.NaN(), 600))
		require.Equal(t, 250.0, orDefault(250, 600))
	})
}

func TestWeightDecimals(t *testing.T) {
	cases := map[float64]int{0.5: 3, 1: 3, 9.99: 3, 10: 2, 99: 2, 100: 1, 999: 1, 1000: 0, 20000: 0}
	for v, want := range cases {
		require.Equal(t, want, WeightDecimals(v), "volume %v", v)
	}
}

// Profil: VadimS tomato-cherry, Cl olib tashlangan.
func tomato() Macro {
	return Macro{NO3: 140.6, NH4: 11.76, P: 60.19, K: 259.01, Ca: 155.36, Mg: 49.4477, Cl: 0}
}

func TestBalance_Properties(t *testing.T) {
	m := Balanced(tomato())
	require.Greater(t, m.S, 0.0)

	t.Run("Ca from balance returns the Ca the balance was built on", func(t *testing.T) {
		require.InDelta(t, m.Ca, CalciumFromBalance(m), 1e-9)
	})

	t.Run("EC depends only on cations", func(t *testing.T) {
		more := m
		more.NO3 += 50
		more.P += 20
		require.Equal(t, EC(m), EC(more))
	})

	t.Run("more cations means higher EC and more sulfate to balance", func(t *testing.T) {
		richer := m
		richer.K += 39.0983 // +1 meq/L
		require.InDelta(t, EC(m)+0.095, EC(richer), 1e-9, "EC = 0.095 × (meq cations + 2)")
		require.Greater(t, SulfurFromBalance(richer), m.S)
	})

	t.Run("pure water still has the +2 meq constant", func(t *testing.T) {
		require.InDelta(t, 0.19, EC(Macro{}), 1e-12)
	})
}

func TestDose_SaltBalanceRecoversProfile(t *testing.T) {
	// Tuzlar grammidan profilni qayta yig'ib, kirish profili chiqishini
	// tekshiramiz (massa balansi). K2SO4 rejimi — hamma tuz ishtirok etadi.
	m := Balanced(tomato())
	c := hpgDefaultSalts()
	d := DosePer10L(m, c, SaltMode{UseK2SO4: true})
	require.True(t, d.Feasible())

	// g/10L × % / 100 × 1000 mg / 10 L = g × % (mg/L)
	ppm := func(g, pct float64) float64 { return g * pct }
	require.InDelta(t, m.P, ppm(d.KH2PO4, c.KH2PO4P), 1e-9)
	require.InDelta(t, m.K, ppm(d.KNO3, c.KNO3K)+ppm(d.KH2PO4, c.KH2PO4K)+ppm(d.K2SO4, c.K2SO4K), 1e-9)
	require.InDelta(t, m.Ca, ppm(d.CaNO3, c.CaNO3Ca), 1e-9)
	require.InDelta(t, m.Mg, ppm(d.MgSO4, c.MgSO4Mg), 1e-9)
	require.InDelta(t, m.S, ppm(d.MgSO4, c.MgSO4S)+ppm(d.K2SO4, c.K2SO4S), 1e-9)
	require.InDelta(t, m.NH4, ppm(d.NH4NO3, c.NH4NO3NH4)+ppm(d.CaNO3, c.CaNO3NH4), 1e-9)
}

func TestDose_InfeasibleProfileIsNegative(t *testing.T) {
	// Mg ko'p, S kam: K2SO4 bilan qurib bo'lmaydi (HPG wiki: Mg(NO3)2 ga o'ting).
	m := tomato()
	m.Mg = 120
	m.S = 60
	d := DosePer10L(m, hpgDefaultSalts(), SaltMode{UseK2SO4: true})
	require.Less(t, d.K2SO4, 0.0)
	require.False(t, d.Feasible())

	d = DosePer10L(m, hpgDefaultSalts(), SaltMode{UseMgNO3: true})
	require.GreaterOrEqual(t, d.MgNO3, 0.0, "magnesium nitrate covers the Mg surplus")
}

func TestWeights_ScaleWithVolume(t *testing.T) {
	d := DosePer10L(Balanced(tomato()), hpgDefaultSalts(), SaltMode{UseK2SO4: true})
	g10, _ := Weights(d, 10)
	g1000, total := Weights(d, 1000)
	require.InDelta(t, g10.CaNO3*100, g1000.CaNO3, 1, "1000 L = 100 × 10 L, rounded to 1 g")
	require.Equal(t, math.Trunc(total), total, "1000 L is weighed in whole grams")
}

func TestRatios(t *testing.T) {
	r, ok := Ratios(Balanced(tomato()))
	require.True(t, ok)
	require.Len(t, r, 30)
	require.InDelta(t, 259.01/155.36, r["K_Ca"], 1e-12)
	require.InDelta(t, 1/r["K_Ca"], r["Ca_K"], 1e-12)

	noNH4 := tomato()
	noNH4.NH4 = 0
	_, ok = Ratios(Balanced(noNH4))
	require.False(t, ok, "HPG skips the matrix when NH4 is zero")
}

func TestMicro(t *testing.T) {
	m := Micro{Fe: 3000, Mn: 550, B: 500, Zn: 330, Cu: 63, Mo: 63}
	pct := Micro{Fe: 13, Mn: 32.5, B: 17.5, Zn: 22.7, Cu: 25.5, Mo: 54.3}

	g := MicroGrams(m, pct, 100)
	require.InDelta(t, 3000.0/13*100/10000, g.Fe, 1e-12)
	require.Zero(t, g.Co, "no percentage, no salt")

	back := MicroFromGrams(g, pct, 100)
	require.Equal(t, m.Fe, back.Fe, "grams → µg/L round-trips")
	require.Equal(t, m.Mo, back.Mo)

	comp, total, ok := ComplexComposition(m, g, 100)
	require.True(t, ok)
	require.InDelta(t, g.Fe+g.Mn+g.B+g.Zn+g.Cu+g.Mo, total, 1e-12)
	require.InDelta(t, 3000*100/(total*10000), comp.Fe, 1e-12)

	cg := ComplexGrams(m.B, 1.0, 10)
	fromComplex := MicroFromComplex(cg, Micro{Fe: 4, B: 1}, 10, m.B)
	require.Equal(t, m.B, fromComplex.B, "boron defines the complex and is kept")
	require.Equal(t, 2000.0, fromComplex.Fe, "Fe 4% vs B 1% → 4× boron")
}

func TestDensity(t *testing.T) {
	c := hpgDefaultSalts()
	require.InDelta(t, 0.998, Density(KNO3, 0, c), 1e-12, "zero concentration ≈ water")
	require.Greater(t, Density(CaNO3, 600, c), Density(CaNO3, 300, c))
	require.Equal(t, 1.0, Density("Fe", 50, c), "micro salts use 1.0")
}

func TestCorrector(t *testing.T) {
	ions := Ions{NO3: 140.6, NH4: 11.76, P: 60.19, K: 259.01, Ca: 155.36, Mg: 49.4477}

	t.Run("no evaporation and same target: correction equals the profile", func(t *testing.T) {
		init := Correct(CorrectorInput{Initial: ions, CurrentVolume: 10, FinalVolume: 30, Final: ions})
		r := Correct(CorrectorInput{Initial: ions, CurrentEC: init.Initial.EC, CurrentVolume: 10, FinalVolume: 30, Final: ions})
		require.Equal(t, 20.0, r.CorrectionVolume)
		require.InDelta(t, ions.K, r.Correction.K, 0.002)
		require.InDelta(t, ions.NO3, r.Correction.NO3, 0.01)
	})

	t.Run("evaporated tank needs a weaker correction", func(t *testing.T) {
		init := Correct(CorrectorInput{Initial: ions, CurrentVolume: 10, FinalVolume: 30, Final: ions})
		r := Correct(CorrectorInput{Initial: ions, CurrentEC: init.Initial.EC * 1.25, CurrentVolume: 10, FinalVolume: 30, Final: ions})
		require.Less(t, r.Correction.K, ions.K)
		require.Less(t, r.Correction.EC, r.Final.EC)

		// Massa balansi: joriy + tuzatuvchi = yakuniy.
		mix := (r.Current.K*10 + r.Correction.K*20) / 30
		require.InDelta(t, ions.K, mix, 0.002)

		p := BuildProtocol(r, 10, 30)
		require.Equal(t, 200, p.VolumeChangePct)
		require.Equal(t, 33, p.OldSharePct)
		require.Equal(t, -20, p.IonChangePct["K"], "1/1.25 = −20%")
	})
}
