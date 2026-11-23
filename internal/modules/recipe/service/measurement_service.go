package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"dehqon-bobo/internal/modules/recipe/domain"
)

type MeasurementRepository interface {
	Create(ctx context.Context, m *domain.Measurement) error
	List(ctx context.Context, farmID int64, f domain.MeasurementFilter) ([]domain.Measurement, error)
}

// MeasurementService — o'lchovlar va tuzatish (korrektor).
type MeasurementService struct {
	farms   Farms
	batches BatchRepository
	repo    MeasurementRepository
	now     func() int64
}

func NewMeasurementService(farms Farms, batches BatchRepository, repo MeasurementRepository) *MeasurementService {
	return &MeasurementService{farms: farms, batches: batches, repo: repo, now: func() int64 { return time.Now().Unix() }}
}

type RecordInput struct {
	domain.Measurement
	// TankVolumeL — bakdagi hozirgi hajm (EC tuzatishi uchun); nil = retsept hajmi.
	TankVolumeL *float64
}

// clockSkew — telefon soati biroz oldinda bo'lishi mumkin.
const clockSkew = 300

func decodeSnapshot(b *domain.Batch) (*Snapshot, error) {
	var s Snapshot
	if err := json.Unmarshal(b.Snapshot, &s); err != nil {
		return nil, fmt.Errorf("decode batch %d snapshot: %w", b.ID, err)
	}
	return &s, nil
}

// Record — o'lchovni saqlaydi va baholaydi (retsept bo'lsa — kutilgan EC va
// pH bilan solishtiradi, chetga chiqsa tuzatish taklif qiladi).
func (s *MeasurementService) Record(ctx context.Context, userID, farmID int64, in RecordInput) (*domain.Measurement, *Assessment, error) {
	m := in.Measurement
	now := s.now()
	if m.MeasuredAt == 0 {
		m.MeasuredAt = now
	}
	if m.MeasuredAt > now+clockSkew || m.MeasuredAt < 0 {
		return nil, nil, domain.ErrInvalidMeasurement
	}
	if in.TankVolumeL != nil && !(*in.TankVolumeL > 0 && *in.TankVolumeL <= maxVolumeL) {
		return nil, nil, domain.ErrInvalidMeasurement
	}
	if err := m.Validate(); err != nil {
		return nil, nil, err
	}

	farm, err := s.farms.RecipeData(ctx, userID, farmID)
	if err != nil {
		return nil, nil, err
	}
	var (
		batch *domain.Batch
		snap  *Snapshot
	)
	if m.BatchID != nil {
		if batch, err = s.batches.Get(ctx, farmID, *m.BatchID); err != nil {
			return nil, nil, err
		}
		if snap, err = decodeSnapshot(batch); err != nil {
			return nil, nil, err
		}
		if m.PlotID == nil {
			m.PlotID = batch.PlotID
		}
	}
	if m.PlotID != nil && !hasPlot(farm, *m.PlotID) {
		return nil, nil, domain.ErrPlotNotFound
	}

	assessment, err := Assess(m, batch, snap, farm.Params, in.TankVolumeL)
	if err != nil {
		return nil, nil, err
	}
	m.FarmID, m.MeasuredBy, m.CreatedAt = farmID, &userID, now
	if err := s.repo.Create(ctx, &m); err != nil {
		return nil, nil, err
	}
	return &m, assessment, nil
}

type MeasurementListInput struct {
	PlotID   *int64
	BatchID  *int64
	BeforeID int64
	Limit    int
}

func (s *MeasurementService) List(ctx context.Context, userID, farmID int64, in MeasurementListInput) ([]domain.Measurement, bool, error) {
	if err := s.farms.CheckOwner(ctx, userID, farmID); err != nil {
		return nil, false, err
	}
	limit := in.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	limit = min(limit, maxListLimit)
	list, err := s.repo.List(ctx, farmID, domain.MeasurementFilter{
		PlotID: in.PlotID, BatchID: in.BatchID, BeforeID: in.BeforeID, Limit: limit + 1,
	})
	if err != nil {
		return nil, false, err
	}
	if len(list) > limit {
		return list[:limit], true, nil
	}
	return list, false, nil
}

// Correct — aylanma bak uchun tuzatuvchi eritma (HPG korrektori). Hech
// narsa saqlamaydi; dehqon natijani yangi retsept sifatida saqlashi mumkin.
func (s *MeasurementService) Correct(ctx context.Context, userID, farmID, batchID int64, in CorrectionInput) (*CorrectionResult, error) {
	farm, err := s.farms.RecipeData(ctx, userID, farmID)
	if err != nil {
		return nil, err
	}
	batch, err := s.batches.Get(ctx, farmID, batchID)
	if err != nil {
		return nil, err
	}
	snap, err := decodeSnapshot(batch)
	if err != nil {
		return nil, err
	}
	return Correct(batch, snap, *farm, in)
}
