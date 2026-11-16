//go:build integration

package repository_test

import (
	"context"
	"errors"
	"testing"

	"dehqon-bobo/internal/modules/farm/domain"
	"dehqon-bobo/internal/modules/farm/repository"
	"dehqon-bobo/internal/modules/farm/service"
	"dehqon-bobo/internal/platform/db"

	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func newFarm(ownerID int64) *domain.Farm {
	return &domain.Farm{
		OwnerID: ownerID, SystemType: domain.SystemSubstrate, MixingMethod: domain.MixingDirect,
		CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt,
	}
}

func TestFarmRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("round-trips every column including numerics", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewFarmRepository(tx)
		owner := createUser(t, tx)
		districtID, regionID := seededDistrict(t, tx)

		f := newFarm(owner)
		f.Name = ptr("Alining issiqxonasi")
		f.RegionID, f.DistrictID = &regionID, &districtID
		f.MixingMethod, f.InjectorRatio, f.StockTankL = domain.MixingAB, ptr(int32(100)), ptr(150.5)
		require.NoError(t, repo.Create(ctx, f))
		require.NotZero(t, f.ID)

		got, err := repo.FindByID(ctx, f.ID)
		require.NoError(t, err)
		require.Equal(t, f, got)

		got.SystemType = domain.SystemHydroponics
		got.UpdatedAt = testUpdatedAt + 1
		require.NoError(t, repo.Update(ctx, got))

		again, err := repo.FindByID(ctx, f.ID)
		require.NoError(t, err)
		require.Equal(t, domain.SystemHydroponics, again.SystemType)
	})

	t.Run("lists only the owner's farms in id order", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewFarmRepository(tx)
		alice, bob := createUser(t, tx), createUser(t, tx)

		a1, a2, b1 := newFarm(alice), newFarm(alice), newFarm(bob)
		for _, f := range []*domain.Farm{a1, a2, b1} {
			require.NoError(t, repo.Create(ctx, f))
		}

		list, err := repo.ListByOwner(ctx, alice)
		require.NoError(t, err)
		require.Len(t, list, 2)
		require.Equal(t, a1.ID, list[0].ID)
		require.Equal(t, a2.ID, list[1].ID)
	})

	t.Run("unknown region is ErrInvalidLocation", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewFarmRepository(tx)
		f := newFarm(createUser(t, tx))
		f.RegionID = ptr(int32(999999))
		require.ErrorIs(t, repo.Create(ctx, f), domain.ErrInvalidLocation)
	})

	t.Run("district region lookup", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewFarmRepository(tx)
		districtID, regionID := seededDistrict(t, tx)

		got, err := repo.DistrictRegion(ctx, districtID)
		require.NoError(t, err)
		require.Equal(t, regionID, got)

		_, err = repo.DistrictRegion(ctx, 999999)
		require.ErrorIs(t, err, domain.ErrInvalidLocation)
	})

	t.Run("delete cascades and missing farm is not found", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewFarmRepository(tx)
		plots := repository.NewPlotRepository(tx)

		f := newFarm(createUser(t, tx))
		require.NoError(t, repo.Create(ctx, f))
		require.NoError(t, plots.Create(ctx, &domain.Plot{FarmID: f.ID, Name: "1", CreatedAt: testCreatedAt, UpdatedAt: testCreatedAt}))

		require.NoError(t, repo.Delete(ctx, f.ID))
		left, err := plots.ListByFarm(ctx, f.ID, true)
		require.NoError(t, err)
		require.Empty(t, left)

		require.ErrorIs(t, repo.Delete(ctx, f.ID), domain.ErrFarmNotFound)
		_, err = repo.FindByID(ctx, f.ID)
		require.ErrorIs(t, err, domain.ErrFarmNotFound)
	})
}

func TestPlotRepository(t *testing.T) {
	ctx := context.Background()
	tx := beginTx(t)
	farms := repository.NewFarmRepository(tx)
	repo := repository.NewPlotRepository(tx)

	f := newFarm(createUser(t, tx))
	require.NoError(t, farms.Create(ctx, f))

	active := &domain.Plot{FarmID: f.ID, Name: "1-issiqxona", AreaM2: ptr(120.5), CreatedAt: testCreatedAt, UpdatedAt: testCreatedAt}
	old := &domain.Plot{FarmID: f.ID, Name: "Eski", CreatedAt: testCreatedAt, UpdatedAt: testCreatedAt}
	require.NoError(t, repo.Create(ctx, active))
	require.NoError(t, repo.Create(ctx, old))

	old.IsArchived = true
	require.NoError(t, repo.Update(ctx, old))

	list, err := repo.ListByFarm(ctx, f.ID, false)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, active.ID, list[0].ID)
	require.Equal(t, 120.5, *list[0].AreaM2)

	all, err := repo.ListByFarm(ctx, f.ID, true)
	require.NoError(t, err)
	require.Len(t, all, 2)
	require.True(t, all[1].IsArchived, "archived come last")

	_, err = repo.FindByID(ctx, 999999)
	require.ErrorIs(t, err, domain.ErrPlotNotFound)
}

func TestWaterAnalysisRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("one active analysis per farm, history kept", func(t *testing.T) {
		tx := beginTx(t)
		farms := repository.NewFarmRepository(tx)
		repo := repository.NewWaterAnalysisRepository(tx)
		f := newFarm(createUser(t, tx))
		require.NoError(t, farms.Create(ctx, f))

		est := domain.WaterAnalysis{Source: domain.WaterWell, IsEstimate: true, EC: 1, Ca: 80, Mg: 30, HCO3Mmol: 4}
		est.FarmID, est.CreatedAt = f.ID, testCreatedAt
		require.NoError(t, repo.Create(ctx, &est))

		lab := domain.WaterAnalysis{
			FarmID: f.ID, Source: domain.WaterWell, EC: 1.35, Ca: 101.5, Mg: 33, HCO3Mmol: 4.8,
			PH: ptr(7.6), LabName: ptr("Agrolab"), CreatedAt: testCreatedAt + 10,
		}
		require.NoError(t, repo.DeactivateAll(ctx, f.ID))
		require.NoError(t, repo.Create(ctx, &lab))

		active, err := repo.FindActive(ctx, f.ID)
		require.NoError(t, err)
		require.Equal(t, lab.ID, active.ID)
		require.False(t, active.IsEstimate)
		require.Equal(t, 1.35, active.EC)
		require.Equal(t, 7.6, *active.PH)
		require.Nil(t, active.Na)

		history, err := repo.ListByFarm(ctx, f.ID, 0)
		require.NoError(t, err)
		require.Len(t, history, 2)
		require.Equal(t, lab.ID, history[0].ID, "newest first")
		require.False(t, history[1].IsActive)
	})

	t.Run("database refuses a second active analysis", func(t *testing.T) {
		tx := beginTx(t)
		farms := repository.NewFarmRepository(tx)
		repo := repository.NewWaterAnalysisRepository(tx)
		f := newFarm(createUser(t, tx))
		require.NoError(t, farms.Create(ctx, f))

		a := domain.WaterAnalysis{Source: domain.WaterTap, IsEstimate: true, EC: 0.5, Ca: 50, Mg: 15, HCO3Mmol: 3}
		a.FarmID, a.CreatedAt = f.ID, testCreatedAt
		require.NoError(t, repo.Create(ctx, &a))

		b := a
		err := repo.Create(ctx, &b)
		require.True(t, db.IsUniqueViolation(err), "uq_water_analyses_active must hold: %v", err)
	})

	t.Run("no analysis is not found", func(t *testing.T) {
		tx := beginTx(t)
		farms := repository.NewFarmRepository(tx)
		f := newFarm(createUser(t, tx))
		require.NoError(t, farms.Create(ctx, f))

		_, err := repository.NewWaterAnalysisRepository(tx).FindActive(ctx, f.ID)
		require.ErrorIs(t, err, domain.ErrWaterAnalysisNotFound)
	})
}

// sqlOnboarder — user modulining MarkOnboarded'ini taqlid qiladi, lekin
// HAQIQIY bazaga ctx dagi tranzaksiya orqali yozadi. Shu bilan modullararo
// tranzaksiya (db.FromContext) haqiqatan bitta ekanini tekshiramiz.
type sqlOnboarder struct{ fail bool }

func (o sqlOnboarder) MarkOnboarded(ctx context.Context, userID int64, regionID, districtID *int32, at int64) error {
	q := db.FromContext(ctx, testPool)
	if _, err := q.Exec(ctx, `UPDATE users SET region_id = $2, district_id = $3, onboarded_at = $4 WHERE id = $1`,
		userID, regionID, districtID, at); err != nil {
		return err
	}
	if o.fail {
		return errors.New("onboarder failed after writing")
	}
	return nil
}

// sqlWaterDefaults — catalog modulini import qilmasdan, seed qilingan
// water_source_defaults jadvalidan o'qiydi (viloyat → mamlakat).
type sqlWaterDefaults struct{}

func (sqlWaterDefaults) Estimate(ctx context.Context, source domain.WaterSource, regionID *int32) (domain.WaterAnalysis, error) {
	var w domain.WaterAnalysis
	err := testPool.QueryRow(ctx, `
		SELECT ec::float8, ca::float8, mg::float8, hco3_mmol::float8 FROM water_source_defaults
		WHERE source = $1 AND (region_id = $2 OR region_id IS NULL)
		ORDER BY region_id NULLS LAST LIMIT 1`, source, regionID).Scan(&w.EC, &w.Ca, &w.Mg, &w.HCO3Mmol)
	if err != nil {
		return domain.WaterAnalysis{}, domain.ErrWaterDefaultMissing
	}
	return w, nil
}

func newServiceOnPool(onboarder service.UserOnboarder) *service.FarmService {
	return service.NewFarmService(
		repository.NewFarmRepository(testPool),
		repository.NewPlotRepository(testPool),
		repository.NewWaterAnalysisRepository(testPool),
		sqlWaterDefaults{},
		onboarder,
		db.NewTxManager(testPool),
	)
}

func TestOnboard_TransactionAcrossModules(t *testing.T) {
	ctx := context.Background()
	districtID, regionID := seededDistrict(t, testPool)

	t.Run("commits farm, water and user flag together", func(t *testing.T) {
		userID := createUser(t, testPool)
		svc := newServiceOnPool(sqlOnboarder{})

		got, err := svc.Onboard(ctx, userID, service.OnboardInput{RegionID: regionID, DistrictID: &districtID, WaterSource: domain.WaterCanal})
		require.NoError(t, err)

		var onboarded *int64
		require.NoError(t, testPool.QueryRow(ctx, `SELECT onboarded_at FROM users WHERE id = $1`, userID).Scan(&onboarded))
		require.NotNil(t, onboarded)

		water, err := repository.NewWaterAnalysisRepository(testPool).FindActive(ctx, got.Farm.ID)
		require.NoError(t, err)
		require.Equal(t, domain.WaterCanal, water.Source)

		// Qayta o'tish yangi xo'jalik yaratmaydi.
		again, err := svc.Onboard(ctx, userID, service.OnboardInput{RegionID: regionID, WaterSource: domain.WaterFilter})
		require.NoError(t, err)
		require.Equal(t, got.Farm.ID, again.Farm.ID)
	})

	t.Run("rolls everything back when the user write fails", func(t *testing.T) {
		userID := createUser(t, testPool)
		svc := newServiceOnPool(sqlOnboarder{fail: true})

		_, err := svc.Onboard(ctx, userID, service.OnboardInput{RegionID: regionID, WaterSource: domain.WaterWell})
		require.Error(t, err)

		farms, err := repository.NewFarmRepository(testPool).ListByOwner(ctx, userID)
		require.NoError(t, err)
		require.Empty(t, farms, "farm insert must be rolled back")

		var onboarded *int64
		require.NoError(t, testPool.QueryRow(ctx, `SELECT onboarded_at FROM users WHERE id = $1`, userID).Scan(&onboarded))
		require.Nil(t, onboarded, "user write must be rolled back too")
	})
}

func TestSettingsRepository(t *testing.T) {
	ctx := context.Background()
	tx := beginTx(t)
	farms := repository.NewFarmRepository(tx)
	repo := repository.NewSettingsRepository(tx)

	f := newFarm(createUser(t, tx))
	require.NoError(t, farms.Create(ctx, f))

	t.Run("fertilizer override round-trip and replace", func(t *testing.T) {
		o := &domain.FertilizerOverride{
			FarmID: f.ID, Code: "CaNO3", Brand: ptr("Yara"),
			Elements: map[string]float64{"Ca": 19, "N_NO3": 14.4, "N_NH4": 1.1},
			StockGL:  ptr(500.0), PriceUZSPerKg: ptr(int64(14000)), UpdatedAt: testCreatedAt,
		}
		require.NoError(t, repo.UpsertFertilizerOverride(ctx, o))

		list, err := repo.ListFertilizerOverrides(ctx, f.ID)
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Equal(t, "CaNO3", list[0].Code)
		require.Equal(t, 1.1, list[0].Elements["N_NH4"])
		require.Equal(t, 500.0, *list[0].StockGL)
		require.Equal(t, int64(14000), *list[0].PriceUZSPerKg)

		// PUT semantikasi: faqat narx qolsa, tarkib standartga qaytadi (NULL).
		require.NoError(t, repo.UpsertFertilizerOverride(ctx, &domain.FertilizerOverride{
			FarmID: f.ID, Code: "CaNO3", PriceUZSPerKg: ptr(int64(15000)), UpdatedAt: testUpdatedAt,
		}))
		list, _ = repo.ListFertilizerOverrides(ctx, f.ID)
		require.Len(t, list, 1)
		require.Nil(t, list[0].Elements)
		require.Nil(t, list[0].StockGL)

		require.NoError(t, repo.DeleteFertilizerOverride(ctx, f.ID, "CaNO3"))
		list, _ = repo.ListFertilizerOverrides(ctx, f.ID)
		require.Empty(t, list)
	})

	t.Run("acid override", func(t *testing.T) {
		require.NoError(t, repo.UpsertFertilizerOverride(ctx, &domain.FertilizerOverride{
			FarmID: f.ID, Code: "HNO3_60", ConcentrationPct: ptr(56.0), DensityGML: ptr(1.345), UpdatedAt: testCreatedAt,
		}))
		list, _ := repo.ListFertilizerOverrides(ctx, f.ID)
		require.Equal(t, 56.0, *list[0].ConcentrationPct)
		require.Equal(t, 1.345, *list[0].DensityGML)
	})

	t.Run("unknown fertilizer code", func(t *testing.T) {
		err := repo.UpsertFertilizerOverride(ctx, &domain.FertilizerOverride{FarmID: f.ID, Code: "Unobtainium", UpdatedAt: 1})
		require.ErrorIs(t, err, domain.ErrFertilizerNotFound)
	})

	t.Run("parameter overrides", func(t *testing.T) {
		require.NoError(t, repo.UpsertParamOverride(ctx, f.ID, "residual_hco3_mmol", 1.0, testCreatedAt))
		require.NoError(t, repo.UpsertParamOverride(ctx, f.ID, "residual_hco3_mmol", 0.8, testUpdatedAt))
		got, err := repo.ListParamOverrides(ctx, f.ID)
		require.NoError(t, err)
		require.Equal(t, map[string]float64{"residual_hco3_mmol": 0.8}, got)

		require.NoError(t, repo.DeleteParamOverride(ctx, f.ID, "residual_hco3_mmol"))
		got, _ = repo.ListParamOverrides(ctx, f.ID)
		require.Empty(t, got)
	})

	t.Run("unknown parameter key", func(t *testing.T) {
		// FK xatosi tranzaksiyani to'xtatadi — alohida tranzaksiyada.
		tx2 := beginTx(t)
		f2 := newFarm(createUser(t, tx2))
		require.NoError(t, repository.NewFarmRepository(tx2).Create(ctx, f2))
		err := repository.NewSettingsRepository(tx2).UpsertParamOverride(ctx, f2.ID, "no_such_key", 1, 1)
		require.ErrorIs(t, err, domain.ErrParameterNotFound)
	})
}

func TestOnboard_UsesSeededWaterDefaults(t *testing.T) {
	ctx := context.Background()
	_, regionID := seededDistrict(t, testPool)
	userID := createUser(t, testPool)

	got, err := newServiceOnPool(sqlOnboarder{}).Onboard(ctx, userID, service.OnboardInput{RegionID: regionID, WaterSource: domain.WaterWell})
	require.NoError(t, err)
	require.True(t, got.Water.IsEstimate)
	require.Equal(t, 80.0, got.Water.Ca, "value comes from water_source_defaults seed, not from code")
	require.Equal(t, 4.0, got.Water.HCO3Mmol)
}
