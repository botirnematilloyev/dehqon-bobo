package domain

import "encoding/json"

type BatchStatus int16

const (
	BatchPlanned BatchStatus = 1 // retsept saqlandi, hali tayyorlanmagan
	BatchMixed   BatchStatus = 2 // eritma tayyorlandi
)

// Tank — batch_items.tank: 0 = bitta bak (to'g'ridan usul), 1 A, 2 B, 3 C.
const (
	TankSingle int16 = 0
	TankA      int16 = 1
	TankB      int16 = 2
	TankC      int16 = 3
)

// Batch — jurnal yozuvi (solution_batches). Snapshot — hisob natijasi
// o'sha paytdagi holatda (service.Snapshot JSON).
type Batch struct {
	ID              int64
	FarmID          int64
	PlotID          *int64
	SeasonID        *int64
	CreatedBy       *int64
	ProfileID       int64
	WaterAnalysisID *int64
	Method          Method
	VolumeL         float64
	StockTankL      *float64
	InjectorRatio   *int32
	TargetEC        float64
	CalcVersion     string
	Snapshot        json.RawMessage
	Status          BatchStatus
	MixedAt         *int64
	CreatedAt       int64
	UpdatedAt       int64
	Items           []BatchItem

	// Ro'yxat uchun (crop_profiles + crops dan).
	CropCode string
	Stage    Stage
}

// BatchItem — qaysi idishga qaysi o'g'itdan qancha (batch_items).
type BatchItem struct {
	FertilizerCode string
	Tank           int16
	AmountG        *float64
	AmountML       *float64
	Position       int16
}

// BatchFilter — ro'yxat: BeforeID > 0 bo'lsa shundan eskilari (kursor).
type BatchFilter struct {
	PlotID   *int64
	BeforeID int64
	Limit    int
}
