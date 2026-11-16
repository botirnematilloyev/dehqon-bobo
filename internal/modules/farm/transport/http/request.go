package http

import (
	"errors"
	"net/http"
	"strconv"

	"dehqon-bobo/internal/modules/farm/domain"
	"dehqon-bobo/internal/modules/farm/service"

	"github.com/go-chi/chi/v5"
)

// API'da enum'lar matn bilan: mijoz raqamlarni yodlab yurmasin, DB'dagi
// SMALLINT qiymatlar esa ichki tafsilot bo'lib qoladi.
var (
	systemTypes = map[string]domain.SystemType{
		"greenhouse_soil": domain.SystemGreenhouseSoil,
		"substrate":       domain.SystemSubstrate,
		"hydroponics":     domain.SystemHydroponics,
		"open_field":      domain.SystemOpenField,
	}
	mixingMethods = map[string]domain.MixingMethod{
		"direct": domain.MixingDirect,
		"ab":     domain.MixingAB,
	}
	waterSources = map[string]domain.WaterSource{
		"well":   domain.WaterWell,
		"canal":  domain.WaterCanal,
		"tap":    domain.WaterTap,
		"filter": domain.WaterFilter,
	}
)

func parseEnum[T any](m map[string]T, raw *string, invalid error) (*T, error) {
	if raw == nil {
		return nil, nil
	}
	v, ok := m[*raw]
	if !ok {
		return nil, invalid
	}
	return &v, nil
}

func nameOf[T comparable](m map[string]T, v T) string {
	for k, val := range m {
		if val == v {
			return k
		}
	}
	return ""
}

type OnboardRequest struct {
	RegionID    int32   `json:"region_id"`
	DistrictID  *int32  `json:"district_id"`
	WaterSource string  `json:"water_source"`
	SystemType  *string `json:"system_type"`
}

func (r OnboardRequest) toInput() (service.OnboardInput, error) {
	if r.RegionID <= 0 {
		return service.OnboardInput{}, domain.ErrInvalidLocation
	}
	src, err := parseEnum(waterSources, &r.WaterSource, domain.ErrInvalidWaterSource)
	if err != nil {
		return service.OnboardInput{}, err
	}
	sys, err := parseEnum(systemTypes, r.SystemType, domain.ErrInvalidSystemType)
	if err != nil {
		return service.OnboardInput{}, err
	}
	return service.OnboardInput{RegionID: r.RegionID, DistrictID: r.DistrictID, WaterSource: *src, SystemType: sys}, nil
}

type FarmRequest struct {
	Name          *string  `json:"name"`
	RegionID      *int32   `json:"region_id"`
	DistrictID    *int32   `json:"district_id"`
	SystemType    *string  `json:"system_type"`
	MixingMethod  *string  `json:"mixing_method"`
	InjectorRatio *int32   `json:"injector_ratio"`
	StockTankL    *float64 `json:"stock_tank_l"`
}

func (r FarmRequest) toInput() (service.FarmInput, error) {
	sys, err := parseEnum(systemTypes, r.SystemType, domain.ErrInvalidSystemType)
	if err != nil {
		return service.FarmInput{}, err
	}
	mix, err := parseEnum(mixingMethods, r.MixingMethod, domain.ErrInvalidMixing)
	if err != nil {
		return service.FarmInput{}, err
	}
	if (r.RegionID != nil && *r.RegionID <= 0) || (r.DistrictID != nil && *r.DistrictID <= 0) {
		return service.FarmInput{}, domain.ErrInvalidLocation
	}
	return service.FarmInput{
		Name:          r.Name,
		RegionID:      r.RegionID,
		DistrictID:    r.DistrictID,
		SystemType:    sys,
		MixingMethod:  mix,
		InjectorRatio: r.InjectorRatio,
		StockTankL:    r.StockTankL,
	}, nil
}

type PlotRequest struct {
	Name           *string  `json:"name"`
	AreaM2         *float64 `json:"area_m2"`
	DefaultVolumeL *float64 `json:"default_volume_l"`
	IsArchived     *bool    `json:"is_archived"`
}

func (r PlotRequest) toInput() service.PlotInput {
	return service.PlotInput{Name: r.Name, AreaM2: r.AreaM2, DefaultVolumeL: r.DefaultVolumeL, IsArchived: r.IsArchived}
}

type WaterRequest struct {
	Source    string   `json:"source"`
	Estimate  bool     `json:"estimate"`
	EC        *float64 `json:"ec"`
	PH        *float64 `json:"ph"`
	Ca        *float64 `json:"ca"`
	Mg        *float64 `json:"mg"`
	Na        *float64 `json:"na"`
	K         *float64 `json:"k"`
	Cl        *float64 `json:"cl"`
	SO4       *float64 `json:"so4"`
	NO3       *float64 `json:"no3"`
	HCO3Mmol  *float64 `json:"hco3_mmol"`
	LabName   *string  `json:"lab_name"`
	SampledAt *int64   `json:"sampled_at"`
}

func (r WaterRequest) toInput() (service.WaterInput, error) {
	src, err := parseEnum(waterSources, &r.Source, domain.ErrInvalidWaterSource)
	if err != nil {
		return service.WaterInput{}, err
	}
	return service.WaterInput{
		Source: *src, Estimate: r.Estimate,
		EC: r.EC, PH: r.PH, Ca: r.Ca, Mg: r.Mg, Na: r.Na, K: r.K, Cl: r.Cl, SO4: r.SO4, NO3: r.NO3,
		HCO3Mmol: r.HCO3Mmol, LabName: r.LabName, SampledAt: r.SampledAt,
	}, nil
}

var errInvalidID = errors.New("invalid id")

func pathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil || id <= 0 {
		return 0, errInvalidID
	}
	return id, nil
}
