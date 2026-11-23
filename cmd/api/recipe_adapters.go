package main

import (
	"context"
	"errors"

	"dehqon-bobo/internal/modules/calc/inputs"
	"dehqon-bobo/internal/modules/calc/water"
	"dehqon-bobo/internal/modules/catalog"
	catalogdomain "dehqon-bobo/internal/modules/catalog/domain"
	"dehqon-bobo/internal/modules/farm"
	farmdomain "dehqon-bobo/internal/modules/farm/domain"
	recipedomain "dehqon-bobo/internal/modules/recipe/domain"
)

// recipeFarms — farm.Module ni recipe/service.Farms portiga moslaydi.
type recipeFarms struct{ m *farm.Module }

func (b recipeFarms) RecipeData(ctx context.Context, userID, farmID int64) (*recipedomain.FarmData, error) {
	in, err := b.m.RecipeInputs(ctx, userID, farmID)
	if errors.Is(err, farmdomain.ErrFarmNotFound) {
		return nil, recipedomain.ErrFarmNotFound
	}
	if err != nil {
		return nil, err
	}
	return toRecipeFarm(in), nil
}

func (b recipeFarms) CheckOwner(ctx context.Context, userID, farmID int64) error {
	err := b.m.CheckOwner(ctx, userID, farmID)
	if errors.Is(err, farmdomain.ErrFarmNotFound) {
		return recipedomain.ErrFarmNotFound
	}
	return err
}

func toRecipeFarm(in *farm.RecipeInputs) *recipedomain.FarmData {
	out := &recipedomain.FarmData{
		FarmID: in.Farm.ID, Method: recipedomain.Method(in.Farm.MixingMethod),
		InjectorRatio: in.Farm.InjectorRatio, StockTankL: in.Farm.StockTankL,
		Fertilizers: make([]recipedomain.Fertilizer, 0, len(in.Fertilizers)),
		Params:      make(map[string]float64, len(in.Params)),
		Plots:       make([]recipedomain.Plot, 0, len(in.Plots)),
	}
	for _, p := range in.Plots {
		out.Plots = append(out.Plots, recipedomain.Plot{ID: p.ID, AreaM2: p.AreaM2})
	}
	if w := in.Water; w != nil {
		val := func(p *float64) float64 {
			if p == nil {
				return 0
			}
			return *p
		}
		out.Water = &recipedomain.Water{
			ID: w.ID, IsEstimate: w.IsEstimate, PH: w.PH,
			Analysis: water.Analysis{
				EC: w.EC, Ca: w.Ca, Mg: w.Mg, K: val(w.K), Na: val(w.Na), Cl: val(w.Cl),
				SO4: val(w.SO4), NO3: val(w.NO3), HCO3Mmol: w.HCO3Mmol,
			},
		}
	}
	for _, f := range in.Fertilizers {
		out.Fertilizers = append(out.Fertilizers, recipedomain.Fertilizer{
			Fertilizer: inputs.Fertilizer{
				Code: f.Code, Kind: f.Kind, Formula: f.Formula, Elements: f.Elements,
				ConcentrationPct: f.ConcentrationPct, DensityGML: f.DensityGML,
			},
			Name: f.Name, Tank: f.Tank, SolubilityGL: f.SolubilityGL, PriceUZSPerKg: f.PriceUZSPerKg,
		})
	}
	for _, p := range in.Params {
		out.Params[p.Key] = p.Value
	}
	return out
}

// recipeProfiles — catalog.Module ni recipe/service.Profiles portiga moslaydi.
type recipeProfiles struct{ m *catalog.Module }

func (b recipeProfiles) Profile(ctx context.Context, cropCode string, stage recipedomain.Stage) (*recipedomain.Profile, error) {
	crop, p, err := b.m.CropProfile(ctx, cropCode, catalogdomain.Stage(stage))
	switch {
	case errors.Is(err, catalogdomain.ErrCropNotFound):
		return nil, recipedomain.ErrCropNotFound
	case errors.Is(err, catalogdomain.ErrCropProfileNotFound):
		return nil, recipedomain.ErrProfileNotFound
	case err != nil:
		return nil, err
	}
	t := p.Targets
	return &recipedomain.Profile{
		ID: p.ID, CropCode: crop.Code, CropName: crop.Name, SalinityECw: crop.SalinityECwThreshold,
		Stage: stage, Reviewed: p.Status == catalogdomain.ProfileApproved, Title: p.Title, Source: p.Source,
		Targets: recipedomain.Targets{
			NH4N: t.NH4N, NO3N: t.NO3N, P: t.P, K: t.K, Ca: t.Ca, Mg: t.Mg, S: t.S, Cl: t.Cl,
			Fe: t.Fe, Mn: t.Mn, B: t.B, Zn: t.Zn, Cu: t.Cu, Mo: t.Mo,
		},
		ECTarget: p.ECTarget, PHMin: p.PHMin, PHMax: p.PHMax,
	}, nil
}

func (b recipeProfiles) Crop(ctx context.Context, code string) (*recipedomain.CropRef, error) {
	c, err := b.m.Crop(ctx, code)
	if errors.Is(err, catalogdomain.ErrCropNotFound) {
		return nil, recipedomain.ErrCropNotFound
	}
	if err != nil {
		return nil, err
	}
	return &recipedomain.CropRef{ID: c.ID, Code: c.Code, Name: c.Name}, nil
}
