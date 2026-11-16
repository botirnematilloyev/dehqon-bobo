package service

import (
	"context"
	"testing"

	"dehqon-bobo/internal/modules/farm/domain"

	"github.com/stretchr/testify/require"
)

func TestWaterEstimate_ComesFromCatalog(t *testing.T) {
	ctx := context.Background()

	t.Run("regional default wins over national", func(t *testing.T) {
		f := newFixture()
		got, err := f.farmSvc.Onboard(ctx, alice, OnboardInput{RegionID: 11, WaterSource: domain.WaterWell})
		require.NoError(t, err)
		require.Equal(t, 140.0, got.Water.Ca, "Region 11 well water from the agronomist's table")
		require.True(t, got.Water.IsEstimate)
	})

	t.Run("national default otherwise", func(t *testing.T) {
		f := newFixture()
		got, err := f.farmSvc.Onboard(ctx, alice, OnboardInput{RegionID: 10, WaterSource: domain.WaterWell})
		require.NoError(t, err)
		require.Equal(t, 80.0, got.Water.Ca)
	})

	t.Run("agronomist change is picked up without a deploy", func(t *testing.T) {
		f := newFixture()
		farm, _ := f.farmSvc.Create(ctx, alice, FarmInput{RegionID: ptr(int32(10))})
		f.waterDefs.global[domain.WaterTap] = domain.WaterAnalysis{EC: 0.7, Ca: 65, Mg: 20, HCO3Mmol: 3.2}

		w, err := f.waterSvc.Record(ctx, alice, farm.ID, WaterInput{Source: domain.WaterTap, Estimate: true})
		require.NoError(t, err)
		require.Equal(t, 65.0, w.Ca)
	})

	t.Run("missing default is an explicit error, not a guess", func(t *testing.T) {
		f := newFixture()
		delete(f.waterDefs.global, domain.WaterCanal)
		_, err := f.farmSvc.Onboard(ctx, alice, OnboardInput{RegionID: 10, WaterSource: domain.WaterCanal})
		require.ErrorIs(t, err, domain.ErrWaterDefaultMissing)
		require.Empty(t, f.onboarder.calls)
	})
}

func TestSettings_Fertilizers(t *testing.T) {
	ctx := context.Background()
	f := newFixture()
	farm, _ := f.farmSvc.Create(ctx, alice, FarmInput{})

	list, err := f.settingsSvc.Fertilizers(ctx, alice, farm.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.False(t, list[0].Overridden)

	eff, err := f.settingsSvc.SetFertilizer(ctx, alice, farm.ID, "CaNO3", FertilizerOverrideInput{
		Brand: ptr("Yara"), Elements: map[string]float64{"Ca": 19, "N_NO3": 14.4, "N_NH4": 1.1},
	})
	require.NoError(t, err)
	require.Equal(t, 19.0, eff.Elements["Ca"])
	require.Equal(t, 600.0, *eff.StockGL, "stock not overridden → default")

	list, _ = f.settingsSvc.Fertilizers(ctx, alice, farm.ID)
	require.True(t, list[0].Overridden)

	require.NoError(t, f.settingsSvc.ResetFertilizer(ctx, alice, farm.ID, "CaNO3"))
	list, _ = f.settingsSvc.Fertilizers(ctx, alice, farm.ID)
	require.False(t, list[0].Overridden)
	require.Equal(t, 16.972, list[0].Elements["Ca"])

	_, err = f.settingsSvc.SetFertilizer(ctx, alice, farm.ID, "Unobtainium", FertilizerOverrideInput{})
	require.ErrorIs(t, err, domain.ErrFertilizerNotFound)

	_, err = f.settingsSvc.SetFertilizer(ctx, alice, farm.ID, "HNO3_38", FertilizerOverrideInput{DensityGML: ptr(5.0)})
	require.ErrorIs(t, err, domain.ErrInvalidSetting)

	_, err = f.settingsSvc.Fertilizers(ctx, bob, farm.ID)
	require.ErrorIs(t, err, domain.ErrFarmNotFound, "other users cannot read or change it")
	_, err = f.settingsSvc.SetFertilizer(ctx, bob, farm.ID, "CaNO3", FertilizerOverrideInput{})
	require.ErrorIs(t, err, domain.ErrFarmNotFound)
}

func TestSettings_Parameters(t *testing.T) {
	ctx := context.Background()
	f := newFixture()
	farm, _ := f.farmSvc.Create(ctx, alice, FarmInput{})

	p, err := f.settingsSvc.SetParameter(ctx, alice, farm.ID, "residual_hco3_mmol", 1.0)
	require.NoError(t, err)
	require.Equal(t, 1.0, p.Value)
	require.Equal(t, 0.5, p.Default)

	list, _ := f.settingsSvc.Parameters(ctx, alice, farm.ID)
	require.Equal(t, 1.0, list[0].Value)
	require.True(t, list[0].Overridden)

	_, err = f.settingsSvc.SetParameter(ctx, alice, farm.ID, "residual_hco3_mmol", 9)
	require.ErrorIs(t, err, domain.ErrInvalidSetting, "outside the agronomist's range")

	_, err = f.settingsSvc.SetParameter(ctx, alice, farm.ID, "water_ion_balance_tolerance_pct", 30)
	require.ErrorIs(t, err, domain.ErrParameterNotOverridable)

	_, err = f.settingsSvc.SetParameter(ctx, alice, farm.ID, "no_such_key", 1)
	require.ErrorIs(t, err, domain.ErrParameterNotFound)

	require.NoError(t, f.settingsSvc.ResetParameter(ctx, alice, farm.ID, "residual_hco3_mmol"))
	list, _ = f.settingsSvc.Parameters(ctx, alice, farm.ID)
	require.Equal(t, 0.5, list[0].Value)
	require.False(t, list[0].Overridden)
}
