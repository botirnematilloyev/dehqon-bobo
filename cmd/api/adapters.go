package main

import (
	"context"
	"errors"

	"dehqon-bobo/internal/modules/catalog"
	catalogdomain "dehqon-bobo/internal/modules/catalog/domain"
	farmdomain "dehqon-bobo/internal/modules/farm/domain"
)

// Modullar bir-birini import qilmaydi: har biri o'z portini e'lon qiladi,
// ularni shu yerda bog'laymiz (binary'dagi cmd/api/adapters.go kabi).

// onboarderFunc — user.Module.MarkOnboarded ni farm/service.UserOnboarder
// portiga moslaydi.
type onboarderFunc func(ctx context.Context, userID int64, regionID, districtID *int32, at int64) error

func (f onboarderFunc) MarkOnboarded(ctx context.Context, userID int64, regionID, districtID *int32, at int64) error {
	return f(ctx, userID, regionID, districtID, at)
}

// catalogBridge — catalog.Module ni farm/service.Catalog va
// farm/service.WaterDefaults portlariga moslaydi.
type catalogBridge struct{ m *catalog.Module }

func (b catalogBridge) Fertilizers(ctx context.Context) ([]farmdomain.FertilizerDef, error) {
	list, err := b.m.Fertilizers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]farmdomain.FertilizerDef, 0, len(list))
	for _, f := range list {
		out = append(out, farmdomain.FertilizerDef{
			Code: f.Code, Name: f.Name, Formula: f.Formula, Kind: int16(f.Kind), Tank: f.DefaultTank, Elements: f.Elements,
			SolubilityGL: f.SolubilityGL, StockGL: f.StockGL, ConcentrationPct: f.ConcentrationPct, DensityGML: f.DensityGML,
		})
	}
	return out, nil
}

func (b catalogBridge) Parameters(ctx context.Context) ([]farmdomain.ParamDef, error) {
	list, err := b.m.Parameters(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]farmdomain.ParamDef, 0, len(list))
	for _, p := range list {
		out = append(out, farmdomain.ParamDef{
			Key: p.Key, Value: p.Value, Min: p.Min, Max: p.Max, Unit: p.Unit,
			Description: p.Description, FarmOverridable: p.FarmOverridable,
		})
	}
	return out, nil
}

func (b catalogBridge) Estimate(ctx context.Context, source farmdomain.WaterSource, regionID *int32) (farmdomain.WaterAnalysis, error) {
	w, err := b.m.WaterDefault(ctx, int16(source), regionID)
	if errors.Is(err, catalogdomain.ErrWaterDefaultNotFound) {
		return farmdomain.WaterAnalysis{}, farmdomain.ErrWaterDefaultMissing
	}
	if err != nil {
		return farmdomain.WaterAnalysis{}, err
	}
	opt := func(v float64) *float64 {
		if v == 0 {
			return nil
		}
		return &v
	}
	return farmdomain.WaterAnalysis{
		EC: w.EC, PH: w.PH, Ca: w.Ca, Mg: w.Mg, HCO3Mmol: w.HCO3Mmol,
		Na: opt(w.Na), K: opt(w.K), Cl: opt(w.Cl), SO4: opt(w.SO4), NO3: opt(w.NO3),
	}, nil
}
