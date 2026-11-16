package hpg

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// Golden test: testdata/golden.json HPG'ning O'ZIDAN (hpg.html funksiyalari
// node'da o'zgartirilmasdan ishga tushirilgan) olingan. Qayta yaratish:
// scripts/hpg-golden/README.md.

type goldenFile struct {
	Source string       `json:"source"`
	Commit string       `json:"commit"`
	Cases  []goldenCase `json:"cases"`
}

type goldenCase struct {
	Name   string       `json:"name"`
	Input  goldenInput  `json:"input"`
	Output goldenOutput `json:"output"`
}

type goldenInput struct {
	Macro        map[string]float64            `json:"macro"`
	Micro        map[string]float64            `json:"micro"`
	MicroPercent map[string]float64            `json:"microPercent"`
	Salts        map[string]map[string]float64 `json:"salts"`
	UseK2SO4     bool                          `json:"useK2SO4"`
	UseMgNO3     bool                          `json:"useMgNO3"`
	Complex      bool                          `json:"complex"`
	Volume       float64                       `json:"volume"`
	GL           map[string]float64            `json:"gl"`
	GML          map[string]float64            `json:"gml"`
	TankA        float64                       `json:"tankA"`
	TankB        float64                       `json:"tankB"`
}

type tableItem struct {
	GL  float64 `json:"g_l"`
	GML float64 `json:"g_ml"`
	ML  float64 `json:"ml"`
	Gr  float64 `json:"gr"`
}

type goldenOutput struct {
	S                  float64                    `json:"S"`
	EC                 float64                    `json:"EC"`
	CaFromBalance      float64                    `json:"CaFromBalance"`
	Grams              map[string]string          `json:"grams"`
	TotalSalts         string                     `json:"totalSalts"`
	Ratios             map[string]string          `json:"ratios"`
	NH4Ratio           *string                    `json:"nh4Ratio"`
	MicroGrams         map[string]string          `json:"microGrams"`
	MicroAfter         map[string]string          `json:"microAfter"`
	TotalMicroGrams    string                     `json:"totalMicroGrams"`
	ComplexComposition map[string]float64         `json:"complexComposition"`
	ConcentrateA       map[string]tableItem       `json:"concentrateA"`
	ConcentrateB       map[string]tableItem       `json:"concentrateB"`
	SumA               string                     `json:"sumA"`
	SumB               string                     `json:"sumB"`
	TankA              string                     `json:"tankA"`
	TankB              string                     `json:"tankB"`
	Density            map[string]float64         `json:"density"`
	Corrector          map[string]json.RawMessage `json:"corrector"`
}

func loadGolden(t *testing.T) goldenFile {
	t.Helper()
	raw, err := os.ReadFile("testdata/golden.json")
	require.NoError(t, err)
	var g goldenFile
	require.NoError(t, json.Unmarshal(raw, &g))
	require.NotEmpty(t, g.Cases)
	require.Regexp(t, `^[0-9a-f]{40}$`, g.Commit, "golden must be pinned to an HPG commit")
	return g
}

// ---------- kirishni HPG qoidalari bilan yig'ish ----------

func saltsFrom(in map[string]map[string]float64) Salts {
	// saltCompositions: fayldagi qiymat, yo'q bo'lsa generator standartni qo'ygan.
	return Salts{
		CaNO3Ca: in["CaNO3"]["Ca"], CaNO3NO3: in["CaNO3"]["NO3"], CaNO3NH4: in["CaNO3"]["NH4"],
		KNO3K: in["KNO3"]["K"], KNO3NO3: in["KNO3"]["NO3"],
		NH4NO3NH4: in["NH4NO3"]["NH4"], NH4NO3NO3: in["NH4NO3"]["NO3"],
		MgSO4Mg: in["MgSO4"]["Mg"], MgSO4S: in["MgSO4"]["S"],
		KH2PO4K: in["KH2PO4"]["K"], KH2PO4P: in["KH2PO4"]["P"],
		K2SO4K: in["K2SO4"]["K"], K2SO4S: in["K2SO4"]["S"],
		MgNO3Mg: in["MgNO3"]["Mg"], MgNO3NO3: in["MgNO3"]["NO3"],
		CaCl2Ca: in["CaCl2"]["Ca"], CaCl2Cl: in["CaCl2"]["Cl"],
	}
}

// densitySalts — HPG getSaltComposition: maydon 0 bo'lsa standart.
func densitySalts(s Salts) Salts {
	d := DefaultSalts()
	s.CaNO3Ca = orDefault(s.CaNO3Ca, d.CaNO3Ca)
	s.KNO3K = orDefault(s.KNO3K, d.KNO3K)
	s.NH4NO3NO3 = orDefault(s.NH4NO3NO3, d.NH4NO3NO3)
	s.MgNO3Mg = orDefault(s.MgNO3Mg, d.MgNO3Mg)
	s.MgSO4Mg = orDefault(s.MgSO4Mg, d.MgSO4Mg)
	s.KH2PO4K = orDefault(s.KH2PO4K, d.KH2PO4K)
	s.K2SO4K = orDefault(s.K2SO4K, d.K2SO4K)
	s.CaCl2Ca = orDefault(s.CaCl2Ca, d.CaCl2Ca)
	return s
}

func macroFrom(m map[string]float64) Macro {
	return Macro{NO3: m["NO3"], NH4: m["NH4"], P: m["P"], K: m["K"], Ca: m["Ca"], Mg: m["Mg"], S: m["S"], Cl: m["Cl"]}
}

func microFrom(m map[string]float64) Micro {
	return Micro{Fe: m["Fe"], Mn: m["Mn"], B: m["B"], Zn: m["Zn"], Cu: m["Cu"], Mo: m["Mo"], Co: m["Co"], Si: m["Si"]}
}

func parse(t *testing.T, s string) float64 {
	t.Helper()
	if s == "" {
		return math.NaN()
	}
	v, err := strconv.ParseFloat(s, 64)
	require.NoError(t, err, s)
	return v
}

// jsParse — `parseFloat(x) || 0`.
func jsParse(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) {
		return 0
	}
	return v
}

var numRe = regexp.MustCompile(`-?\d+(?:\.\d+)?`)

func numbers(t *testing.T, s string) []float64 {
	t.Helper()
	var out []float64
	for _, m := range numRe.FindAllString(s, -1) {
		out = append(out, parse(t, m))
	}
	return out
}

// ---------- testlar ----------

func TestGolden(t *testing.T) {
	g := loadGolden(t)

	for _, c := range g.Cases {
		t.Run(c.Name, func(t *testing.T) {
			in, out := c.Input, c.Output
			volume := in.Volume
			salts := saltsFrom(in.Salts)
			mode := SaltMode{UseK2SO4: in.UseK2SO4, UseMgNO3: in.UseMgNO3}

			// --- balans va EC: bit-ma-bit ---
			m := Balanced(macroFrom(in.Macro))
			require.Equal(t, out.S, m.S, "S from balance")
			require.Equal(t, out.EC, EC(m), "EC")
			require.Equal(t, out.CaFromBalance, CalciumFromBalance(m), "Ca from balance")

			// --- tuzlar grammi (HPG ko'rsatgan satrlar bilan) ---
			dose := DosePer10L(m, salts, mode)
			grams, total := Weights(dose, volume)
			d := WeightDecimals(volume)
			for _, s := range MacroSalts {
				require.Equal(t, out.Grams[string(s)], ToFixed(grams.Get(s), d), "grams %s", s)
			}
			require.Equal(t, out.TotalSalts, ToFixed(total, d), "total salts")

			// --- nisbatlar ---
			ratios, ok := Ratios(m)
			if len(out.Ratios) == 0 {
				require.False(t, ok, "HPG skipped the ratio matrix")
			} else {
				require.True(t, ok)
				for k, want := range out.Ratios {
					require.Equal(t, want, Display(ratios[k], 3), "ratio %s", k)
				}
			}
			if out.NH4Ratio != nil {
				r, ok := NH4Ratio(m)
				require.True(t, ok)
				require.Equal(t, *out.NH4Ratio, Display(r, 3))
			}

			// --- mikro ---
			micro, pct := microFrom(in.Micro), microFrom(in.MicroPercent)
			microAfter := micro
			if !in.Complex {
				mg := MicroGrams(micro, pct, volume)
				for _, k := range microKeys {
					got := ToFixed(mg.get(k), 5)
					require.Equal(t, out.MicroGrams[k], got, "micro grams %s", k)
				}
				parsed := Micro{
					Fe: parse(t, out.MicroGrams["Fe"]), Mn: parse(t, out.MicroGrams["Mn"]), B: parse(t, out.MicroGrams["B"]),
					Zn: parse(t, out.MicroGrams["Zn"]), Cu: parse(t, out.MicroGrams["Cu"]), Mo: parse(t, out.MicroGrams["Mo"]),
					Co: parse(t, out.MicroGrams["Co"]), Si: parse(t, out.MicroGrams["Si"]),
				}
				comp, sum, ok := ComplexComposition(micro, parsed, volume)
				require.Equal(t, out.TotalMicroGrams, ToFixed(sum, 5))
				if ok {
					for _, k := range microKeys {
						require.Equal(t, out.ComplexComposition[k], comp.get(k), "composition %s", k)
					}
				}
			} else {
				cg := ComplexGrams(micro.B, pct.B, volume)
				require.Equal(t, out.TotalMicroGrams, ToFixed(cg, 5), "complex grams")
				microAfter = MicroFromComplex(cg, pct, volume, micro.B)
				for _, k := range microKeys {
					require.Equal(t, parse(t, out.MicroAfter[k]), microAfter.get(k), "micro from complex %s", k)
				}
			}

			// --- konsentratlar (HPG `||` standartlari bilan) ---
			glDef := map[string]float64{"CaNO3": 600, "KNO3": 250, "NH4NO3": 100, "MgNO3": 500, "CaCl2": 100, "MgSO4": 600, "KH2PO4": 150, "K2SO4": 100}
			stocks := map[Salt]Stock{}
			for _, s := range MacroSalts {
				stocks[s] = Stock{GL: orDefault(in.GL[string(s)], glDef[string(s)]), GML: orDefault(in.GML[string(s)], 1)}
			}
			microStock := map[string]Stock{}
			for _, k := range append(append([]string{}, microKeys...), "Cmplx") {
				microStock[k] = Stock{GL: orDefault(in.GL[k], 10), GML: orDefault(in.GML[k], 1)}
			}
			pctC := Micro{
				Fe: orDefault(pct.Fe, 20.1), Mn: orDefault(pct.Mn, 36.4), B: orDefault(pct.B, 17.5), Zn: orDefault(pct.Zn, 22.7),
				Cu: orDefault(pct.Cu, 25.5), Mo: orDefault(pct.Mo, 54.3), Co: orDefault(pct.Co, 13.0), Si: orDefault(pct.Si, 7.0),
			}
			weighed := SaltGrams{
				CaNO3: jsParse(out.Grams["CaNO3"]), KNO3: jsParse(out.Grams["KNO3"]), NH4NO3: jsParse(out.Grams["NH4NO3"]),
				MgSO4: jsParse(out.Grams["MgSO4"]), KH2PO4: jsParse(out.Grams["KH2PO4"]), K2SO4: jsParse(out.Grams["K2SO4"]),
				MgNO3: jsParse(out.Grams["MgNO3"]), CaCl2: jsParse(out.Grams["CaCl2"]),
			}
			microForConc := Micro{}
			for _, k := range microKeys {
				microForConc = setMicro(microForConc, k, jsParse(out.MicroAfter[k]))
			}
			conc := CalcConcentrates(ConcentrateInput{
				Volume: orDefault(volume, 10), Grams: weighed, Micro: microForConc, MicroPct: pctC, Complex: in.Complex,
				Salts: stocks, MicroStock: microStock, TankAml: orDefault(in.TankA, 500), TankBml: orDefault(in.TankB, 500),
			})
			for k, want := range out.ConcentrateA {
				got := conc.A[k]
				require.Equal(t, want.ML, got.ML, "A %s ml", k)
				require.Equal(t, want.Gr, got.Weight, "A %s grams", k)
			}
			for k, want := range out.ConcentrateB {
				got := conc.B[k]
				require.Equal(t, want.ML, got.ML, "B %s ml", k)
				require.Equal(t, want.Gr, got.Weight, "B %s grams", k)
			}

			a := numbers(t, out.SumA)
			require.Equal(t, []float64{parse(t, ToFixed(conc.SumA.Volume, 2)), parse(t, ToFixed(conc.SumA.Mass, 2)), parse(t, ToFixed(conc.SumA.Density, 2))}, a, out.SumA)

			b := numbers(t, out.SumB)
			require.Equal(t, parse(t, ToFixed(jsRound(conc.SumB.Volume*10)/10, 1)), b[0], "B volume")
			if in.Complex {
				require.Equal(t, parse(t, ToFixed(conc.SumB.Mass, 2)), b[1], "B mass")
				require.Equal(t, parse(t, ToFixed(conc.SumB.Density, 2)), b[2], "B density")
			} else {
				// HPG farqi: HPG massaga misni qo'shmaydi. Ikkala yig'indi ham
				// 0.01 gacha yaxlitlangani uchun farq ±0.01 ichida.
				require.InDelta(t, b[1]+conc.B["Cu"].Weight, conc.SumB.Mass, 0.0101,
					"our B mass = HPG mass + copper sulfate")
			}

			ta := numbers(t, out.TankA)
			require.Equal(t, []float64{conc.TankA.Ratio, 1, conc.TankA.AddWater, conc.TankA.PerLitre, 1}, ta, out.TankA)
			tb := numbers(t, out.TankB)
			require.Equal(t, []float64{conc.TankB.Ratio, 1, conc.TankB.AddWater, conc.TankB.PerLitre, 1}, tb, out.TankB)

			// --- zichlik ---
			ds := densitySalts(salts)
			for s, want := range out.Density {
				require.Equal(t, want, Density(Salt(s), in.GL[s], ds), "density %s", s)
			}

			// --- korrektor ---
			checkCorrector(t, in, out)
		})
	}
}

func setMicro(m Micro, k string, v float64) Micro {
	switch k {
	case "Fe":
		m.Fe = v
	case "Mn":
		m.Mn = v
	case "B":
		m.B = v
	case "Zn":
		m.Zn = v
	case "Cu":
		m.Cu = v
	case "Mo":
		m.Mo = v
	case "Co":
		m.Co = v
	case "Si":
		m.Si = v
	}
	return m
}

func checkCorrector(t *testing.T, in goldenInput, out goldenOutput) {
	t.Helper()

	cols := map[string]map[string]string{}
	for _, col := range []string{"init", "curr", "corr", "final"} {
		var m map[string]string
		require.NoError(t, json.Unmarshal(out.Corrector[col], &m))
		cols[col] = m
	}
	var currEC, corrVol, protocol string
	require.NoError(t, json.Unmarshal(out.Corrector["currEC"], &currEC))
	require.NoError(t, json.Unmarshal(out.Corrector["corrVolume"], &corrVol))
	require.NoError(t, json.Unmarshal(out.Corrector["protocol"], &protocol))

	ions := Ions{NO3: in.Macro["NO3"], NH4: in.Macro["NH4"], P: in.Macro["P"], K: in.Macro["K"], Ca: in.Macro["Ca"], Mg: in.Macro["Mg"], Cl: in.Macro["Cl"]}
	res := Correct(CorrectorInput{Initial: ions, CurrentEC: parse(t, currEC), CurrentVolume: 10, FinalVolume: 30, Final: ions})

	require.Equal(t, corrVol, ToFixed(res.CorrectionVolume, 1))

	check := func(name string, c CorrectorColumn, rawIons bool) {
		h := cols[name]
		for _, el := range []string{"NO3", "NH4", "P", "K", "Ca", "Mg", "Cl"} {
			got := curGet(c.Ions, el)
			if rawIons {
				require.Equal(t, parse(t, h[el]), got, "%s %s", name, el)
			} else {
				require.Equal(t, h[el], ToFixed(got, ionDecimals(el)), "%s %s", name, el)
			}
		}
		require.Equal(t, h["S"], ToFixed(c.S, 3), "%s S", name)
		require.Equal(t, h["EC"], ToFixed(c.EC, 3), "%s EC", name)
		require.Equal(t, h["N"], ToFixed(c.N, 3), "%s N", name)
	}
	check("init", res.Initial, true)
	check("curr", res.Current, false)
	check("corr", res.Correction, false)
	check("final", res.Final, true)

	p := BuildProtocol(res, 10, 30)
	want := []float64{
		float64(p.VolumeChangePct), float64(p.OldSharePct), float64(p.ECChangePct), float64(p.NChangePct),
	}
	for _, el := range []string{"NO3", "NH4", "P", "K", "Ca", "Mg"} {
		want = append(want, float64(p.IonChangePct[el]))
	}
	want = append(want, float64(p.SChangePPM), float64(p.ClChangePPM))
	for _, k := range []string{"NH4:NO3", "K:N", "K:Ca", "K:Mg"} {
		want = append(want, parse(t, ToFixed(p.Before[k], 3)), parse(t, ToFixed(p.After[k], 3)))
	}
	// Matnda "NO3", "NH4" kabi raqamli nomlar bor, shuning uchun faqat
	// ":" dan keyingi, "до/после" va "ppm" qiymatlari olinadi.
	filtered := []float64{}
	for _, line := range regexp.MustCompile(`:\s*(-?\d+(?:\.\d+)?)%?|до\s+(-?\d+\.\d+)\s+после\s+(-?\d+\.\d+)|на:\s*(-?\d+)\s*ppm`).FindAllStringSubmatch(protocol, -1) {
		for _, g := range line[1:] {
			if g != "" {
				filtered = append(filtered, parse(t, g))
			}
		}
	}
	require.Equal(t, want, filtered, fmt.Sprintf("protocol:\n%s", protocol))
}
