package inputs

import (
	"testing"

	"dehqon-bobo/internal/modules/calc/calctest"
	"dehqon-bobo/internal/modules/calc/hpg"
	"dehqon-bobo/internal/modules/calc/water"

	"github.com/stretchr/testify/require"
)

func f64(v float64) *float64 { return &v }

// seedLike — migratsiya 000011 seed'idagi makro tuzlar.
func seedLike() []Fertilizer {
	return []Fertilizer{
		{Code: "CaNO3", Kind: KindMacro, Elements: map[string]float64{"Ca": 16.972, "N_NO3": 11.863}},
		{Code: "KNO3", Kind: KindMacro, Elements: map[string]float64{"K": 38.672, "N_NO3": 13.854}},
		{Code: "NH4NO3", Kind: KindMacro, Elements: map[string]float64{"N_NH4": 17.499, "N_NO3": 17.499}},
		{Code: "MgNO3", Kind: KindMacro, Elements: map[string]float64{"Mg": 9.479, "N_NO3": 10.925}},
		{Code: "CaCl2", Kind: KindMacro, Elements: map[string]float64{"Ca": 18.294, "Cl": 32.366}},
		{Code: "MgSO4", Kind: KindMacro, Elements: map[string]float64{"Mg": 9.861, "S": 13.01}},
		{Code: "KH2PO4", Kind: KindMacro, Elements: map[string]float64{"K": 28.731, "P": 22.761}},
		{Code: "K2SO4", Kind: KindMacro, Elements: map[string]float64{"K": 44.874, "S": 18.401}},
		{Code: "FeChelate", Kind: KindMicro, Elements: map[string]float64{"Fe": 13}},
		{Code: "FeEDDHA", Kind: KindMicro, Elements: map[string]float64{"Fe": 6}},
		{Code: "H3BO3", Kind: KindMicro, Elements: map[string]float64{"B": 17.5}},
		{Code: "HNO3_38", Kind: KindAcid, Formula: "HNO3", ConcentrationPct: f64(38), DensityGML: f64(1.23)},
		{Code: "H3PO4_85", Kind: KindAcid, Formula: "H3PO4", ConcentrationPct: f64(85), DensityGML: f64(1.69)},
	}
}

func TestSalts_FromCatalogEqualsHPGDefaults(t *testing.T) {
	s, err := Salts(seedLike())
	require.NoError(t, err)
	require.Equal(t, calctest.PureSalts(), s, "seeded catalog must reproduce HPG's composition exactly")
}

func TestSalts_FarmOverrideChangesTheResult(t *testing.T) {
	fs := seedLike()
	// Dehqon yorliqdan: kalsiy selitra 15.5% N (shu jumladan 1.1% NH4), 19% Ca.
	fs[0].Elements = map[string]float64{"Ca": 19, "N_NO3": 14.4, "N_NH4": 1.1}
	s, err := Salts(fs)
	require.NoError(t, err)
	require.Equal(t, 19.0, s.CaNO3Ca)
	require.Equal(t, 1.1, s.CaNO3NH4)

	m := hpg.Balanced(hpg.Macro{NO3: 140, NH4: 12, P: 50, K: 250, Ca: 160, Mg: 45})
	before := hpg.DosePer10L(m, calctest.PureSalts(), hpg.SaltMode{UseK2SO4: true})
	after := hpg.DosePer10L(m, s, hpg.SaltMode{UseK2SO4: true})
	require.Less(t, after.CaNO3, before.CaNO3, "richer label → fewer grams")
	require.Less(t, after.NH4NO3, before.NH4NO3, "NH4 in calcium nitrate replaces some ammonium nitrate")
}

func TestSalts_MissingSalt(t *testing.T) {
	_, err := Salts(seedLike()[1:])
	require.ErrorIs(t, err, ErrMissingSalt)
}

func TestMicroPercent(t *testing.T) {
	require.Equal(t, 13.0, MicroPercent(seedLike()).Fe, "first in catalog order")
	require.Equal(t, 6.0, MicroPercent(seedLike(), "FeEDDHA").Fe, "farm's chosen chelate wins")
	require.Equal(t, 17.5, MicroPercent(seedLike()).B)
	require.Zero(t, MicroPercent(seedLike()).Mo, "no molybdate in this list")
}

func TestAcids(t *testing.T) {
	acids, err := Acids(seedLike(), []string{"H3PO4_85", "HNO3_38"})
	require.NoError(t, err)
	require.Equal(t, []water.Acid{
		{Kind: water.Phosphoric, Percent: 85, Density: 1.69},
		{Kind: water.Nitric, Percent: 38, Density: 1.23},
	}, acids, "order is the farmer's order")

	_, err = Acids(seedLike(), []string{"KNO3"})
	require.ErrorIs(t, err, ErrUnknownAcid, "a salt is not an acid")

	fs := seedLike()
	fs[11].DensityGML = nil
	_, err = Acids(fs, []string{"HNO3_38"})
	require.ErrorIs(t, err, ErrAcidIncomplete)

	fs[11].DensityGML = f64(3)
	_, err = Acids(fs, []string{"HNO3_38"})
	require.ErrorIs(t, err, water.ErrInvalidAcid)
}

func seedParams() map[string]float64 {
	return map[string]float64{
		ParamResidualHCO3: 0.5, ParamIonBalanceTolerance: 10,
		ParamUseK2SO4: 1, ParamUseMgNO3: 0, ParamWeightDecimals: -1,
	}
}

func TestParamsFrom(t *testing.T) {
	p, err := ParamsFrom(seedParams())
	require.NoError(t, err)
	require.Equal(t, Params{ResidualHCO3: 0.5, IonBalanceTolerancePct: 10, Mode: hpg.SaltMode{UseK2SO4: true}, WeightDecimals: -1}, p)

	for _, k := range []string{ParamResidualHCO3, ParamIonBalanceTolerance, ParamUseK2SO4, ParamUseMgNO3, ParamWeightDecimals} {
		v := seedParams()
		delete(v, k)
		_, err := ParamsFrom(v)
		require.ErrorIs(t, err, ErrMissingParameter, "no silent default for %s", k)
	}

	v := seedParams()
	v[ParamWeightDecimals] = 1.5
	_, err = ParamsFrom(v)
	require.ErrorIs(t, err, ErrInvalidParameter)
}

func TestParams_WeightDecimals(t *testing.T) {
	dose := hpg.SaltGrams{CaNO3: 7.4567}
	auto, _ := Params{WeightDecimals: -1}.Weights(dose, 10)
	require.Equal(t, 7.46, auto.CaNO3, "HPG: 10 L → 0.01 g")

	coarse, _ := Params{WeightDecimals: 0}.Weights(dose, 10)
	require.Equal(t, 7.0, coarse.CaNO3, "farmer's scale weighs whole grams")

	fine, _ := Params{WeightDecimals: 3}.Weights(dose, 10)
	require.Equal(t, 7.457, fine.CaNO3)
}
