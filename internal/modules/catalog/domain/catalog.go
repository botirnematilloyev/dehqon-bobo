package domain

import (
	"errors"
	"math"
)

var (
	ErrFertilizerNotFound   = errors.New("fertilizer not found")
	ErrParameterNotFound    = errors.New("parameter not found")
	ErrWaterDefaultNotFound = errors.New("water default not found")

	ErrInvalidElements = errors.New("invalid elements: unknown element, percent outside (0, 100] or total over 100")
	ErrInvalidValue    = errors.New("invalid value")
	ErrOutOfRange      = errors.New("value outside the allowed range")
)

type FertilizerKind int16

const (
	KindMacro FertilizerKind = 1
	KindMicro FertilizerKind = 2
	KindAcid  FertilizerKind = 3
)

// Elements — fertilizer_elements.element CHECK bilan bir xil.
var Elements = map[string]bool{
	"N_NO3": true, "N_NH4": true, "P": true, "K": true, "Ca": true, "Mg": true, "S": true, "Cl": true,
	"Fe": true, "Mn": true, "B": true, "Zn": true, "Cu": true, "Mo": true, "Co": true, "Si": true,
}

type Fertilizer struct {
	ID               int32
	Code             string
	Name             map[string]string
	Formula          string
	Kind             FertilizerKind
	DefaultTank      int16
	SolubilityGL     *float64
	StockGL          *float64
	ConcentrationPct *float64
	DensityGML       *float64
	Elements         map[string]float64 // %
	Source           *string
	IsReviewed       bool
	IsActive         bool
	UpdatedBy        *int64
	UpdatedAt        *int64
}

// ValidateElements — element nomlari ma'lum, har biri (0, 100], jami ≤ 100.
func ValidateElements(e map[string]float64) error {
	total := 0.0
	for k, v := range e {
		if !Elements[k] || !(v > 0 && v <= 100) {
			return ErrInvalidElements
		}
		total += v
	}
	if total > 100+1e-9 {
		return ErrInvalidElements
	}
	return nil
}

// Validate — o'g'it turiga qarab majburiy maydonlar.
func (f *Fertilizer) Validate() error {
	if err := ValidateElements(f.Elements); err != nil {
		return err
	}
	if f.StockGL != nil && !(*f.StockGL > 0 && *f.StockGL <= 3000) {
		return ErrInvalidValue
	}
	if f.SolubilityGL != nil && !(*f.SolubilityGL > 0 && *f.SolubilityGL <= 5000) {
		return ErrInvalidValue
	}
	if f.Kind == KindAcid {
		if f.ConcentrationPct == nil || f.DensityGML == nil {
			return ErrInvalidValue
		}
		if err := ValidateAcid(*f.ConcentrationPct, *f.DensityGML); err != nil {
			return err
		}
	}
	return nil
}

func ValidateAcid(concentrationPct, densityGML float64) error {
	if !(concentrationPct > 0 && concentrationPct <= 100) || !(densityGML >= 0.9 && densityGML <= 2.0) {
		return ErrInvalidValue
	}
	return nil
}

// WaterDefault — suv manbai uchun taxminiy tarkib (ion mg/L, HCO3 mmol/L).
type WaterDefault struct {
	ID         int32
	Source     int16 // 1 quduq, 2 kanal, 3 vodoprovod, 4 filtr
	RegionID   *int32
	EC         float64
	PH         *float64
	Ca         float64
	Mg         float64
	Na         float64
	K          float64
	Cl         float64
	SO4        float64
	NO3        float64
	HCO3Mmol   float64
	Note       *string
	IsReviewed bool
	UpdatedBy  *int64
	UpdatedAt  int64
}

func (w *WaterDefault) Validate() error {
	if w.Source < 1 || w.Source > 4 {
		return ErrInvalidValue
	}
	in := func(v, max float64) bool { return v >= 0 && v <= max && !math.IsNaN(v) }
	if !in(w.EC, 20) || !in(w.HCO3Mmol, 30) {
		return ErrInvalidValue
	}
	for _, v := range []float64{w.Ca, w.Mg, w.Na, w.K, w.Cl, w.SO4, w.NO3} {
		if !in(v, 2000) {
			return ErrInvalidValue
		}
	}
	if w.PH != nil && !in(*w.PH, 14) {
		return ErrInvalidValue
	}
	return nil
}

// Parameter — hisob parametri: standart qiymat va ruxsat etilgan oraliq.
type Parameter struct {
	Key             string
	Value           float64
	Min             float64
	Max             float64
	Unit            *string
	Description     map[string]string
	Source          *string
	FarmOverridable bool
	IsReviewed      bool
	UpdatedBy       *int64
	UpdatedAt       int64
}

func (p *Parameter) Check(v float64) error {
	if math.IsNaN(v) || v < p.Min || v > p.Max {
		return ErrOutOfRange
	}
	return nil
}
