//go:build integration

package repository_test

import (
	"context"
	"testing"

	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/modules/user/repository"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func clearLocations(t *testing.T, tx pgx.Tx) {
	t.Helper()

	_, err := tx.Exec(context.Background(), `DELETE FROM districts`)
	require.NoError(t, err)
	_, err = tx.Exec(context.Background(), `DELETE FROM regions`)
	require.NoError(t, err)
}

func insertNamedRegion(t *testing.T, tx pgx.Tx, uzName string) int32 {
	t.Helper()

	var id int32
	err := tx.QueryRow(context.Background(),
		`INSERT INTO regions (name) VALUES ($1) RETURNING id`,
		`{"uz":"`+uzName+`","ru":"`+uzName+`-ru"}`,
	).Scan(&id)
	require.NoError(t, err)

	return id
}

func insertNamedDistrict(t *testing.T, tx pgx.Tx, regionID int32, uzName string) int32 {
	t.Helper()

	var id int32
	err := tx.QueryRow(context.Background(),
		`INSERT INTO districts (name, region_id) VALUES ($1, $2) RETURNING id`,
		`{"uz":"`+uzName+`"}`, regionID,
	).Scan(&id)
	require.NoError(t, err)

	return id
}

func TestRegionRepository_List(t *testing.T) {
	ctx := context.Background()

	t.Run("returns regions sorted by uzbek name", func(t *testing.T) {
		tx := beginTx(t)
		clearLocations(t, tx)
		repo := repository.NewRegionRepository(tx)

		insertNamedRegion(t, tx, "Samarqand")
		insertNamedRegion(t, tx, "Andijon")
		insertNamedRegion(t, tx, "Buxoro")

		got, err := repo.List(ctx)
		require.NoError(t, err)
		require.Len(t, got, 3)

		require.Equal(t, "Andijon", got[0].Name["uz"])
		require.Equal(t, "Buxoro", got[1].Name["uz"])
		require.Equal(t, "Samarqand", got[2].Name["uz"])
	})

	t.Run("round-trips every language", func(t *testing.T) {
		tx := beginTx(t)
		clearLocations(t, tx)
		repo := repository.NewRegionRepository(tx)

		insertNamedRegion(t, tx, "Toshkent")

		got, err := repo.List(ctx)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, map[string]string{
			"uz": "Toshkent",
			"ru": "Toshkent-ru",
		}, got[0].Name)
	})

	t.Run("returns an empty slice, not nil", func(t *testing.T) {
		tx := beginTx(t)
		clearLocations(t, tx)
		repo := repository.NewRegionRepository(tx)

		got, err := repo.List(ctx)
		require.NoError(t, err)

		require.NotNil(t, got)
		require.Empty(t, got)
	})
}

func TestDistrictRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("lists every district", func(t *testing.T) {
		tx := beginTx(t)
		clearLocations(t, tx)
		repo := repository.NewDistrictRepository(tx)

		r1 := insertNamedRegion(t, tx, "Toshkent")
		r2 := insertNamedRegion(t, tx, "Samarqand")
		insertNamedDistrict(t, tx, r1, "Chilonzor")
		insertNamedDistrict(t, tx, r2, "Urgut")

		got, err := repo.List(ctx)
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Equal(t, "Chilonzor", got[0].Name["uz"], "sorted by uzbek name")
	})

	t.Run("filters by region", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewDistrictRepository(tx)

		r1 := insertNamedRegion(t, tx, "Toshkent")
		r2 := insertNamedRegion(t, tx, "Samarqand")
		insertNamedDistrict(t, tx, r1, "Chilonzor")
		insertNamedDistrict(t, tx, r1, "Yunusobod")
		insertNamedDistrict(t, tx, r2, "Urgut")

		got, err := repo.ListByRegion(ctx, r1)
		require.NoError(t, err)
		require.Len(t, got, 2)
		for _, d := range got {
			require.Equal(t, r1, d.RegionID)
		}
	})

	t.Run("returns an empty slice for a region without districts", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewDistrictRepository(tx)

		got, err := repo.ListByRegion(ctx, 999999)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Empty(t, got)
	})

	t.Run("finds one by id", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewDistrictRepository(tx)

		regionID := insertNamedRegion(t, tx, "Toshkent")
		districtID := insertNamedDistrict(t, tx, regionID, "Chilonzor")

		got, err := repo.FindByID(ctx, districtID)
		require.NoError(t, err)
		require.Equal(t, districtID, got.ID)
		require.Equal(t, regionID, got.RegionID, "needed for the region integrity check")
		require.Equal(t, "Chilonzor", got.Name["uz"])
	})

	t.Run("returns ErrDistrictNotFound", func(t *testing.T) {
		tx := beginTx(t)
		repo := repository.NewDistrictRepository(tx)

		got, err := repo.FindByID(ctx, 999999)
		require.Nil(t, got)
		require.ErrorIs(t, err, domain.ErrDistrictNotFound)
	})
}
