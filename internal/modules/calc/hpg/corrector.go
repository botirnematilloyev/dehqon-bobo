package hpg

import "strconv"

// Ions — korrektor ustunidagi ionlar, mg/L.
type Ions struct {
	NO3, NH4, P, K, Ca, Mg, Cl float64
}

func (i Ions) macro() Macro {
	return Macro{NO3: i.NO3, NH4: i.NH4, P: i.P, K: i.K, Ca: i.Ca, Mg: i.Mg, Cl: i.Cl}
}

// CorrectorColumn — bitta ustun: ionlar, balansdan S, EC va N.
type CorrectorColumn struct {
	Ions
	S, EC, N float64
}

// CorrectorInput — HPG "Корректор" yorlig'i.
//
// HPG modeli: bakdagi eritma suv bug'lanib quyuqlashgan, tarkibi
// boshlang'ichga proporsional (ionlar EC nisbatida o'sgan). Maqsad —
// FinalVolume litrda Final profilga yetish. Buning uchun CurrentVolume
// ustiga qancha va qanday tarkibli tuzatuvchi eritma quyish kerak.
type CorrectorInput struct {
	Initial       Ions
	CurrentEC     float64
	CurrentVolume float64 // HPG web-versiyasida doim 10 L ("условно")
	FinalVolume   float64
	Final         Ions
}

type CorrectorResult struct {
	Initial, Current, Correction, Final CorrectorColumn
	CorrectionVolume                    float64
}

// HPG UI oraliq qiymatlarni maydonga yozib (toFixed), keyin qaytadan
// o'qiydi. Natija bir xil bo'lishi uchun shu yaxlitlash qayta yaratilgan.
func fixed(v float64, d int) float64 {
	f, _ := strconv.ParseFloat(ToFixed(v, d), 64)
	return f
}

func ionDecimals(name string) int {
	if name == "NO3" || name == "NH4" {
		return 2
	}
	return 3
}

func column(i Ions) CorrectorColumn {
	m := i.macro()
	return CorrectorColumn{
		Ions: i,
		S:    fixed(SulfurFromBalance(m), 3),
		EC:   fixed(EC(m), 3),
		N:    fixed(i.NO3+i.NH4, 3),
	}
}

func mapIons(i Ions, f func(name string, v float64) float64) Ions {
	return Ions{
		NO3: f("NO3", i.NO3), NH4: f("NH4", i.NH4), P: f("P", i.P), K: f("K", i.K),
		Ca: f("Ca", i.Ca), Mg: f("Mg", i.Mg), Cl: f("Cl", i.Cl),
	}
}

// Correct — HPG: korrection.
//
//	joriy = boshlang'ich × (EC_joriy / EC_boshlang'ich)
//	V_tuzatuvchi = V_yakuniy − V_joriy
//	C_tuzatuvchi = (C_yakuniy·V_yakuniy − C_joriy·V_joriy) / V_tuzatuvchi
func Correct(in CorrectorInput) CorrectorResult {
	var res CorrectorResult

	res.Initial = column(in.Initial)

	ec0 := orDefault(res.Initial.EC, 1)
	ec1 := orDefault(in.CurrentEC, ec0)
	kEC := 1.0
	if ec0 != 0 {
		kEC = ec1 / ec0
	}

	cur := mapIons(in.Initial, func(n string, v float64) float64 { return fixed(v*kEC, ionDecimals(n)) })
	res.Current = column(cur)
	res.Current.EC = fixed(ec1, 3) // joriy EC o'lchangan, hisoblanmaydi

	vK := in.FinalVolume - in.CurrentVolume
	res.CorrectionVolume = fixed(vK, 1)

	corr := mapIons(in.Final, func(n string, c2 float64) float64 {
		c1 := curGet(cur, n)
		ck := 0.0
		if vK != 0 {
			ck = (float64(c2*in.FinalVolume) - float64(c1*in.CurrentVolume)) / vK
		}
		return fixed(ck, ionDecimals(n))
	})
	res.Correction = column(corr)
	res.Final = column(in.Final)
	return res
}

func curGet(i Ions, n string) float64 {
	switch n {
	case "NO3":
		return i.NO3
	case "NH4":
		return i.NH4
	case "P":
		return i.P
	case "K":
		return i.K
	case "Ca":
		return i.Ca
	case "Mg":
		return i.Mg
	case "Cl":
		return i.Cl
	}
	return 0
}

// Protocol — tuzatish bayonnomasi (foiz o'zgarishlar va nisbatlar).
// HPG: updateCorrectionProtocol.
type Protocol struct {
	VolumeChangePct int
	OldSharePct     int
	ECChangePct     int
	NChangePct      int
	IonChangePct    map[string]int // NO3, NH4, P, K, Ca, Mg
	SChangePPM      int
	ClChangePPM     int
	Before, After   map[string]float64 // NH4:NO3, K:N, K:Ca, K:Mg
}

// BuildProtocol — joriy (CurrentVolume) va yakuniy ustunlardan.
func BuildProtocol(r CorrectorResult, currentVolume, finalVolume float64) Protocol {
	pct := func(after, before float64) int { return int(jsRound(float64((after - before) / before * 100))) }
	c, f := r.Current, r.Final

	// HPG `|| 1` / `|| 0` standartlari.
	ec1, ec2 := orDefault(c.EC, 1), orDefault(f.EC, 1)
	no31, no32 := orDefault(c.NO3, 1), orDefault(f.NO3, 1)
	nh41, nh42 := orDefault(c.NH4, 1), orDefault(f.NH4, 1)
	p1, p2 := orDefault(c.P, 1), orDefault(f.P, 1)
	k1, k2 := orDefault(c.K, 1), orDefault(f.K, 1)
	ca1, ca2 := orDefault(c.Ca, 1), orDefault(f.Ca, 1)
	mg1, mg2 := orDefault(c.Mg, 1), orDefault(f.Mg, 1)

	return Protocol{
		VolumeChangePct: int(jsRound(float64((finalVolume - currentVolume) / currentVolume * 100))),
		OldSharePct:     int(jsRound(float64(currentVolume / finalVolume * 100))),
		ECChangePct:     pct(ec2, ec1),
		NChangePct:      pct(f.N, c.N),
		IonChangePct: map[string]int{
			"NO3": pct(no32, no31), "NH4": pct(nh42, nh41), "P": pct(p2, p1),
			"K": pct(k2, k1), "Ca": pct(ca2, ca1), "Mg": pct(mg2, mg1),
		},
		SChangePPM:  int(jsRound(f.S - c.S)),
		ClChangePPM: int(jsRound(f.Cl - c.Cl)),
		Before: map[string]float64{
			"NH4:NO3": nh41 / no31, "K:N": k1 / c.N, "K:Ca": k1 / ca1, "K:Mg": k1 / mg1,
		},
		After: map[string]float64{
			"NH4:NO3": nh42 / no32, "K:N": k2 / f.N, "K:Ca": k2 / ca2, "K:Mg": k2 / mg2,
		},
	}
}
