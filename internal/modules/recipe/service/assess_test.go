package service

import (
	"encoding/json"
	"testing"

	"dehqon-bobo/internal/modules/calc/water"
	"dehqon-bobo/internal/modules/recipe/domain"

	"github.com/stretchr/testify/require"
)

func measureParams() map[string]float64 {
	p := paramsFixture()
	p[ParamECTolerance], p[ParamDrainECExcess], p[ParamDrainPctMin], p[ParamDrainPctMax] = 10, 1.0, 20, 50
	return p
}

// savedBatch — saqlangan retsept: snapshot JSON orqali o'tadi (bazadagidek).
func savedBatch(t *testing.T, farm domain.FarmData, in PreviewInput, targetEC float64) (*domain.Batch, *Snapshot) {
	t.Helper()
	p, err := Compute(tomato(), farm, in)
	require.NoError(t, err)
	raw, err := json.Marshal(NewSnapshot(p))
	require.NoError(t, err)
	var snap Snapshot
	require.NoError(t, json.Unmarshal(raw, &snap))
	b := &domain.Batch{ID: 1, FarmID: 1, ProfileID: 1, Stage: domain.StageFruiting, Method: p.Method,
		VolumeL: p.VolumeL, StockTankL: p.StockTankL, InjectorRatio: p.InjectorRatio, TargetEC: targetEC, Snapshot: raw}
	return b, &snap
}

func meas(ec, ph *float64) domain.Measurement { return domain.Measurement{ECIn: ec, PHIn: ph} }

func TestAssess_EC(t *testing.T) {
	farm := farmWith(water.Analysis{EC: 0.5, Ca: 40, HCO3Mmol: 2, SO4: 30, Mg: 10})
	batch, snap := savedBatch(t, farm, direct(200, "HNO3_38"), 2.7)
	require.Equal(t, 0.5, snap.Water.EC)

	a, err := Assess(meas(f64(2.75), f64(6.0)), batch, snap, measureParams(), nil)
	require.NoError(t, err)
	require.Equal(t, StatusOK, a.EC.Status)
	require.Equal(t, StatusOK, a.PH.Status)
	require.Nil(t, a.Correction)
	require.Empty(t, a.Advice)

	// Yuqori: 200·(3.2 − 2.7)/(2.7 − 0.5) = 45.5 L suv.
	a, err = Assess(meas(f64(3.2), nil), batch, snap, measureParams(), nil)
	require.NoError(t, err)
	require.Equal(t, StatusHigh, a.EC.Status)
	require.Equal(t, 18.5, a.EC.DeviationPct)
	require.Equal(t, ActionAddWater, a.Correction.Action)
	require.Equal(t, 45.5, a.Correction.WaterL)
	require.Equal(t, []string{AdviceAddWater}, a.Advice)

	// Bakda 150 L qolgan bo'lsa — mutanosib kamroq.
	a, err = Assess(meas(f64(3.2), nil), batch, snap, measureParams(), f64(150))
	require.NoError(t, err)
	require.Equal(t, 34.1, a.Correction.WaterL)

	// Past: f = (2.7 − 2.2)/(2.2 − 0.5) = 0.294; kislota qo'shilmaydi.
	a, err = Assess(meas(f64(2.2), nil), batch, snap, measureParams(), nil)
	require.NoError(t, err)
	require.Equal(t, StatusLow, a.EC.Status)
	require.Equal(t, ActionAddSalts, a.Correction.Action)
	require.Equal(t, 0.294, a.Correction.Fraction)
	orig := map[string]float64{}
	for _, tk := range snap.Tanks {
		for _, it := range tk.Items {
			orig[it.Code] = it.Grams
		}
	}
	for _, it := range a.Correction.Items {
		require.NotEqual(t, "HNO3_38", it.Code)
		require.InDelta(t, orig[it.Code]*0.5/1.7, it.Grams, 0.06, it.Code)
	}
}

func TestAssess_ABInjector(t *testing.T) {
	farm := farmWith(water.Analysis{EC: 0.5, Ca: 40, HCO3Mmol: 2, SO4: 30, Mg: 10})
	farm.Method, farm.StockTankL, farm.InjectorRatio = domain.MethodAB, f64(100), i32(100)
	batch, snap := savedBatch(t, farm, PreviewInput{CropCode: "tomato", Stage: domain.StageFruiting, Acids: []string{"HNO3_38"}}, 2.7)

	// N' = 100 × (3.2 − 0.5)/(2.7 − 0.5) = 122.7 → 1:123 (ko'proq suyultirish).
	a, err := Assess(meas(f64(3.2), nil), batch, snap, measureParams(), nil)
	require.NoError(t, err)
	require.Equal(t, ActionAdjustInjectr, a.Correction.Action)
	require.Equal(t, int32(123), a.Correction.InjectorRatio)
}

func TestAssess_PHAndDrain(t *testing.T) {
	batch, snap := savedBatch(t, farmWith(water.Analysis{}), direct(100), 2.7)

	m := domain.Measurement{ECIn: f64(2.7), PHIn: f64(7.0), ECDrain: f64(4.0), DrainPct: f64(10)}
	a, err := Assess(m, batch, snap, measureParams(), nil)
	require.NoError(t, err)
	require.Equal(t, StatusHigh, a.PH.Status)
	require.Equal(t, 1.3, *a.Drain.ECExcess)
	require.Equal(t, StatusHigh, a.Drain.ECStatus)
	require.Equal(t, StatusLow, a.Drain.PctStatus)
	require.ElementsMatch(t, []string{AdvicePHHigh, AdviceDrainECHigh, AdviceDrainPctLow}, a.Advice)

	a, err = Assess(domain.Measurement{PHIn: f64(5.0)}, batch, snap, measureParams(), nil)
	require.NoError(t, err)
	require.Equal(t, []string{AdvicePHLow}, a.Advice)

	// Retseptsiz: faqat drenaj.
	a, err = Assess(domain.Measurement{ECIn: f64(2.5), ECDrain: f64(3.0), DrainPct: f64(30)}, nil, nil, measureParams(), nil)
	require.NoError(t, err)
	require.Nil(t, a.EC)
	require.Nil(t, a.PH)
	require.Equal(t, StatusOK, a.Drain.ECStatus)
	require.Equal(t, StatusOK, a.Drain.PctStatus)

	// Parametr yo'q — jim standart emas, xato.
	p := measureParams()
	delete(p, ParamDrainPctMax)
	_, err = Assess(domain.Measurement{DrainPct: f64(30)}, nil, nil, p, nil)
	require.ErrorIs(t, err, domain.ErrCatalogIncomplete)
}

func TestCorrect_Recirculating(t *testing.T) {
	farm := farmWith(water.Analysis{})
	batch, snap := savedBatch(t, farm, direct(200), 2.7)
	ec := snap.EC.Estimated

	// EC o'zgarmagan: tuzatuvchi eritma = shu retsept, 150 L uchun.
	r, err := Correct(batch, snap, farm, CorrectionInput{ECNow: ec, VolumeNowL: 50, VolumeFinalL: 200})
	require.NoError(t, err)
	require.False(t, r.DiluteOnly)
	require.Equal(t, 150.0, r.CorrectionVolumeL)
	fresh, err := Compute(tomato(), farm, direct(150))
	require.NoError(t, err)
	require.Equal(t, items(fresh.Tanks[0]), items(r.Recipe.Tanks[0]))
	requireNear(t, fresh.Achieved, r.Recipe.Achieved, 0.05)

	// 25% quyuqlashgan: C_tuz = (200·C − 100·1.25C)/100 = 0.75·C.
	r, err = Correct(batch, snap, farm, CorrectionInput{ECNow: ec * 1.25, VolumeNowL: 100, VolumeFinalL: 200})
	require.NoError(t, err)
	require.InDelta(t, 0.75*snap.AchievedMgL.K, r.CorrectionMgL.K, 0.01)
	require.InDelta(t, 0.75*snap.AchievedMgL.K, r.Recipe.Achieved.K, 0.05)

	// Juda quyuq: tuz emas, suv. 100·2.5 − 100 = 150 L (suv EC = 0).
	r, err = Correct(batch, snap, farm, CorrectionInput{ECNow: ec * 2.5, VolumeNowL: 100, VolumeFinalL: 200})
	require.NoError(t, err)
	require.True(t, r.DiluteOnly)
	require.Nil(t, r.Recipe)
	require.Equal(t, 150.0, r.WaterToAddL)

	for _, in := range []CorrectionInput{
		{ECNow: 0, VolumeNowL: 50, VolumeFinalL: 200},
		{ECNow: 2, VolumeNowL: 200, VolumeFinalL: 200},
		{ECNow: 2, VolumeNowL: 0, VolumeFinalL: 200},
	} {
		_, err := Correct(batch, snap, farm, in)
		require.ErrorIs(t, err, domain.ErrInvalidCorrection)
	}
}
