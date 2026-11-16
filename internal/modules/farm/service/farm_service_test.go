package service

import (
	"context"
	"errors"
	"testing"

	"dehqon-bobo/internal/modules/farm/domain"

	"github.com/stretchr/testify/require"
)

const (
	alice = int64(1)
	bob   = int64(2)
)

func TestFarmService_Onboard(t *testing.T) {
	ctx := context.Background()

	t.Run("creates farm, estimated water and marks the user", func(t *testing.T) {
		f := newFixture()
		sys := domain.SystemSubstrate

		got, err := f.farmSvc.Onboard(ctx, alice, OnboardInput{
			RegionID: 10, DistrictID: ptr(int32(20)), WaterSource: domain.WaterWell, SystemType: &sys,
		})
		require.NoError(t, err)

		require.NotZero(t, got.Farm.ID)
		require.Equal(t, alice, got.Farm.OwnerID)
		require.Equal(t, int32(10), *got.Farm.RegionID)
		require.Equal(t, domain.SystemSubstrate, got.Farm.SystemType)
		require.Equal(t, domain.MixingDirect, got.Farm.MixingMethod)

		require.NotNil(t, got.Water)
		require.True(t, got.Water.IsEstimate)
		require.True(t, got.Water.IsActive)
		require.Equal(t, domain.WaterWell, got.Water.Source)
		require.Equal(t, alice, *got.Water.CreatedBy)

		require.Len(t, f.onboarder.calls, 1)
		require.Equal(t, alice, f.onboarder.calls[0].userID)
		require.Equal(t, int32(20), *f.onboarder.calls[0].districtID)
		require.Equal(t, 1, f.tx.calls, "everything in one transaction")
	})

	t.Run("defaults to greenhouse soil when system is not given", func(t *testing.T) {
		f := newFixture()
		got, err := f.farmSvc.Onboard(ctx, alice, OnboardInput{RegionID: 10, WaterSource: domain.WaterTap})
		require.NoError(t, err)
		require.Equal(t, domain.SystemGreenhouseSoil, got.Farm.SystemType)
	})

	t.Run("is idempotent: second run updates the same farm and replaces water", func(t *testing.T) {
		f := newFixture()
		first, err := f.farmSvc.Onboard(ctx, alice, OnboardInput{RegionID: 10, WaterSource: domain.WaterWell})
		require.NoError(t, err)

		second, err := f.farmSvc.Onboard(ctx, alice, OnboardInput{RegionID: 11, WaterSource: domain.WaterFilter})
		require.NoError(t, err)

		require.Equal(t, first.Farm.ID, second.Farm.ID)
		farms, _ := f.farms.ListByOwner(ctx, alice)
		require.Len(t, farms, 1)
		require.Equal(t, int32(11), *farms[0].RegionID)

		active, err := f.water.FindActive(ctx, first.Farm.ID)
		require.NoError(t, err)
		require.Equal(t, domain.WaterFilter, active.Source)
		require.Equal(t, 2, f.water.countFor(first.Farm.ID), "old analysis is kept as history")
	})

	t.Run("rejects a district from another region", func(t *testing.T) {
		f := newFixture()
		_, err := f.farmSvc.Onboard(ctx, alice, OnboardInput{RegionID: 10, DistrictID: ptr(int32(30)), WaterSource: domain.WaterWell})
		require.ErrorIs(t, err, domain.ErrInvalidLocation)
		require.Empty(t, f.onboarder.calls)
	})

	t.Run("rejects an unknown water source before touching storage", func(t *testing.T) {
		f := newFixture()
		_, err := f.farmSvc.Onboard(ctx, alice, OnboardInput{RegionID: 10, WaterSource: 0})
		require.ErrorIs(t, err, domain.ErrInvalidWaterSource)
		require.Zero(t, f.tx.calls)
	})

	t.Run("propagates onboarder failure", func(t *testing.T) {
		f := newFixture()
		boom := errors.New("boom")
		f.onboarder.err = boom
		_, err := f.farmSvc.Onboard(ctx, alice, OnboardInput{RegionID: 10, WaterSource: domain.WaterWell})
		require.ErrorIs(t, err, boom)
	})
}

func TestFarmService_Ownership(t *testing.T) {
	ctx := context.Background()
	f := newFixture()
	farm, err := f.farmSvc.Create(ctx, alice, FarmInput{Name: ptr("Alining xo'jaligi")})
	require.NoError(t, err)

	t.Run("owner sees the overview", func(t *testing.T) {
		got, err := f.farmSvc.Get(ctx, alice, farm.ID)
		require.NoError(t, err)
		require.Equal(t, farm.ID, got.Farm.ID)
		require.Nil(t, got.Water, "no analysis yet is not an error")
		require.Empty(t, got.Plots)
	})

	t.Run("someone else gets not found, not forbidden", func(t *testing.T) {
		_, err := f.farmSvc.Get(ctx, bob, farm.ID)
		require.ErrorIs(t, err, domain.ErrFarmNotFound)

		_, err = f.farmSvc.Update(ctx, bob, farm.ID, FarmInput{Name: ptr("o'g'irlangan")})
		require.ErrorIs(t, err, domain.ErrFarmNotFound)

		require.ErrorIs(t, f.farmSvc.Delete(ctx, bob, farm.ID), domain.ErrFarmNotFound)
		_, stillThere := f.farms.farms[farm.ID]
		require.True(t, stillThere)
	})

	t.Run("list returns only own farms", func(t *testing.T) {
		_, err := f.farmSvc.Create(ctx, bob, FarmInput{})
		require.NoError(t, err)

		mine, err := f.farmSvc.List(ctx, alice)
		require.NoError(t, err)
		require.Len(t, mine, 1)
		require.Equal(t, farm.ID, mine[0].ID)
	})
}

func TestFarmService_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("switching to A/B requires injector settings", func(t *testing.T) {
		f := newFixture()
		farm, _ := f.farmSvc.Create(ctx, alice, FarmInput{})
		ab := domain.MixingAB

		_, err := f.farmSvc.Update(ctx, alice, farm.ID, FarmInput{MixingMethod: &ab})
		require.ErrorIs(t, err, domain.ErrInvalidMixing)

		got, err := f.farmSvc.Update(ctx, alice, farm.ID, FarmInput{
			MixingMethod: &ab, InjectorRatio: ptr(int32(100)), StockTankL: ptr(200.0),
		})
		require.NoError(t, err)
		require.Equal(t, int32(100), *got.InjectorRatio)
	})

	t.Run("switching back to direct clears injector settings", func(t *testing.T) {
		f := newFixture()
		ab, direct := domain.MixingAB, domain.MixingDirect
		farm, err := f.farmSvc.Create(ctx, alice, FarmInput{MixingMethod: &ab, InjectorRatio: ptr(int32(50)), StockTankL: ptr(100.0)})
		require.NoError(t, err)

		got, err := f.farmSvc.Update(ctx, alice, farm.ID, FarmInput{MixingMethod: &direct})
		require.NoError(t, err)
		require.Nil(t, got.InjectorRatio)
		require.Nil(t, got.StockTankL)
	})

	t.Run("changes only provided fields and trims the name", func(t *testing.T) {
		f := newFixture()
		sys := domain.SystemHydroponics
		farm, _ := f.farmSvc.Create(ctx, alice, FarmInput{Name: ptr("Eski"), SystemType: &sys})

		got, err := f.farmSvc.Update(ctx, alice, farm.ID, FarmInput{Name: ptr("  Yangi  ")})
		require.NoError(t, err)
		require.Equal(t, "Yangi", *got.Name)
		require.Equal(t, domain.SystemHydroponics, got.SystemType)
	})
}

func TestPlotService(t *testing.T) {
	ctx := context.Background()

	setup := func() (*fixture, int64, int64) {
		f := newFixture()
		mine, _ := f.farmSvc.Create(ctx, alice, FarmInput{})
		other, _ := f.farmSvc.Create(ctx, bob, FarmInput{})
		return f, mine.ID, other.ID
	}

	t.Run("creates and lists active plots", func(t *testing.T) {
		f, farmID, _ := setup()
		p, err := f.plotSvc.Create(ctx, alice, farmID, PlotInput{Name: ptr(" 1-issiqxona "), AreaM2: ptr(120.0)})
		require.NoError(t, err)
		require.Equal(t, "1-issiqxona", p.Name)

		list, err := f.plotSvc.List(ctx, alice, farmID, false)
		require.NoError(t, err)
		require.Len(t, list, 1)
	})

	t.Run("name is required", func(t *testing.T) {
		f, farmID, _ := setup()
		_, err := f.plotSvc.Create(ctx, alice, farmID, PlotInput{})
		require.ErrorIs(t, err, domain.ErrInvalidName)
	})

	t.Run("archive hides the plot but keeps it", func(t *testing.T) {
		f, farmID, _ := setup()
		p, _ := f.plotSvc.Create(ctx, alice, farmID, PlotInput{Name: ptr("Eski issiqxona")})

		require.NoError(t, f.plotSvc.Archive(ctx, alice, farmID, p.ID))

		active, _ := f.plotSvc.List(ctx, alice, farmID, false)
		require.Empty(t, active)
		all, _ := f.plotSvc.List(ctx, alice, farmID, true)
		require.Len(t, all, 1)
		require.True(t, all[0].IsArchived)
	})

	t.Run("cannot reach another farm's plot through own farm id", func(t *testing.T) {
		f, myFarm, otherFarm := setup()
		theirs, err := f.plotSvc.Create(ctx, bob, otherFarm, PlotInput{Name: ptr("Bobniki")})
		require.NoError(t, err)

		_, err = f.plotSvc.Update(ctx, alice, myFarm, theirs.ID, PlotInput{Name: ptr("meniki")})
		require.ErrorIs(t, err, domain.ErrPlotNotFound)

		_, err = f.plotSvc.List(ctx, alice, otherFarm, false)
		require.ErrorIs(t, err, domain.ErrFarmNotFound)
	})
}

func TestWaterService_Record(t *testing.T) {
	ctx := context.Background()

	setup := func() (*fixture, int64) {
		f := newFixture()
		farm, _ := f.farmSvc.Create(ctx, alice, FarmInput{})
		return f, farm.ID
	}

	t.Run("lab result needs ec, ca, mg and hco3", func(t *testing.T) {
		f, farmID := setup()
		_, err := f.waterSvc.Record(ctx, alice, farmID, WaterInput{Source: domain.WaterWell, EC: ptr(1.2)})
		require.ErrorIs(t, err, domain.ErrInvalidWaterValue)
	})

	t.Run("lab result becomes the only active analysis", func(t *testing.T) {
		f, farmID := setup()
		_, err := f.waterSvc.Record(ctx, alice, farmID, WaterInput{Source: domain.WaterWell, Estimate: true})
		require.NoError(t, err)

		got, err := f.waterSvc.Record(ctx, alice, farmID, WaterInput{
			Source: domain.WaterWell, EC: ptr(1.4), Ca: ptr(110.0), Mg: ptr(35.0), HCO3Mmol: ptr(5.0),
			LabName: ptr("  Agrolab  "),
		})
		require.NoError(t, err)
		require.False(t, got.IsEstimate)
		require.Equal(t, "Agrolab", *got.LabName)

		active, err := f.waterSvc.Active(ctx, alice, farmID)
		require.NoError(t, err)
		require.Equal(t, got.ID, active.ID)

		history, err := f.waterSvc.History(ctx, alice, farmID, 0)
		require.NoError(t, err)
		require.Len(t, history, 2)
		require.Equal(t, got.ID, history[0].ID, "newest first")
	})

	t.Run("rejects unrealistic values", func(t *testing.T) {
		f, farmID := setup()
		_, err := f.waterSvc.Record(ctx, alice, farmID, WaterInput{
			Source: domain.WaterWell, EC: ptr(40.0), Ca: ptr(1.0), Mg: ptr(1.0), HCO3Mmol: ptr(1.0),
		})
		require.ErrorIs(t, err, domain.ErrInvalidWaterValue)
	})

	t.Run("another user cannot record or read", func(t *testing.T) {
		f, farmID := setup()
		_, err := f.waterSvc.Record(ctx, bob, farmID, WaterInput{Source: domain.WaterWell, Estimate: true})
		require.ErrorIs(t, err, domain.ErrFarmNotFound)
		_, err = f.waterSvc.Active(ctx, bob, farmID)
		require.ErrorIs(t, err, domain.ErrFarmNotFound)
	})
}
