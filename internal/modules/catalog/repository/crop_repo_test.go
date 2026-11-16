//go:build integration

package repository_test

import (
	"context"
	"testing"

	"dehqon-bobo/internal/modules/calc/hpg"
	"dehqon-bobo/internal/modules/catalog/domain"
	"dehqon-bobo/internal/modules/catalog/repository"
	"dehqon-bobo/internal/modules/catalog/service"

	"github.com/stretchr/testify/require"
)

// noTx — repo allaqachon test tranzaksiyasi ustida.
type noTx struct{}

func (noTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

func TestSeed_CropProfilesAreIonBalanced(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewCatalogRepository(testPool)

	crops, err := repo.ListCrops(ctx)
	require.NoError(t, err)
	codes := []string{}
	for _, c := range crops {
		codes = append(codes, c.Code)
		require.NotNil(t, c.SalinityECwThreshold, c.Code)
	}
	require.Equal(t, []string{"tomato", "cucumber", "pepper", "strawberry", "lettuce"}, codes)

	for _, code := range []string{"tomato", "cucumber"} {
		crop, err := repo.FindCrop(ctx, code)
		require.NoError(t, err)
		profiles, err := repo.ListCropProfiles(ctx, crop.ID)
		require.NoError(t, err)
		require.Len(t, profiles, 2, code)
		require.Equal(t, domain.StageVegetative, profiles[0].Stage)
		require.Equal(t, domain.StageFruiting, profiles[1].Stage)

		for _, p := range profiles {
			require.Equal(t, domain.ProfileDraft, p.Status, "seed agronom tasdiqlamagan")
			// Manbadagi SO4 HPG balansi bilan mos (ion ekvivalentligi).
			require.InDelta(t, hpg.SulfurFromBalance(p.Targets.Macro()), p.Targets.S, 0.05, code)
			// HPG EC taxmini manbadagi EC dan 0.6 mS/sm dan uzoq emas.
			require.InDelta(t, *p.ECTarget, hpg.EC(p.Targets.Macro()), 0.6, code)
		}
	}
}

func TestCropProfiles_CreateApproveArchive(t *testing.T) {
	ctx := context.Background()
	tx := beginTx(t)
	repo := repository.NewCatalogRepository(tx)
	svc := service.NewCatalogService(repo, noTx{})
	agronomist := createUser(t, tx)

	crop, err := repo.FindCrop(ctx, "tomato")
	require.NoError(t, err)
	seeded, err := repo.ListCropProfiles(ctx, crop.ID)
	require.NoError(t, err)
	seedFruiting := seeded[1]

	approved, err := svc.ApproveCropProfile(ctx, agronomist, seedFruiting.ID)
	require.NoError(t, err)
	require.Equal(t, domain.ProfileApproved, approved.Status)

	in := service.ProfileInput{
		Stage: domain.StageFruiting, Targets: seedFruiting.Targets, ECTarget: seedFruiting.ECTarget,
		PHMin: 5.6, PHMax: 6.2,
	}
	in.Targets.K = 400
	draft, err := svc.CreateCropProfile(ctx, "tomato", in)
	require.NoError(t, err)

	got, err := repo.FindCropProfile(ctx, draft.ID)
	require.NoError(t, err)
	require.Equal(t, 400.0, got.Targets.K)
	require.InDelta(t, draft.Targets.S, got.Targets.S, 1e-9)

	// Ikkinchisini tasdiqlash — birinchisi arxivga (unique index buzilmaydi).
	_, err = svc.ApproveCropProfile(ctx, agronomist, draft.ID)
	require.NoError(t, err)

	list, err := repo.ListCropProfiles(ctx, crop.ID)
	require.NoError(t, err)
	ids := map[int64]domain.ProfileStatus{}
	for _, p := range list {
		ids[p.ID] = p.Status
	}
	require.NotContains(t, ids, seedFruiting.ID, "arxivlangan profil ro'yxatda yo'q")
	require.Equal(t, domain.ProfileApproved, ids[draft.ID])

	_, err = svc.UpdateCropProfile(ctx, draft.ID, in)
	require.ErrorIs(t, err, domain.ErrProfileLocked)

	_, err = repo.FindCropProfile(ctx, 1<<40)
	require.ErrorIs(t, err, domain.ErrCropProfileNotFound)
}
