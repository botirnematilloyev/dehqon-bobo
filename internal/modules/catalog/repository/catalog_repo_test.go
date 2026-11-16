//go:build integration

package repository_test

import (
	"context"
	"testing"

	"dehqon-bobo/internal/modules/calc/calctest"
	"dehqon-bobo/internal/modules/calc/inputs"
	"dehqon-bobo/internal/modules/catalog/domain"
	"dehqon-bobo/internal/modules/catalog/repository"

	"github.com/stretchr/testify/require"
)

func toInputs(fs []domain.Fertilizer) []inputs.Fertilizer {
	out := make([]inputs.Fertilizer, 0, len(fs))
	for _, f := range fs {
		out = append(out, inputs.Fertilizer{
			Code: f.Code, Kind: int16(f.Kind), Formula: f.Formula, Elements: f.Elements,
			ConcentrationPct: f.ConcentrationPct, DensityGML: f.DensityGML,
		})
	}
	return out
}

func TestSeed_ReproducesHPG(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewCatalogRepository(testPool)

	list, err := repo.ListFertilizers(ctx)
	require.NoError(t, err)
	require.Len(t, list, 22, "8 macro + 8 micro + 6 acids")

	s, err := inputs.Salts(toInputs(list))
	require.NoError(t, err)
	require.Equal(t, calctest.PureSalts(), s, "seeded compositions must equal HPG's saltCompositions")

	micro := inputs.MicroPercent(toInputs(list))
	require.Equal(t, 13.0, micro.Fe)
	require.Equal(t, 54.3, micro.Mo)

	acids, err := inputs.Acids(toInputs(list), []string{"H3PO4_85", "HNO3_38"})
	require.NoError(t, err)
	require.Len(t, acids, 2)

	params, err := repo.ListParameters(ctx)
	require.NoError(t, err)
	values := map[string]float64{}
	for _, p := range params {
		values[p.Key] = p.Value
	}
	got, err := inputs.ParamsFrom(values)
	require.NoError(t, err, "every parameter the calculator needs is seeded")
	require.Equal(t, 0.5, got.ResidualHCO3)
	require.True(t, got.Mode.UseK2SO4)
	require.Equal(t, -1, got.WeightDecimals)
}

func TestFertilizer_UpdateReplacesElements(t *testing.T) {
	ctx := context.Background()
	tx := beginTx(t)
	repo := repository.NewCatalogRepository(tx)

	f, err := repo.FindFertilizer(ctx, "CaNO3")
	require.NoError(t, err)
	require.Equal(t, 16.972, f.Elements["Ca"])

	agronomist := createUser(t, tx)
	at := int64(1763300000)
	f.Elements = map[string]float64{"Ca": 19, "N_NO3": 14.4, "N_NH4": 1.1}
	f.IsReviewed, f.UpdatedBy, f.UpdatedAt = true, &agronomist, &at
	require.NoError(t, repo.UpdateFertilizer(ctx, f))

	again, err := repo.FindFertilizer(ctx, "CaNO3")
	require.NoError(t, err)
	require.Equal(t, map[string]float64{"Ca": 19, "N_NO3": 14.4, "N_NH4": 1.1}, again.Elements)
	require.True(t, again.IsReviewed)
	require.Equal(t, agronomist, *again.UpdatedBy)

	_, err = repo.FindFertilizer(ctx, "nope")
	require.ErrorIs(t, err, domain.ErrFertilizerNotFound)
}

func TestWaterDefaults_RegionalOverridesNational(t *testing.T) {
	ctx := context.Background()
	tx := beginTx(t)
	repo := repository.NewCatalogRepository(tx)
	_, regionID := seededDistrict(t, tx)

	national, err := repo.FindWaterDefault(ctx, 1, &regionID)
	require.NoError(t, err)
	require.Nil(t, national.RegionID, "no regional row yet → national")
	require.Equal(t, 80.0, national.Ca)

	require.NoError(t, repo.UpsertWaterDefault(ctx, &domain.WaterDefault{
		Source: 1, RegionID: &regionID, EC: 1.6, Ca: 140, Mg: 45, HCO3Mmol: 5.5, UpdatedAt: 1,
	}))
	regional, err := repo.FindWaterDefault(ctx, 1, &regionID)
	require.NoError(t, err)
	require.Equal(t, 140.0, regional.Ca)

	// Upsert yangilaydi, ikkinchi qator yaratmaydi.
	require.NoError(t, repo.UpsertWaterDefault(ctx, &domain.WaterDefault{
		Source: 1, RegionID: &regionID, EC: 1.7, Ca: 150, Mg: 45, HCO3Mmol: 5.5, UpdatedAt: 2,
	}))
	regional, _ = repo.FindWaterDefault(ctx, 1, &regionID)
	require.Equal(t, 150.0, regional.Ca)

	// Boshqa viloyat hali mamlakat qiymatini oladi.
	other := regionID + 1
	n, err := repo.FindWaterDefault(ctx, 1, &other)
	require.NoError(t, err)
	require.Equal(t, 80.0, n.Ca)

	// Mamlakat qiymatini ham agronom yangilay oladi.
	require.NoError(t, repo.UpsertWaterDefault(ctx, &domain.WaterDefault{Source: 4, EC: 0.02, HCO3Mmol: 0.1, UpdatedAt: 3}))
	ro, _ := repo.FindWaterDefault(ctx, 4, nil)
	require.Equal(t, 0.02, ro.EC)

	list, err := repo.ListWaterDefaults(ctx)
	require.NoError(t, err)
	require.Len(t, list, 5, "4 national + 1 regional")
}

func TestParameters(t *testing.T) {
	ctx := context.Background()
	tx := beginTx(t)
	repo := repository.NewCatalogRepository(tx)

	p, err := repo.FindParameter(ctx, "residual_hco3_mmol")
	require.NoError(t, err)
	require.Equal(t, 0.0, p.Min)
	require.Equal(t, 3.0, p.Max)
	require.NotEmpty(t, p.Description["uz"])

	p.Value = 0.8
	require.NoError(t, repo.UpdateParameter(ctx, p))
	again, _ := repo.FindParameter(ctx, "residual_hco3_mmol")
	require.Equal(t, 0.8, again.Value)

	p.Value = 10
	require.Error(t, repo.UpdateParameter(ctx, p), "CHECK constraint guards the range too")
}
