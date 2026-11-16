package domain

import (
	"errors"
	"math"

	"dehqon-bobo/internal/modules/calc/hpg"
)

var (
	ErrCropNotFound        = errors.New("crop not found")
	ErrCropProfileNotFound = errors.New("crop profile not found")
	ErrInvalidStage        = errors.New("stage must be seedling, vegetative or fruiting")
	// ErrIonImbalance — anionlar (NO3 + H2PO4 + Cl) kationlardan ko'p:
	// balansdan chiqqan SO4 manfiy. Bunday eritmani tuzlardan yig'ib
	// bo'lmaydi — kation (K, Ca, Mg) qo'shish yoki NO3/P/Cl ni kamaytirish kerak.
	ErrIonImbalance = errors.New("anions exceed cations: sulfate from ion balance would be negative")
	// ErrProfileLocked — tasdiqlangan yoki arxivdagi profil o'zgartirilmaydi:
	// unga jurnal yozuvlari bog'langan. Yangi draft yaratib tasdiqlanadi.
	ErrProfileLocked = errors.New("only draft profiles can be edited")
)

type Crop struct {
	ID   int32
	Code string
	Name map[string]string
	// FAO-29: hosil kamaya boshlaydigan sug'orish suvi EC, dS/m.
	SalinityECwThreshold *float64
	SortOrder            int16
	IsActive             bool
}

type Stage int16

const (
	StageSeedling   Stage = 1 // ko'chat
	StageVegetative Stage = 2 // o'sish
	StageFruiting   Stage = 3 // hosil
)

func (s Stage) Valid() bool { return s >= StageSeedling && s <= StageFruiting }

type ProfileStatus int16

const (
	ProfileDraft    ProfileStatus = 1
	ProfileApproved ProfileStatus = 2
	ProfileArchived ProfileStatus = 3
)

// Targets — maqsad eritma, mg/L. S kiritilmaydi: ion balansidan hisoblanadi.
type Targets struct {
	NH4N, NO3N, P, K, Ca, Mg, S, Cl float64
	Fe, Mn, B, Zn, Cu, Mo           float64
}

// Macro — HPG balansi uchun.
func (t Targets) Macro() hpg.Macro {
	return hpg.Macro{NO3: t.NO3N, NH4: t.NH4N, P: t.P, K: t.K, Ca: t.Ca, Mg: t.Mg, S: t.S, Cl: t.Cl}
}

// CropProfile — rasmiy (agronom) profil: ekin + bosqich uchun maqsad.
type CropProfile struct {
	ID         int64
	CropID     int32
	CropCode   string
	Stage      Stage
	Title      *string
	Status     ProfileStatus
	Source     *string
	Targets    Targets
	ECTarget   *float64
	PHMin      float64
	PHMax      float64
	ReviewedBy *int64
	ReviewedAt *int64
	CreatedAt  int64
	UpdatedAt  int64
}

// Normalize — S ni ion balansidan hisoblaydi va qiymatlarni tekshiradi.
// Chegaralar fiziologik emas, xatoni ushlash uchun: makro ≤ 1000 mg/L,
// mikro ≤ 20 mg/L (eng yuqori Fe tavsiyalari ~5 mg/L).
func (p *CropProfile) Normalize() error {
	if !p.Stage.Valid() {
		return ErrInvalidStage
	}
	t := &p.Targets
	in := func(v, max float64) bool { return !math.IsNaN(v) && v >= 0 && v <= max }
	for _, v := range []float64{t.NH4N, t.NO3N, t.P, t.K, t.Ca, t.Mg, t.Cl} {
		if !in(v, 1000) {
			return ErrInvalidValue
		}
	}
	for _, v := range []float64{t.Fe, t.Mn, t.B, t.Zn, t.Cu, t.Mo} {
		if !in(v, 20) {
			return ErrInvalidValue
		}
	}
	if !(p.PHMin >= 4 && p.PHMax <= 8 && p.PHMin <= p.PHMax) {
		return ErrInvalidValue
	}
	if p.ECTarget != nil && !(*p.ECTarget >= 0.2 && *p.ECTarget <= 8) {
		return ErrInvalidValue
	}

	s := hpg.SulfurFromBalance(t.Macro())
	if s < -0.05 { // ~0.003 meq/L — yaxlitlash shovqini
		return ErrIonImbalance
	}
	t.S = math.Round(math.Max(s, 0)*1000) / 1000
	return nil
}
