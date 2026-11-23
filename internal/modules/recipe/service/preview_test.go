package service

import (
	"context"
	"math"
	"testing"

	"dehqon-bobo/internal/modules/calc/inputs"
	"dehqon-bobo/internal/modules/calc/water"
	"dehqon-bobo/internal/modules/recipe/domain"

	"github.com/stretchr/testify/require"
)

func f64(v float64) *float64 { return &v }
func i32(v int32) *int32     { return &v }

func fert(code string, kind, tank int16, formula string, el map[string]float64, sol *float64) domain.Fertilizer {
	return domain.Fertilizer{
		Fertilizer: inputs.Fertilizer{Code: code, Kind: kind, Formula: formula, Elements: el},
		Name:       map[string]string{"uz": code}, Tank: tank, SolubilityGL: sol,
	}
}

func acid(code, formula string, pct, density float64) domain.Fertilizer {
	f := fert(code, inputs.KindAcid, 3, formula, map[string]float64{}, nil)
	f.ConcentrationPct, f.DensityGML = &pct, &density
	return f
}

// catalogFixture — migratsiya 000011 seed'i (HPG tarkiblari).
func catalogFixture() []domain.Fertilizer {
	return []domain.Fertilizer{
		fert("CaNO3", 1, 1, "Ca(NO3)2·4H2O", map[string]float64{"Ca": 16.972, "N_NO3": 11.863}, f64(1290)),
		fert("KNO3", 1, 1, "KNO3", map[string]float64{"K": 38.672, "N_NO3": 13.854}, f64(316)),
		fert("NH4NO3", 1, 1, "NH4NO3", map[string]float64{"N_NH4": 17.499, "N_NO3": 17.499}, f64(1920)),
		fert("MgNO3", 1, 1, "Mg(NO3)2·6H2O", map[string]float64{"Mg": 9.479, "N_NO3": 10.925}, nil),
		fert("CaCl2", 1, 1, "CaCl2·6H2O", map[string]float64{"Ca": 18.294, "Cl": 32.366}, nil),
		fert("MgSO4", 1, 2, "MgSO4·7H2O", map[string]float64{"Mg": 9.861, "S": 13.010}, f64(710)),
		fert("KH2PO4", 1, 2, "KH2PO4", map[string]float64{"K": 28.731, "P": 22.761}, f64(330)),
		fert("K2SO4", 1, 2, "K2SO4", map[string]float64{"K": 44.874, "S": 18.401}, f64(100)),
		fert("FeChelate", 2, 2, "Fe-DTPA", map[string]float64{"Fe": 13.0}, nil),
		fert("MnSO4", 2, 2, "MnSO4·H2O", map[string]float64{"Mn": 32.5}, nil),
		fert("H3BO3", 2, 2, "H3BO3", map[string]float64{"B": 17.5}, nil),
		fert("ZnSO4", 2, 2, "ZnSO4·7H2O", map[string]float64{"Zn": 22.7}, nil),
		fert("CuSO4", 2, 2, "CuSO4·5H2O", map[string]float64{"Cu": 25.5}, nil),
		fert("NH4Mo", 2, 2, "(NH4)6Mo7O24·4H2O", map[string]float64{"Mo": 54.3}, nil),
		acid("HNO3_38", "HNO3", 38, 1.23),
		acid("H3PO4_85", "H3PO4", 85, 1.69),
	}
}

func paramsFixture() map[string]float64 {
	return map[string]float64{
		inputs.ParamResidualHCO3: 0.5, inputs.ParamIonBalanceTolerance: 10,
		inputs.ParamUseK2SO4: 1, inputs.ParamUseMgNO3: 0, inputs.ParamWeightDecimals: -1,
	}
}

// tomato — Sonneveld & Voogt (2009) standarti, migratsiya 000012 bilan bir xil.
func tomato() domain.Profile {
	return domain.Profile{
		ID: 1, CropCode: "tomato", SalinityECw: f64(1.7), Stage: domain.StageFruiting, Reviewed: true,
		Targets: domain.Targets{
			NH4N: 16.808, NO3N: 224.107, P: 46.461, K: 371.434, Ca: 216.421, Mg: 58.332, S: 141.086,
			Fe: 0.8377, Mn: 0.5494, B: 0.3243, Zn: 0.3269, Cu: 0.0477, Mo: 0.048,
		},
		ECTarget: f64(2.6), PHMin: 5.5, PHMax: 6.5,
	}
}

func farmWith(w water.Analysis) domain.FarmData {
	return domain.FarmData{
		FarmID: 1, Method: domain.MethodDirect,
		Water:       &domain.Water{ID: 5, Analysis: w},
		Fertilizers: catalogFixture(), Params: paramsFixture(),
	}
}

func direct(volume float64, acids ...string) PreviewInput {
	return PreviewInput{CropCode: "tomato", Stage: domain.StageFruiting, VolumeL: &volume, Acids: acids}
}

func items(t Tank) map[string]float64 {
	m := map[string]float64{}
	for _, it := range t.Items {
		m[it.Code] = it.Grams
	}
	return m
}

func codes(ws []water.Warning) []string {
	out := []string{}
	for _, w := range ws {
		out = append(out, w.Code)
	}
	return out
}

func requireNear(t *testing.T, want, got water.Nutrients, tol float64) {
	t.Helper()
	pairs := [][3]any{
		{"NO3", want.NO3N, got.NO3N}, {"NH4", want.NH4N, got.NH4N}, {"P", want.P, got.P}, {"K", want.K, got.K},
		{"Ca", want.Ca, got.Ca}, {"Mg", want.Mg, got.Mg}, {"S", want.S, got.S}, {"Cl", want.Cl, got.Cl},
	}
	for _, p := range pairs {
		require.InDelta(t, p[1].(float64), p[2].(float64), tol, p[0].(string))
	}
}

func TestCompute_PureWater_ReachesTarget(t *testing.T) {
	p, err := Compute(tomato(), farmWith(water.Analysis{}), direct(1000))
	require.NoError(t, err)
	require.Empty(t, p.Warnings)
	require.Equal(t, 1000.0, p.VolumeL)

	// Tuzlar maqsadni to'liq yig'adi.
	requireNear(t, p.Target, p.Achieved, 0.01)

	// Qo'lda: Ca / %Ca × V/10 = 216.421 / 16.972 × 100 = 1275.2 → 1000 L da 1 g aniqlik.
	a, b := items(p.Tanks[0]), items(p.Tanks[1])
	require.Equal(t, 1275.0, a["CaNO3"])
	require.NotContains(t, a, "MgNO3", "use_mgno3 = 0")
	require.Contains(t, b, "K2SO4", "use_k2so4 = 1")
	// Fe: 0.8377 mg/L × 1000 / 13% × 1000 L / 10000 = 6.444 g.
	require.Equal(t, 6.444, b["FeChelate"])

	// HPG EC: 0.095 × (26.3 meq + 2) ≈ 2.69 — manbadagi 2.6 ga yaqin.
	require.InDelta(t, 2.69, p.ECEstimate, 0.02)
	require.Equal(t, water.CalcVersion, p.CalcVersion)
}

func TestCompute_HardWater_AcidNeutralizesBicarbonate(t *testing.T) {
	// Muvozanatli suv: kationlar 5.098 = anionlar 5.098 meq/L.
	w := water.Analysis{EC: 0.6, Ca: 60, Mg: 15, Na: 20, Cl: 25, SO4: 18.9, HCO3Mmol: 4}
	p, err := Compute(tomato(), farmWith(w), direct(200, "H3PO4_85", "HNO3_38"))
	require.NoError(t, err)

	require.InDelta(t, 0.5, p.HCO3LeftMmol, 1e-9, "residual_hco3_mmol")
	require.Len(t, p.Acids, 2)
	require.Equal(t, "H3PO4_85", p.Acids[0].Code)
	require.Equal(t, "P", p.Acids[0].Element)
	require.Equal(t, "HNO3_38", p.Acids[1].Code)
	require.Equal(t, "C", p.Acids[0].Tank)
	// H3PO4 P maqsadini to'liq beradi → KH2PO4 kerak emas.
	require.NotContains(t, items(p.Tanks[1]), "KH2PO4")
	require.Greater(t, p.Acids[0].ML, 0.0)

	// Suvdagi Cl (25) maqsaddan (0) ko'p.
	require.Contains(t, codes(p.Warnings), water.WarnWaterExceedsTarget)
	require.NotContains(t, codes(p.Warnings), water.WarnPoorIonBalance)

	// Kationlar va S, P aniq maqsadda; NO3 — "erkin" anion (HPG modeli):
	// elektroneytrallik bo'yicha u Na − Cl − qoldiq HCO3 ga siljiydi.
	dev := p.Deviation
	wantNO3 := (20/22.98977 - 25/35.453 - 0.5) * 14.0067
	require.InDelta(t, wantNO3, dev.NO3N, 0.05)
	require.InDelta(t, 25, dev.Cl, 1e-9)
	dev.NO3N, dev.Cl = 0, 0
	requireNear(t, water.Nutrients{}, dev, 0.01)
}

func TestCompute_NoAcid_Warns(t *testing.T) {
	p, err := Compute(tomato(), farmWith(water.Analysis{EC: 0.5, Ca: 40, HCO3Mmol: 3, SO4: 50, Mg: 5}), direct(100))
	require.NoError(t, err)
	require.Contains(t, codes(p.Warnings), WarnNoAcidSelected)
	require.Contains(t, codes(p.Warnings), water.WarnBicarbonateLeft)
	require.InDelta(t, 3, p.HCO3LeftMmol, 1e-9)
}

func TestCompute_FarmWarnings(t *testing.T) {
	prof := tomato()
	prof.Reviewed = false
	farm := farmWith(water.Analysis{EC: 2.1, Ca: 80, Mg: 30, Na: 150, Cl: 180, SO4: 200, HCO3Mmol: 2})
	farm.Water.IsEstimate = true
	farm.Fertilizers = farm.Fertilizers[1:] // CaNO3 yo'q → katalog to'liq emas
	_, err := Compute(prof, farm, direct(100))
	require.ErrorIs(t, err, domain.ErrCatalogIncomplete)

	farm.Fertilizers = catalogFixture()
	var noFe []domain.Fertilizer
	for _, f := range farm.Fertilizers {
		if f.Code != "FeChelate" {
			noFe = append(noFe, f)
		}
	}
	farm.Fertilizers = noFe
	p, err := Compute(prof, farm, direct(100, "HNO3_38"))
	require.NoError(t, err)
	got := codes(p.Warnings)
	for _, c := range []string{WarnProfileNotReviewed, WarnWaterIsEstimate, WarnWaterSalinity, WarnMicroMissing} {
		require.Contains(t, got, c)
	}
	require.NotContains(t, items(p.Tanks[1]), "FeChelate")
}

func TestCompute_ABConcentrate(t *testing.T) {
	farm := farmWith(water.Analysis{})
	farm.Method, farm.StockTankL, farm.InjectorRatio = domain.MethodAB, f64(100), i32(100)

	p, err := Compute(tomato(), farm, PreviewInput{CropCode: "tomato", Stage: domain.StageFruiting})
	require.NoError(t, err)
	require.Equal(t, 10000.0, p.VolumeL, "100 L × 1:100")
	require.Equal(t, 12752.0, items(p.Tanks[0])["CaNO3"])
	require.NotContains(t, codes(p.Warnings), WarnSolubilityExceeded)

	// 20 L idish × 1:500: K2SO4 ~174 g/L > 100 g/L eruvchanlik.
	p, err = Compute(tomato(), farm, PreviewInput{
		CropCode: "tomato", Stage: domain.StageFruiting, StockTankL: f64(20), InjectorRatio: i32(500),
	})
	require.NoError(t, err)
	var sol []water.Warning
	for _, w := range p.Warnings {
		if w.Code == WarnSolubilityExceeded {
			sol = append(sol, w)
		}
	}
	require.Len(t, sol, 1)
	require.Equal(t, "K2SO4", sol[0].Element)
	require.Greater(t, sol[0].Value, 100.0)
}

func TestCompute_Errors(t *testing.T) {
	farm := farmWith(water.Analysis{})
	ab := domain.MethodAB
	bad := domain.Method(9)

	cases := []struct {
		name string
		farm func(*domain.FarmData)
		in   PreviewInput
		want error
	}{
		{"no volume", nil, PreviewInput{}, domain.ErrInvalidMethod},
		{"huge volume", nil, direct(math.Inf(1)), domain.ErrInvalidMethod},
		{"ab without tank", nil, PreviewInput{Method: &ab}, domain.ErrInvalidMethod},
		{"unknown method", nil, PreviewInput{Method: &bad}, domain.ErrInvalidMethod},
		{"no water", func(f *domain.FarmData) { f.Water = nil }, direct(100), domain.ErrWaterRequired},
		{"unknown acid", nil, direct(100, "HCl"), domain.ErrInvalidAcid},
		{"missing param", func(f *domain.FarmData) { delete(f.Params, inputs.ParamResidualHCO3) }, direct(100), domain.ErrCatalogIncomplete},
		{"bad param", func(f *domain.FarmData) { f.Params[inputs.ParamResidualHCO3] = 9 }, direct(100), domain.ErrCatalogIncomplete},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := farm
			f.Params = paramsFixture()
			f.Water = farm.Water
			if tc.farm != nil {
				tc.farm(&f)
			}
			_, err := Compute(tomato(), f, tc.in)
			require.ErrorIs(t, err, tc.want)
		})
	}
}

// ---------- PreviewService ----------

type fakeFarms struct {
	data *domain.FarmData
	err  error
}

func (f fakeFarms) RecipeData(context.Context, int64, int64) (*domain.FarmData, error) {
	return f.data, f.err
}

func (f fakeFarms) CheckOwner(context.Context, int64, int64) error { return f.err }

type fakeProfiles struct{ calls int }

func (f *fakeProfiles) Profile(_ context.Context, code string, _ domain.Stage) (*domain.Profile, error) {
	f.calls++
	if code != "tomato" {
		return nil, domain.ErrCropNotFound
	}
	p := tomato()
	return &p, nil
}

func (f *fakeProfiles) Crop(_ context.Context, code string) (*domain.CropRef, error) {
	if code != "tomato" {
		return nil, domain.ErrCropNotFound
	}
	return &domain.CropRef{ID: 1, Code: code}, nil
}

func TestPreviewService(t *testing.T) {
	ctx := context.Background()
	farm := farmWith(water.Analysis{})

	profiles := &fakeProfiles{}
	_, err := NewPreviewService(fakeFarms{err: domain.ErrFarmNotFound}, profiles).Preview(ctx, 1, 2, direct(100))
	require.ErrorIs(t, err, domain.ErrFarmNotFound)
	require.Zero(t, profiles.calls, "begona xo'jalik uchun profil so'ralmaydi")

	svc := NewPreviewService(fakeFarms{data: &farm}, profiles)
	in := direct(100)
	in.Stage = 0
	_, err = svc.Preview(ctx, 1, 1, in)
	require.ErrorIs(t, err, domain.ErrInvalidStage)

	in = direct(100)
	in.CropCode = "rice"
	_, err = svc.Preview(ctx, 1, 1, in)
	require.ErrorIs(t, err, domain.ErrCropNotFound)

	p, err := svc.Preview(ctx, 1, 1, direct(100))
	require.NoError(t, err)
	require.Equal(t, "tomato", p.Profile.CropCode)
}
