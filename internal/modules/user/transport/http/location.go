package http

import (
	"context"
	"dehqon-bobo/internal/platform/i18n"
	"log/slog"
	"net/http"
	"strconv"

	"dehqon-bobo/internal/modules/user/domain"
)

type LocationService interface {
	ListRegions(ctx context.Context) ([]domain.Region, error)
	ListDistricts(ctx context.Context, regionID *int32) ([]domain.District, error)
}

type LocationHandler struct {
	locations LocationService
}

func NewLocationHandler(locations LocationService) *LocationHandler {
	return &LocationHandler{locations: locations}
}

const locationsCacheControl = "public, max-age=3600"

func (h *LocationHandler) Regions(w http.ResponseWriter, r *http.Request) {
	regions, err := h.locations.ListRegions(r.Context())
	if err != nil {
		slog.Error("list regions failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, i18n.ErrInternal)
		return
	}

	out := make([]RegionResponse, 0, len(regions))
	for _, reg := range regions {
		out = append(out, RegionResponse{ID: reg.ID, Name: reg.Name})
	}

	w.Header().Set("Cache-Control", locationsCacheControl)
	writeJSON(w, http.StatusOK, RegionsResponse{Regions: out})
}

func (h *LocationHandler) Districts(w http.ResponseWriter, r *http.Request) {
	regionID, err := parseOptionalInt32(r.URL.Query().Get("region_id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, msgRegionIDInvalid)
		return
	}

	districts, err := h.locations.ListDistricts(r.Context(), regionID)
	if err != nil {
		slog.Error("list districts failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, i18n.ErrInternal)
		return
	}

	out := make([]DistrictResponse, 0, len(districts))
	for _, d := range districts {
		out = append(out, DistrictResponse{ID: d.ID, RegionID: d.RegionID, Name: d.Name})
	}

	w.Header().Set("Cache-Control", locationsCacheControl)
	writeJSON(w, http.StatusOK, DistrictsResponse{Districts: out})
}

func parseOptionalInt32(raw string) (*int32, error) {
	if raw == "" {
		return nil, nil
	}

	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n <= 0 {
		return nil, errInvalidQueryParam
	}

	v := int32(n)
	return &v, nil
}
