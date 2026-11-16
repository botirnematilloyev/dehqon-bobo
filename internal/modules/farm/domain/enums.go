package domain

// SystemType — yetishtirish tizimi. farms.system_type bilan bir xil.
type SystemType int16

const (
	SystemGreenhouseSoil SystemType = 1 // issiqxona, tuproq
	SystemSubstrate      SystemType = 2 // substrat yoki tuvak
	SystemHydroponics    SystemType = 3
	SystemOpenField      SystemType = 4
)

func (s SystemType) Valid() bool {
	return s >= SystemGreenhouseSoil && s <= SystemOpenField
}

// HasDrain — drenaj EC o'lchanadigan tizimmi (substrat, gidroponika).
// O'lchash ekrani shunga qarab drenaj qatorini ko'rsatadi.
func (s SystemType) HasDrain() bool {
	return s == SystemSubstrate || s == SystemHydroponics
}

// MixingMethod — eritma qanday tayyorlanadi. farms.mixing_method.
type MixingMethod int16

const (
	MixingDirect MixingMethod = 1 // tuzlar ish eritmasi bakida eritiladi
	MixingAB     MixingMethod = 2 // A/B konsentrat + injektor
)

func (m MixingMethod) Valid() bool {
	return m == MixingDirect || m == MixingAB
}

// WaterSource — suv manbasi. water_analyses.source.
type WaterSource int16

const (
	WaterWell   WaterSource = 1 // quduq
	WaterCanal  WaterSource = 2 // ariq / kanal
	WaterTap    WaterSource = 3 // vodoprovod
	WaterFilter WaterSource = 4 // filtr / osmos
)

func (w WaterSource) Valid() bool {
	return w >= WaterWell && w <= WaterFilter
}
