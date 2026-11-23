package domain

import (
	"errors"
	"math"
)

var (
	ErrInvalidMeasurement = errors.New("invalid measurement: need ec_in, ph_in or ec_drain within range")
	ErrInvalidCorrection  = errors.New("invalid correction input: ec_now > 0, 0 < volume_now_l < volume_final_l")
)

// Measurement — EC / pH / drenaj o'lchovi (measurements jadvali).
type Measurement struct {
	ID         int64
	FarmID     int64
	PlotID     *int64
	BatchID    *int64
	MeasuredBy *int64
	ECIn       *float64 // mS/sm
	PHIn       *float64
	ECDrain    *float64
	PHDrain    *float64
	DrainPct   *float64
	TempC      *float64
	Note       *string
	MeasuredAt int64
	CreatedAt  int64
}

const maxNoteLen = 1000

// Validate — jadval CHECK'lari va "xato kiritildi" chegaralari: EC ≤ 20
// (dengiz suvining yarmi), harorat −5…60 °C.
func (m *Measurement) Validate() error {
	in := func(p *float64, lo, hi float64) bool {
		return p == nil || (!math.IsNaN(*p) && *p >= lo && *p <= hi)
	}
	if m.ECIn == nil && m.PHIn == nil && m.ECDrain == nil {
		return ErrInvalidMeasurement
	}
	if !in(m.ECIn, 0, 20) || !in(m.ECDrain, 0, 20) || !in(m.PHIn, 0, 14) || !in(m.PHDrain, 0, 14) ||
		!in(m.DrainPct, 0, 100) || !in(m.TempC, -5, 60) {
		return ErrInvalidMeasurement
	}
	if m.Note != nil && len([]rune(*m.Note)) > maxNoteLen {
		return ErrInvalidMeasurement
	}
	return nil
}

type MeasurementFilter struct {
	PlotID   *int64
	BatchID  *int64
	BeforeID int64
	Limit    int
}
