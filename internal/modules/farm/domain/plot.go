package domain

// Plot — bitta issiqxona yoki dala bo'lagi ("Issiqxonalarim").
type Plot struct {
	ID             int64
	FarmID         int64
	Name           string
	AreaM2         *float64
	DefaultVolumeL *float64
	IsArchived     bool
	CreatedAt      int64
	UpdatedAt      int64
}

func (p *Plot) Validate() error {
	if err := ValidateName(p.Name); err != nil {
		return err
	}
	if p.AreaM2 != nil && (*p.AreaM2 <= 0 || *p.AreaM2 > 10_000_000) {
		return ErrInvalidArea
	}
	if p.DefaultVolumeL != nil && (*p.DefaultVolumeL <= 0 || *p.DefaultVolumeL > 1_000_000) {
		return ErrInvalidArea
	}
	return nil
}
