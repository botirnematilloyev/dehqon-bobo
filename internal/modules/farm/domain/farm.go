package domain

import "strings"

const maxNameLen = 100

type Farm struct {
	ID            int64
	OwnerID       int64
	Name          *string
	RegionID      *int32
	DistrictID    *int32
	SystemType    SystemType
	MixingMethod  MixingMethod
	InjectorRatio *int32
	StockTankL    *float64
	CreatedAt     int64
	UpdatedAt     int64
}

// Validate — farms jadvalidagi CHECK cheklovlarining Go'dagi aksi.
// Baza baribir to'sadi, lekin bu yerda tushunarli xato qaytariladi
// (500 o'rniga 400).
func (f *Farm) Validate() error {
	if f.Name != nil {
		if err := ValidateName(*f.Name); err != nil {
			return err
		}
	}
	if !f.SystemType.Valid() {
		return ErrInvalidSystemType
	}
	if !f.MixingMethod.Valid() {
		return ErrInvalidMixing
	}
	if f.MixingMethod == MixingAB {
		if f.InjectorRatio == nil || *f.InjectorRatio <= 0 || *f.InjectorRatio > 1000 {
			return ErrInvalidMixing
		}
		if f.StockTankL == nil || *f.StockTankL <= 0 || *f.StockTankL > 100000 {
			return ErrInvalidMixing
		}
	}
	return nil
}

// ValidateName — xo'jalik va issiqxona nomi uchun umumiy qoida.
func ValidateName(name string) error {
	n := strings.TrimSpace(name)
	if n == "" || len([]rune(n)) > maxNameLen {
		return ErrInvalidName
	}
	return nil
}
