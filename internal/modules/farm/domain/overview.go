package domain

// FarmOverview — Profil ekrani va bosh sahifa uchun bitta so'rovda
// kerak bo'ladigan hamma narsa: xo'jalik, faol suv tahlili, issiqxonalar.
type FarmOverview struct {
	Farm  Farm
	Water *WaterAnalysis
	Plots []Plot
}
