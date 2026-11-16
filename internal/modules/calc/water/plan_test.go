package water

import (
	"math"
	"math/rand"
	"testing"

	"dehqon-bobo/internal/modules/calc/calctest"
	"dehqon-bobo/internal/modules/calc/hpg"

	"github.com/stretchr/testify/require"
)

// VadimS tomato-cherry (HPG misoli), Cl = 0.
func tomato() hpg.Macro {
	return hpg.Macro{NO3: 140.6, NH4: 11.76, P: 60.19, K: 259.01, Ca: 155.36, Mg: 49.4477}
}

func baseInput() Input {
	return Input{
		Target: tomato(), Salts: calctest.PureSalts(), Mode: hpg.SaltMode{UseK2SO4: true},
		ResidualHCO3: 0.5, IonBalanceTolerancePct: 10,
	}
}

func codes(ws []Warning) []string {
	out := []string{}
	for _, w := range ws {
		out = append(out, w.Code)
	}
	return out
}

// balancedWater — kation va anionlar teng bo'lishi uchun HCO3 tanlanadi.
func balancedWater(ca, mg, k, na, cl, so4, no3 float64) Analysis {
	a := Analysis{Ca: ca, Mg: mg, K: k, Na: na, Cl: cl, SO4: so4, NO3: no3}
	n := NutrientsFromWater(a)
	a.HCO3Mmol = cationsMeq(n, na) - anionsMeq(n, 0)
	if a.HCO3Mmol < 0 {
		// Anionlar ko'p: farqni Na bilan yopamiz (NaCl / Na2SO4 suvi).
		a.Na += -a.HCO3Mmol * massNa
		a.HCO3Mmol = 0
	}
	return a
}

func achievedMeq(a Achieved) (cations, anions float64) {
	return cationsMeq(a.Nutrients, a.Na), anionsMeq(a.Nutrients, a.HCO3Mmol)
}

func TestPureWater_IsExactlyHPG(t *testing.T) {
	in := baseInput()
	p, err := Calculate(in)
	require.NoError(t, err)

	want := hpg.DosePer10L(hpg.Balanced(in.Target), in.Salts, in.Mode)
	require.Equal(t, want, p.Dose, "RO water and no acid must reproduce HPG exactly")
	require.Empty(t, p.Acids)
	require.Empty(t, p.Warnings)
	require.Equal(t, CalcVersion, p.Version)

	d := p.Achieved.Deviation
	for name, v := range map[string]float64{"P": d.P, "K": d.K, "Ca": d.Ca, "Mg": d.Mg, "S": d.S, "NH4": d.NH4N} {
		require.InDelta(t, 0, v, 1e-9, name)
	}
	require.InDelta(t, 0, d.NO3N, 0.5, "NO3 follows from salt stoichiometry")
	require.InDelta(t, hpg.EC(p.Target), p.EC, 0.01)
}

func TestWaterNutrients_AreSubtracted(t *testing.T) {
	in := baseInput()
	in.Water = balancedWater(80, 30, 4, 0, 0, 96.061, 0) // 1 mmol SO4
	p, err := Calculate(in)
	require.NoError(t, err)

	c := in.Salts
	require.InDelta(t, (p.Target.Ca-80)/c.CaNO3Ca, p.Dose.CaNO3, 1e-12, "calcium nitrate covers only what water lacks")
	require.InDelta(t, (p.Target.Mg-30)/c.MgSO4Mg, p.Dose.MgSO4, 1e-12)

	a := p.Achieved
	require.InDelta(t, p.Target.Ca, a.Ca, 1e-9)
	require.InDelta(t, p.Target.Mg, a.Mg, 1e-9)
	require.InDelta(t, p.Target.K, a.K, 1e-9)
	require.InDelta(t, p.Target.S, a.S, 1e-9, "sulfate from water counted")

	require.Equal(t, []string{WarnBicarbonateLeft}, codes(p.Warnings), "no acid given for this water")
}

func TestUnitConversions(t *testing.T) {
	n := NutrientsFromWater(Analysis{NO3: 62.0037, SO4: 96.061})
	require.InDelta(t, 14.0067, n.NO3N, 1e-9, "1 mmol nitrate = 14 mg N")
	require.InDelta(t, 32.065, n.S, 1e-9, "1 mmol sulfate = 32 mg S")
}

func TestAcid_PhosphoricFirstThenNitric(t *testing.T) {
	in := baseInput()
	in.Target.P = 40
	in.Water = balancedWater(2*massCa, 0, 0, 0, 0, 0, 0) // Ca(HCO3)2: 4 mmol/L HCO3
	in.Acids = []Acid{Phosphoric85, Nitric38}

	p, err := Calculate(in)
	require.NoError(t, err)
	require.Len(t, p.Acids, 2)

	h3po4, hno3 := p.Acids[0], p.Acids[1]
	pMmol := 40 / massP
	require.InDelta(t, pMmol, h3po4.Mmol, 1e-12, "phosphoric acid capped at the P target")
	require.InDelta(t, 40, h3po4.Nutrient, 1e-9)
	require.False(t, h3po4.OverTarget)

	require.InDelta(t, 3.5-pMmol, hno3.Mmol, 1e-12, "nitric covers the rest down to 0.5 mmol/L")
	require.InDelta(t, (3.5-pMmol)*massN, hno3.Nutrient, 1e-9)

	// ml: mmol/L × 1000 L ÷ (ρ·1000·w / M)
	perML := 1.23 * 1000 * 0.38 / (massH + massN + 3*massO)
	require.InDelta(t, hno3.Mmol*1000/perML, hno3.MLPer1000L, 1e-9)
	require.InDelta(t, 7.418, Nitric38.MmolPerML(), 0.001)

	require.Zero(t, p.Dose.KH2PO4, "acid supplies all phosphorus")
	require.InDelta(t, 40, p.Achieved.P, 1e-9)
	require.InDelta(t, 0.5, p.Achieved.HCO3Mmol, 1e-12)
	require.Empty(t, p.Warnings)

	require.InDelta(t, p.Target.K, p.Achieved.K, 1e-9, "potassium is rebalanced onto other salts")
}

func TestAcid_SulfuricIsDiprotic(t *testing.T) {
	in := baseInput()
	in.Water = Analysis{HCO3Mmol: 2.5}
	in.Acids = []Acid{Sulfuric96}

	p, err := Calculate(in)
	require.NoError(t, err)
	require.Len(t, p.Acids, 1)
	require.InDelta(t, 1.0, p.Acids[0].Mmol, 1e-12, "2 mmol HCO3 needs 1 mmol H2SO4")
	require.InDelta(t, 2.0, p.Acids[0].Neutralized, 1e-12)
	require.InDelta(t, massS, p.Acids[0].Nutrient, 1e-9)
	require.InDelta(t, p.Target.S, p.Achieved.S, 1e-9, "acid sulfur replaces potassium sulfate")
}

func TestAcid_LastAcidOvershootsWithWarning(t *testing.T) {
	in := baseInput()
	in.Target.NO3 = 40 // ~2.86 mmol N
	in.Target.K = 120
	in.Target.Ca = 80
	in.Water = Analysis{HCO3Mmol: 8}
	in.Acids = []Acid{Nitric38}

	p, err := Calculate(in)
	require.NoError(t, err)
	require.InDelta(t, 7.5, p.Acids[0].Mmol, 1e-12, "pH wins over the nitrogen target")
	require.True(t, p.Acids[0].OverTarget)
	require.Contains(t, codes(p.Warnings), WarnAcidOverTarget)
	require.Greater(t, p.Achieved.NO3N, p.Target.NO3)
}

func TestAcid_NoBicarbonateNoAcid(t *testing.T) {
	in := baseInput()
	in.Water = Analysis{HCO3Mmol: 0.4}
	in.Acids = []Acid{Nitric38}
	p, err := Calculate(in)
	require.NoError(t, err)
	require.Empty(t, p.Acids, "already below the 0.5 mmol/L buffer")
	require.InDelta(t, 0.4, p.Achieved.HCO3Mmol, 1e-12)
}

func TestWaterExceedingTarget(t *testing.T) {
	in := baseInput()
	in.Water = balancedWater(300, 0, 0, 0, 0, 0, 0)
	in.Acids = []Acid{Nitric38}

	p, err := Calculate(in)
	require.NoError(t, err)
	require.Contains(t, codes(p.Warnings), WarnWaterExceedsTarget)
	require.Zero(t, p.Dose.CaNO3, "no calcium salt when water already has too much")
	require.InDelta(t, 300, p.Achieved.Ca, 1e-9)
	require.Greater(t, p.Achieved.Deviation.Ca, 0.0)
}

func TestElectroneutrality_RandomWaters(t *testing.T) {
	// Har qanday muvozanatli suv + kislota + tuzlar → tayyor eritma ham
	// elektroneytral bo'lishi kerak. Bu butun qatlam kimyoviy jihatdan
	// izchil ekanining eng kuchli tekshiruvi.
	rng := rand.New(rand.NewSource(42))
	acidSets := [][]Acid{{Phosphoric85, Nitric38}, {Nitric60}, {Sulfuric96}, {Phosphoric75, Sulfuric37}}
	checked := 0

	for i := 0; i < 200; i++ {
		in := baseInput()
		in.Water = balancedWater(
			rng.Float64()*100, rng.Float64()*40, rng.Float64()*8, rng.Float64()*90,
			rng.Float64()*70, rng.Float64()*150, rng.Float64()*30,
		)
		in.Acids = acidSets[i%len(acidSets)]

		p, err := Calculate(in)
		require.NoError(t, err)
		if !p.Dose.Feasible() {
			continue
		}
		checked++

		cat, an := achievedMeq(p.Achieved)
		require.InDelta(t, cat, an, 0.03, "case %d: cations %.4f vs anions %.4f meq/L", i, cat, an)

		// Ozuqa kationlari (Na emas) har doim maqsadga yetadi yoki suv
		// tufayli oshadi — hech qachon kam bo'lmaydi.
		require.GreaterOrEqual(t, p.Achieved.Deviation.Ca, -1e-9)
		require.GreaterOrEqual(t, p.Achieved.Deviation.Mg, -1e-9)
		require.GreaterOrEqual(t, p.Achieved.Deviation.K, -1e-9)
	}
	require.GreaterOrEqual(t, checked, 150, "most random waters must be solvable, otherwise the test checks nothing")
	t.Logf("electroneutrality checked on %d/200 random waters", checked)
}

func TestSodium_RaisesECAndLowersNitrate(t *testing.T) {
	in := baseInput()
	clean, err := Calculate(in)
	require.NoError(t, err)

	in.Water = balancedWater(0, 0, 0, 2*massNa, 2*massCl, 0, 0) // 2 mmol NaCl
	salty, err := Calculate(in)
	require.NoError(t, err)

	require.InDelta(t, clean.EC+2*0.095, salty.EC, 0.01, "Na adds conductivity; Cl is balanced by it")
	require.InDelta(t, 2*massCl, salty.Achieved.Cl, 1e-9)
}

func TestIonBalanceWarning(t *testing.T) {
	in := baseInput()
	in.Water = Analysis{Ca: 120, HCO3Mmol: 0.5} // ~6 meq cations vs 0.5 anions
	p, err := Calculate(in)
	require.NoError(t, err)
	require.Contains(t, codes(p.Warnings), WarnPoorIonBalance)

	require.InDelta(t, 0, IonBalanceError(balancedWater(80, 30, 4, 46, 35, 96, 6)), 1e-9)
}

func TestValidation(t *testing.T) {
	in := baseInput()
	in.Water = Analysis{Ca: -1}
	_, err := Calculate(in)
	require.ErrorIs(t, err, ErrNegativeValue)

	in = baseInput()
	in.Acids = []Acid{{Kind: Nitric, Percent: 150, Density: 1.2}}
	_, err = Calculate(in)
	require.ErrorIs(t, err, ErrInvalidAcid)

	in.Acids = []Acid{{Kind: "HCl", Percent: 30, Density: 1.15}}
	_, err = Calculate(in)
	require.ErrorIs(t, err, ErrInvalidAcid)

	in = baseInput()
	in.ResidualHCO3 = 5
	_, err = Calculate(in)
	require.ErrorIs(t, err, ErrInvalidInput, "residual must be a sane buffer")

	in = baseInput()
	in.IonBalanceTolerancePct = 0
	_, err = Calculate(in)
	require.ErrorIs(t, err, ErrInvalidInput, "parameters are required, there is no hidden default")
}

func TestParameters_ComeFromInput(t *testing.T) {
	// Agronom yoki dehqon bufer qiymatini o'zgartirsa, natija ham o'zgaradi.
	in := baseInput()
	in.Water = balancedWater(2*massCa, 0, 0, 0, 0, 0, 0) // 4 mmol/L HCO3
	in.Acids = []Acid{Nitric38}

	in.ResidualHCO3 = 0.5
	a, err := Calculate(in)
	require.NoError(t, err)
	in.ResidualHCO3 = 1.0
	b, err := Calculate(in)
	require.NoError(t, err)
	require.InDelta(t, 3.5, a.Acids[0].Mmol, 1e-12)
	require.InDelta(t, 3.0, b.Acids[0].Mmol, 1e-12)

	in.ResidualHCO3 = 0
	full, err := Calculate(in)
	require.NoError(t, err)
	require.InDelta(t, 4.0, full.Acids[0].Mmol, 1e-12, "0 = full neutralization")

	// Tahlil sifati chegarasi ham parametr.
	in = baseInput()
	in.Water = Analysis{Ca: 120, HCO3Mmol: 4.5} // ~25% farq
	in.IonBalanceTolerancePct = 10
	p, _ := Calculate(in)
	require.Contains(t, codes(p.Warnings), WarnPoorIonBalance)
	in.IonBalanceTolerancePct = 40
	p, _ = Calculate(in)
	require.NotContains(t, codes(p.Warnings), WarnPoorIonBalance)
}

func TestAcidMolarMasses(t *testing.T) {
	require.InDelta(t, 63.01, Nitric.molarMass(), 0.01)
	require.InDelta(t, 97.99, Phosphoric.molarMass(), 0.01)
	require.InDelta(t, 98.08, Sulfuric.molarMass(), 0.01)
	require.False(t, math.IsNaN(Phosphoric85.MmolPerML()))
	require.InDelta(t, 14.66, Phosphoric85.MmolPerML(), 0.01)
	require.InDelta(t, 18.01, Sulfuric96.MmolPerML(), 0.01)
}
