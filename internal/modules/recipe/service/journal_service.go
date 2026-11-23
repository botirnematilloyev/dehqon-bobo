package service

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"dehqon-bobo/internal/modules/recipe/domain"
)

type BatchRepository interface {
	Create(ctx context.Context, b *domain.Batch) error
	List(ctx context.Context, farmID int64, f domain.BatchFilter) ([]domain.Batch, error)
	Get(ctx context.Context, farmID, id int64) (*domain.Batch, error)
	MarkMixed(ctx context.Context, farmID, id, at int64) (bool, error)
	DeletePlanned(ctx context.Context, farmID, id int64) (bool, error)
}

type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// OpenSeasons — issiqxonaning ochiq mavsumi (yo'q bo'lsa nil).
type OpenSeasons interface {
	OpenForPlot(ctx context.Context, plotID int64) (*domain.Season, error)
}

// JournalService — saqlangan retseptlar (solution_batches).
type JournalService struct {
	farms    Farms
	profiles Profiles
	repo     BatchRepository
	seasons  OpenSeasons
	tx       TxManager
	now      func() int64
}

func NewJournalService(farms Farms, profiles Profiles, repo BatchRepository, seasons OpenSeasons, tx TxManager) *JournalService {
	return &JournalService{farms: farms, profiles: profiles, repo: repo, seasons: seasons, tx: tx, now: func() int64 { return time.Now().Unix() }}
}

type SaveInput struct {
	PreviewInput
	PlotID *int64
}

// Save — preview bilan bir xil hisob, natija o'zgarmas snapshot sifatida
// saqlanadi. Hisob serverda QAYTA bajariladi: mijoz yuborgan grammlarga
// ishonilmaydi.
func (s *JournalService) Save(ctx context.Context, userID, farmID int64, in SaveInput) (*domain.Batch, error) {
	if !in.Stage.Valid() {
		return nil, domain.ErrInvalidStage
	}
	farm, err := s.farms.RecipeData(ctx, userID, farmID)
	if err != nil {
		return nil, err
	}
	if in.PlotID != nil && !hasPlot(farm, *in.PlotID) {
		return nil, domain.ErrPlotNotFound
	}
	profile, err := s.profiles.Profile(ctx, in.CropCode, in.Stage)
	if err != nil {
		return nil, err
	}
	p, err := Compute(*profile, *farm, in.PreviewInput)
	if err != nil {
		return nil, err
	}
	items, err := batchItems(p)
	if err != nil {
		return nil, err
	}
	snapshot, err := json.Marshal(NewSnapshot(p))
	if err != nil {
		return nil, err
	}

	// Issiqxonada shu ekinning ochiq mavsumi bo'lsa — retsept unga bog'lanadi
	// (natija kartochkasi xarajatni shundan hisoblaydi).
	var seasonID *int64
	if in.PlotID != nil {
		open, err := s.seasons.OpenForPlot(ctx, *in.PlotID)
		if err != nil {
			return nil, err
		}
		if open != nil && open.CropCode == profile.CropCode {
			seasonID = &open.ID
		}
	}

	now := s.now()
	b := &domain.Batch{
		SeasonID: seasonID,
		FarmID:   farmID, PlotID: in.PlotID, CreatedBy: &userID, ProfileID: profile.ID,
		WaterAnalysisID: &p.WaterID, Method: p.Method, VolumeL: p.VolumeL,
		StockTankL: p.StockTankL, InjectorRatio: p.InjectorRatio,
		// Kutilgan EC — suv + kislota + tuzlar (o'lchov shu bilan solishtiriladi).
		TargetEC:    math.Round(p.ECEstimate*100) / 100,
		CalcVersion: p.CalcVersion, Snapshot: snapshot, Status: domain.BatchPlanned,
		CreatedAt: now, UpdatedAt: now, Items: items,
		CropCode: profile.CropCode, Stage: profile.Stage,
	}
	if err := s.tx.WithinTx(ctx, func(ctx context.Context) error { return s.repo.Create(ctx, b) }); err != nil {
		return nil, err
	}
	return b, nil
}

// batchItems — to'g'ridan usulda hammasi bitta bakka (0); A/B da A, B va
// kislota idishi.
func batchItems(p *Preview) ([]domain.BatchItem, error) {
	tankOf := func(name string) int16 {
		if p.Method == domain.MethodDirect {
			return domain.TankSingle
		}
		switch name {
		case "A":
			return domain.TankA
		case "B":
			return domain.TankB
		}
		return domain.TankC
	}
	var out []domain.BatchItem
	pos := int16(0)
	for _, t := range p.Tanks {
		for _, it := range t.Items {
			if it.Grams < 0 {
				return nil, domain.ErrInfeasible
			}
			g := it.Grams
			out = append(out, domain.BatchItem{FertilizerCode: it.Code, Tank: tankOf(t.Tank), AmountG: &g, Position: pos})
			pos++
		}
	}
	for _, a := range p.Acids {
		ml := a.ML
		out = append(out, domain.BatchItem{FertilizerCode: a.Code, Tank: tankOf(a.Tank), AmountML: &ml, Position: pos})
		pos++
	}
	return out, nil
}

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

type ListInput struct {
	PlotID   *int64
	BeforeID int64
	Limit    int // 0 = standart
}

// List — yangisi birinchi. Ikkinchi qiymat: yana sahifa bormi.
func (s *JournalService) List(ctx context.Context, userID, farmID int64, in ListInput) ([]domain.Batch, bool, error) {
	if err := s.farms.CheckOwner(ctx, userID, farmID); err != nil {
		return nil, false, err
	}
	limit := in.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	limit = min(limit, maxListLimit)
	// Bitta ortiqcha o'qiladi: keyingi sahifa borligini bilish uchun.
	list, err := s.repo.List(ctx, farmID, domain.BatchFilter{PlotID: in.PlotID, BeforeID: in.BeforeID, Limit: limit + 1})
	if err != nil {
		return nil, false, err
	}
	if len(list) > limit {
		return list[:limit], true, nil
	}
	return list, false, nil
}

func (s *JournalService) Get(ctx context.Context, userID, farmID, id int64) (*domain.Batch, error) {
	if err := s.farms.CheckOwner(ctx, userID, farmID); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, farmID, id)
}

// MarkMixed — "tayyorladim". Qayta bosilsa o'zgarmaydi (idempotent).
func (s *JournalService) MarkMixed(ctx context.Context, userID, farmID, id int64) (*domain.Batch, error) {
	if err := s.farms.CheckOwner(ctx, userID, farmID); err != nil {
		return nil, err
	}
	if _, err := s.repo.MarkMixed(ctx, farmID, id, s.now()); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, farmID, id)
}

// Delete — faqat tayyorlanmagan retsept; tayyorlangani jurnalda qoladi.
func (s *JournalService) Delete(ctx context.Context, userID, farmID, id int64) error {
	if err := s.farms.CheckOwner(ctx, userID, farmID); err != nil {
		return err
	}
	b, err := s.repo.Get(ctx, farmID, id)
	if err != nil {
		return err
	}
	if b.Status != domain.BatchPlanned {
		return domain.ErrAlreadyMixed
	}
	ok, err := s.repo.DeletePlanned(ctx, farmID, id)
	if err != nil {
		return err
	}
	if !ok { // oraliqda tayyorlandi deb belgilangan
		return domain.ErrAlreadyMixed
	}
	return nil
}

func hasPlot(farm *domain.FarmData, id int64) bool {
	_, ok := farm.Plot(id)
	return ok
}
