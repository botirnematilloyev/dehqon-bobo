package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"dehqon-bobo/internal/modules/user/domain"

	"github.com/stretchr/testify/require"
)

type fakeLocationService struct {
	regions   []domain.Region
	districts []domain.District

	gotRegionIDs []*int32
	err          error
}

func (f *fakeLocationService) ListRegions(ctx context.Context) ([]domain.Region, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.regions, nil
}

func (f *fakeLocationService) ListDistricts(ctx context.Context, regionID *int32) ([]domain.District, error) {
	f.gotRegionIDs = append(f.gotRegionIDs, regionID)
	if f.err != nil {
		return nil, f.err
	}
	return f.districts, nil
}

func TestLocationHandler_Regions(t *testing.T) {
	t.Run("returns the multilingual names", func(t *testing.T) {
		svc := &fakeLocationService{regions: []domain.Region{
			{ID: 1, Name: map[string]string{"uz": "Toshkent", "ru": "Ташкент"}},
		}}
		rec := httptest.NewRecorder()

		NewLocationHandler(svc).Regions(rec, httptest.NewRequest(http.MethodGet, "/regions", nil))

		require.Equal(t, http.StatusOK, rec.Code)

		var body RegionsResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Len(t, body.Regions, 1)
		require.Equal(t, "Toshkent", body.Regions[0].Name["uz"])
		require.Equal(t, "Ташкент", body.Regions[0].Name["ru"])
	})

	t.Run("sets a cache header", func(t *testing.T) {
		svc := &fakeLocationService{}
		rec := httptest.NewRecorder()

		NewLocationHandler(svc).Regions(rec, httptest.NewRequest(http.MethodGet, "/regions", nil))

		require.Equal(t, "public, max-age=3600", rec.Header().Get("Cache-Control"))
	})

	t.Run("returns an empty array, not null", func(t *testing.T) {
		svc := &fakeLocationService{}
		rec := httptest.NewRecorder()

		NewLocationHandler(svc).Regions(rec, httptest.NewRequest(http.MethodGet, "/regions", nil))

		require.Contains(t, rec.Body.String(), `"regions":[]`)
	})

	t.Run("does not leak internal errors", func(t *testing.T) {
		svc := &fakeLocationService{err: errors.New(`pq: relation "regions" does not exist`)}
		rec := httptest.NewRecorder()

		NewLocationHandler(svc).Regions(rec, httptest.NewRequest(http.MethodGet, "/regions", nil))

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		require.Equal(t, "internal server error", decodeError(t, rec))
	})
}

func TestLocationHandler_Districts(t *testing.T) {
	get := func(t *testing.T, query string) *http.Request {
		t.Helper()
		return httptest.NewRequest(http.MethodGet, "/districts"+query, nil)
	}

	t.Run("passes nil when region_id is absent", func(t *testing.T) {
		svc := &fakeLocationService{}
		rec := httptest.NewRecorder()

		NewLocationHandler(svc).Districts(rec, get(t, ""))

		require.Equal(t, http.StatusOK, rec.Code)
		require.Len(t, svc.gotRegionIDs, 1)
		require.Nil(t, svc.gotRegionIDs[0], "no filter means every district")
	})

	t.Run("passes the parsed region_id", func(t *testing.T) {
		svc := &fakeLocationService{}
		rec := httptest.NewRecorder()

		NewLocationHandler(svc).Districts(rec, get(t, "?region_id=7"))

		require.Equal(t, http.StatusOK, rec.Code)
		require.NotNil(t, svc.gotRegionIDs[0])
		require.Equal(t, int32(7), *svc.gotRegionIDs[0])
	})

	t.Run("rejects a malformed region_id", func(t *testing.T) {
		for _, q := range []string{"?region_id=abc", "?region_id=0", "?region_id=-1", "?region_id=99999999999"} {
			t.Run(q, func(t *testing.T) {
				svc := &fakeLocationService{}
				rec := httptest.NewRecorder()

				NewLocationHandler(svc).Districts(rec, get(t, q))

				require.Equal(t, http.StatusBadRequest, rec.Code)
				require.Empty(t, svc.gotRegionIDs, "service must not be reached")
			})
		}
	})

	t.Run("returns districts with their region", func(t *testing.T) {
		svc := &fakeLocationService{districts: []domain.District{
			{ID: 5, RegionID: 7, Name: map[string]string{"uz": "Chilonzor"}},
		}}
		rec := httptest.NewRecorder()

		NewLocationHandler(svc).Districts(rec, get(t, "?region_id=7"))

		var body DistrictsResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Len(t, body.Districts, 1)
		require.Equal(t, int32(5), body.Districts[0].ID)
		require.Equal(t, int32(7), body.Districts[0].RegionID)
		require.Equal(t, "Chilonzor", body.Districts[0].Name["uz"])
	})
}
