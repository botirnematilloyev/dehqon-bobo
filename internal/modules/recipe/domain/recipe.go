// Package domain — retsept: ekin profili + xo'jalik suvi + o'g'itlar →
// qaysi tuzdan necha gramm. Modul `farm` va `catalog` ni import qilmaydi;
// ular portlar orqali (adapter cmd/api da) shu turlarga o'giriladi.
package domain

import (
	"errors"

	"dehqon-bobo/internal/modules/calc/inputs"
	"dehqon-bobo/internal/modules/calc/water"
)

var (
	ErrFarmNotFound    = errors.New("farm not found")
	ErrCropNotFound    = errors.New("crop not found")
	ErrProfileNotFound = errors.New("no profile for this crop and stage")
	ErrInvalidStage    = errors.New("invalid stage")
	ErrInvalidMethod   = errors.New("invalid method or volume")
	// ErrWaterRequired — xo'jalikda faol suv tahlili yo'q. Birinchi sozlash
	// taxminiy tahlil yozadi, shuning uchun bu odatda ro'y bermaydi.
	ErrWaterRequired = errors.New("active water analysis required")
	ErrInvalidAcid   = errors.New("unknown or incomplete acid")
	// ErrCatalogIncomplete — katalogda kerakli tuz yoki parametr yo'q yoki
	// noto'g'ri (agronom tuzatishi kerak).
	ErrCatalogIncomplete = errors.New("catalog is missing a salt or calc parameter")

	ErrBatchNotFound = errors.New("recipe not found")
	ErrPlotNotFound  = errors.New("plot not found in this farm")
	// ErrInfeasible — manfiy gramm: tanlangan tuzlar bilan profilni yig'ib
	// bo'lmaydi. Preview ko'rsatadi, lekin saqlab bo'lmaydi.
	ErrInfeasible = errors.New("recipe has negative amounts and cannot be mixed")
	// ErrAlreadyMixed — tayyorlangan eritma jurnaldan o'chirilmaydi.
	ErrAlreadyMixed = errors.New("recipe is already mixed")
)

type Stage int16

const (
	StageSeedling   Stage = 1
	StageVegetative Stage = 2
	StageFruiting   Stage = 3
)

func (s Stage) Valid() bool { return s >= StageSeedling && s <= StageFruiting }

type Method int16

const (
	MethodDirect Method = 1 // tuzlar ish eritmasi bakida eritiladi
	MethodAB     Method = 2 // A/B konsentrat + injektor
)

// Targets — maqsad eritma, mg/L (mikro ham mg/L).
type Targets struct {
	NH4N, NO3N, P, K, Ca, Mg, S, Cl float64
	Fe, Mn, B, Zn, Cu, Mo           float64
}

// Profile — ekin + bosqich uchun rasmiy profil (catalog'dan).
type Profile struct {
	ID       int64
	CropCode string
	CropName map[string]string
	// FAO-29: suv EC shundan oshsa hosil kamayadi, dS/m (= mS/sm).
	SalinityECw *float64
	Stage       Stage
	Reviewed    bool // agronom tasdiqlagan (status = approved)
	Title       *string
	Source      *string
	Targets     Targets
	ECTarget    *float64
	PHMin       float64
	PHMax       float64
}

// Fertilizer — xo'jalik uchun effective o'g'it (standart + o'zgartirish).
type Fertilizer struct {
	inputs.Fertilizer
	Name         map[string]string
	Tank         int16 // 1 A, 2 B, 3 C
	SolubilityGL *float64
	// PriceUZSPerKg — dehqon kiritgan narx (farm_fertilizers); nil = noma'lum.
	PriceUZSPerKg *int64
}

// Plot — xo'jalikdagi issiqxona (farm modulidan).
type Plot struct {
	ID     int64
	AreaM2 *float64
}

// Water — xo'jalikning faol suv tahlili.
type Water struct {
	ID         int64
	IsEstimate bool
	PH         *float64
	Analysis   water.Analysis
}

// FarmData — xo'jalikdan retsept uchun kerak bo'lgan hamma narsa.
type FarmData struct {
	FarmID        int64
	Method        Method
	InjectorRatio *int32
	StockTankL    *float64
	Water         *Water // nil = faol tahlil yo'q
	Plots         []Plot // arxivlanmagan issiqxonalar
	Fertilizers   []Fertilizer
	Params        map[string]float64 // effective calc_parameters
}

// Plot — shu xo'jalikning arxivlanmagan issiqxonasi.
func (f *FarmData) Plot(id int64) (Plot, bool) {
	for _, p := range f.Plots {
		if p.ID == id {
			return p, true
		}
	}
	return Plot{}, false
}
