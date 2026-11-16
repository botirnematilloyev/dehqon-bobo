package domain

import "errors"

var (
	// Begona xo'jalik ham "topilmadi" qaytaradi, "taqiqlangan" emas:
	// aks holda ID'larni sanab, kimda xo'jalik borligini bilib olish mumkin.
	ErrFarmNotFound          = errors.New("farm not found")
	ErrPlotNotFound          = errors.New("plot not found")
	ErrWaterAnalysisNotFound = errors.New("water analysis not found")

	ErrInvalidName        = errors.New("invalid name")
	ErrInvalidLocation    = errors.New("invalid region or district")
	ErrInvalidSystemType  = errors.New("invalid system type")
	ErrInvalidMixing      = errors.New("invalid mixing settings")
	ErrInvalidWaterSource = errors.New("invalid water source")
	ErrInvalidWaterValue  = errors.New("invalid water analysis value")
	ErrInvalidArea        = errors.New("invalid area or volume")
)
