package service

import (
	"math"

	"dehqon-bobo/internal/modules/calc/water"
	"dehqon-bobo/internal/modules/recipe/domain"
)

// CorrectionInput — aylanma (yopiq) tizim baki: eritma bir qismi
// ishlatilgan / bug'langan, EC o'zgargan. HPG "Корректор" yorlig'i.
type CorrectionInput struct {
	ECNow        float64  // bakda hozir o'lchangan EC
	VolumeNowL   float64  // bakda qolgan hajm
	VolumeFinalL float64  // to'ldirilgandan keyingi hajm
	Acids        []string // nil = retseptdagi kislotalar
}

type CorrectionResult struct {
	CurrentMgL        water.Nutrients // hozirgi taxminiy tarkib
	CorrectionMgL     water.Nutrients // quyiladigan eritma tarkibi
	CorrectionVolumeL float64
	// DiluteOnly — bakdagi eritma shunchalik quyuqki, qaysidir ion
	// tuzatuvchi eritmada manfiy bo'lardi: tuz emas, faqat suv qo'shiladi.
	DiluteOnly  bool
	WaterToAddL float64
	// Recipe — tuzatuvchi eritma retsepti (CorrectionVolumeL litr uchun).
	Recipe *Preview
}

// negligible — tuzatuvchi eritmada shundan kichik manfiy qiymat (mg/L)
// yaxlitlash shovqini hisoblanadi.
const negligible = 0.5

// Correct — HPG korrektor modeli (hpg.Correct bilan bir xil):
//
//	joriy = yangi eritma × (EC_hozir / EC_kutilgan)
//	V_tuz = V_yakuniy − V_hozir
//	C_tuz = (C_yakuniy·V_yakuniy − C_joriy·V_hozir) / V_tuz
//
// Yakuniy tarkib — retseptda erishilgan tarkib (suv bilan birga). Tuzatuvchi
// eritma o'sha suv va kislotalar bilan tayyorlanadi, shuning uchun uning
// retsepti oddiy hisob (Compute) bilan chiqadi: suv ayiriladi, bikarbonat
// neytrallanadi, HPG tuzlarni topadi.
func Correct(batch *domain.Batch, snap *Snapshot, farm domain.FarmData, in CorrectionInput) (*CorrectionResult, error) {
	ecExpected := snap.EC.Estimated
	if !(in.ECNow > 0 && in.ECNow <= 20) || !(in.VolumeNowL > 0) || !(in.VolumeFinalL > in.VolumeNowL) ||
		in.VolumeFinalL > maxVolumeL || ecExpected <= 0 {
		return nil, domain.ErrInvalidCorrection
	}
	k := in.ECNow / ecExpected
	vc, vf := in.VolumeNowL, in.VolumeFinalL
	vk := vf - vc

	a := snap.AchievedMgL
	final := water.Nutrients{NO3N: a.NO3N, NH4N: a.NH4N, P: a.P, K: a.K, Ca: a.Ca, Mg: a.Mg, S: a.S, Cl: a.Cl}
	scale := func(n water.Nutrients, f float64) water.Nutrients {
		return water.Nutrients{NO3N: n.NO3N * f, NH4N: n.NH4N * f, P: n.P * f, K: n.K * f, Ca: n.Ca * f, Mg: n.Mg * f, S: n.S * f, Cl: n.Cl * f}
	}
	cur := scale(final, k)
	mix := func(fin, c float64) float64 { return (fin*vf - c*vc) / vk }
	corr := water.Nutrients{
		NO3N: mix(final.NO3N, cur.NO3N), NH4N: mix(final.NH4N, cur.NH4N), P: mix(final.P, cur.P), K: mix(final.K, cur.K),
		Ca: mix(final.Ca, cur.Ca), Mg: mix(final.Mg, cur.Mg), S: mix(final.S, cur.S), Cl: mix(final.Cl, cur.Cl),
	}
	res := &CorrectionResult{CurrentMgL: cur, CorrectionMgL: corr, CorrectionVolumeL: vk}

	for _, v := range []float64{corr.NO3N, corr.NH4N, corr.P, corr.K, corr.Ca, corr.Mg, corr.S} {
		if v < -negligible {
			// Faqat suv: V·(EC_hozir − EC_suv) = V'·(EC_kutilgan − EC_suv).
			res.DiluteOnly = true
			if w := snap.Water.EC; ecExpected-w > 0 {
				total := vc * (in.ECNow - w) / (ecExpected - w)
				res.WaterToAddL = math.Max(0, math.Round((total-vc)*10)/10)
			}
			return res, nil
		}
	}

	var micro MicroJSON
	if snap.TargetMicroMgL != nil {
		micro = *snap.TargetMicroMgL
	}
	mixMicro := func(t float64) float64 { return math.Max(0, mix(t, t*k)) }
	clamp := func(v float64) float64 { return math.Max(0, v) }
	profile := domain.Profile{
		ID: batch.ProfileID, CropCode: snap.Profile.Crop, CropName: snap.Profile.CropName, Stage: batch.Stage,
		Reviewed: true, Title: snap.Profile.Title, Source: snap.Profile.Source,
		Targets: domain.Targets{
			NH4N: clamp(corr.NH4N), NO3N: clamp(corr.NO3N), P: clamp(corr.P), K: clamp(corr.K),
			Ca: clamp(corr.Ca), Mg: clamp(corr.Mg), S: clamp(corr.S), Cl: clamp(corr.Cl),
			Fe: mixMicro(micro.Fe), Mn: mixMicro(micro.Mn), B: mixMicro(micro.B),
			Zn: mixMicro(micro.Zn), Cu: mixMicro(micro.Cu), Mo: mixMicro(micro.Mo),
		},
		PHMin: snap.PH.Min, PHMax: snap.PH.Max,
	}
	acids := in.Acids
	if acids == nil {
		for _, ac := range snap.Acids {
			acids = append(acids, ac.Code)
		}
	}
	method := domain.MethodDirect
	p, err := Compute(profile, farm, PreviewInput{
		CropCode: snap.Profile.Crop, Stage: batch.Stage, Method: &method, VolumeL: &vk, Acids: acids,
	})
	if err != nil {
		return nil, err
	}
	res.Recipe = p
	return res, nil
}
