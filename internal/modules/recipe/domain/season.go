package domain

import (
	"errors"
	"math"
)

var (
	ErrSeasonNotFound   = errors.New("season not found")
	ErrSeasonOpenExists = errors.New("plot already has an open season")
	ErrSeasonEnded      = errors.New("season is already ended")
	ErrInvalidSeason    = errors.New("invalid season values")
)

// CropRef — katalogdagi ekin.
type CropRef struct {
	ID   int32
	Code string
	Name map[string]string
}

// Season — bitta issiqxonadagi bitta ekin davri (seasons).
type Season struct {
	ID                int64
	FarmID            int64 // plots orqali
	PlotID            int64
	CropID            int32
	CropCode          string
	CropName          map[string]string
	Stage             Stage
	StartedAt         int64
	EndedAt           *int64
	YieldKg           *float64
	AreaM2            *float64
	FertilizerCostUZS *int64
	CreatedAt         int64
	UpdatedAt         int64
}

func (s *Season) Open() bool { return s.EndedAt == nil }

// SeasonEnd — mavsum yakuni (natija kartochkasi uchun).
type SeasonEnd struct {
	EndedAt           int64
	YieldKg           float64
	AreaM2            *float64 // nil = issiqxona maydoni
	FertilizerCostUZS *int64   // nil = retseptlar va narxlardan taxmin
}

// Validate — "xato kiritildi" chegaralari: 10 000 t hosil, 1000 ga maydon.
func (e SeasonEnd) Validate(startedAt int64) error {
	ok := func(v, max float64) bool { return !math.IsNaN(v) && v >= 0 && v <= max }
	if e.EndedAt < startedAt || !ok(e.YieldKg, 1e7) {
		return ErrInvalidSeason
	}
	if e.AreaM2 != nil && !(ok(*e.AreaM2, 1e7) && *e.AreaM2 > 0) {
		return ErrInvalidSeason
	}
	if e.FertilizerCostUZS != nil && *e.FertilizerCostUZS < 0 {
		return ErrInvalidSeason
	}
	return nil
}

type SeasonFilter struct {
	PlotID   *int64
	Open     *bool // nil = hammasi
	BeforeID int64
	Limit    int
}

// FertilizerUse — mavsumda tayyorlangan eritmalarga ketgan o'g'it.
type FertilizerUse struct {
	Code  string
	Grams float64
	ML    float64
}

// MeasurementSummary — mavsum davomidagi o'lchovlar.
type MeasurementSummary struct {
	Count       int
	AvgECIn     *float64
	AvgPHIn     *float64
	AvgDrainPct *float64
}
