package service

import (
	"context"
	"strings"
	"time"

	"dehqon-bobo/internal/modules/farm/domain"
)

// PlotService — "Issiqxonalarim". Har bir amal avval xo'jalik egasini
// tekshiradi (FarmService.owned orqali).
type PlotService struct {
	farms *FarmService
	plots PlotRepository
}

func NewPlotService(farms *FarmService, plots PlotRepository) *PlotService {
	return &PlotService{farms: farms, plots: plots}
}

type PlotInput struct {
	Name           *string
	AreaM2         *float64
	DefaultVolumeL *float64
	IsArchived     *bool
}

func (s *PlotService) List(ctx context.Context, userID, farmID int64, includeArchived bool) ([]domain.Plot, error) {
	if _, err := s.farms.owned(ctx, userID, farmID); err != nil {
		return nil, err
	}
	return s.plots.ListByFarm(ctx, farmID, includeArchived)
}

func (s *PlotService) Create(ctx context.Context, userID, farmID int64, in PlotInput) (*domain.Plot, error) {
	if _, err := s.farms.owned(ctx, userID, farmID); err != nil {
		return nil, err
	}
	if in.Name == nil {
		return nil, domain.ErrInvalidName
	}

	now := time.Now().Unix()
	plot := domain.Plot{FarmID: farmID, CreatedAt: now, UpdatedAt: now}
	applyPlot(&plot, in)
	plot.IsArchived = false

	if err := plot.Validate(); err != nil {
		return nil, err
	}
	if err := s.plots.Create(ctx, &plot); err != nil {
		return nil, err
	}
	return &plot, nil
}

func (s *PlotService) Update(ctx context.Context, userID, farmID, plotID int64, in PlotInput) (*domain.Plot, error) {
	current, err := s.ownedPlot(ctx, userID, farmID, plotID)
	if err != nil {
		return nil, err
	}

	updated := *current
	applyPlot(&updated, in)
	updated.UpdatedAt = time.Now().Unix()

	if err := updated.Validate(); err != nil {
		return nil, err
	}
	if err := s.plots.Update(ctx, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

// Archive — o'chirish o'rniga: issiqxonaga mavsumlar va jurnal
// yozuvlari bog'langan, ularni yo'qotmaslik kerak.
func (s *PlotService) Archive(ctx context.Context, userID, farmID, plotID int64) error {
	archived := true
	_, err := s.Update(ctx, userID, farmID, plotID, PlotInput{IsArchived: &archived})
	return err
}

// ownedPlot — issiqxona shu xo'jalikka, xo'jalik esa shu foydalanuvchiga
// tegishli ekanini tekshiradi. URL dagi farmID ni boshqa xo'jalikning
// plotID si bilan aralashtirib kirish mumkin bo'lmasin.
func (s *PlotService) ownedPlot(ctx context.Context, userID, farmID, plotID int64) (*domain.Plot, error) {
	if _, err := s.farms.owned(ctx, userID, farmID); err != nil {
		return nil, err
	}
	plot, err := s.plots.FindByID(ctx, plotID)
	if err != nil {
		return nil, err
	}
	if plot.FarmID != farmID {
		return nil, domain.ErrPlotNotFound
	}
	return plot, nil
}

func applyPlot(p *domain.Plot, in PlotInput) {
	if in.Name != nil {
		p.Name = strings.TrimSpace(*in.Name)
	}
	if in.AreaM2 != nil {
		p.AreaM2 = in.AreaM2
	}
	if in.DefaultVolumeL != nil {
		p.DefaultVolumeL = in.DefaultVolumeL
	}
	if in.IsArchived != nil {
		p.IsArchived = *in.IsArchived
	}
}
