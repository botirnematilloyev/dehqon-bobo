package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func TestFarm_Validate(t *testing.T) {
	valid := func() Farm {
		return Farm{SystemType: SystemSubstrate, MixingMethod: MixingDirect}
	}

	t.Run("direct mixing needs no injector", func(t *testing.T) {
		f := valid()
		require.NoError(t, f.Validate())
	})

	t.Run("ab mixing needs ratio and stock tank", func(t *testing.T) {
		f := valid()
		f.MixingMethod = MixingAB
		require.ErrorIs(t, f.Validate(), ErrInvalidMixing)

		f.InjectorRatio = ptr(int32(100))
		require.ErrorIs(t, f.Validate(), ErrInvalidMixing, "stock tank still missing")

		f.StockTankL = ptr(100.0)
		require.NoError(t, f.Validate())

		f.InjectorRatio = ptr(int32(0))
		require.ErrorIs(t, f.Validate(), ErrInvalidMixing)
	})

	t.Run("rejects unknown enums", func(t *testing.T) {
		f := valid()
		f.SystemType = 9
		require.ErrorIs(t, f.Validate(), ErrInvalidSystemType)

		f = valid()
		f.MixingMethod = 0
		require.ErrorIs(t, f.Validate(), ErrInvalidMixing)
	})

	t.Run("rejects blank and too long names", func(t *testing.T) {
		f := valid()
		f.Name = ptr("   ")
		require.ErrorIs(t, f.Validate(), ErrInvalidName)

		f.Name = ptr(strings.Repeat("a", 101))
		require.ErrorIs(t, f.Validate(), ErrInvalidName)

		f.Name = ptr("1-issiqxona")
		require.NoError(t, f.Validate())
	})
}

func TestSystemType_HasDrain(t *testing.T) {
	require.True(t, SystemSubstrate.HasDrain())
	require.True(t, SystemHydroponics.HasDrain())
	require.False(t, SystemGreenhouseSoil.HasDrain())
	require.False(t, SystemOpenField.HasDrain())
}

func TestPlot_Validate(t *testing.T) {
	require.NoError(t, (&Plot{Name: "1-issiqxona", AreaM2: ptr(100.0)}).Validate())
	require.ErrorIs(t, (&Plot{Name: ""}).Validate(), ErrInvalidName)
	require.ErrorIs(t, (&Plot{Name: "a", AreaM2: ptr(0.0)}).Validate(), ErrInvalidArea)
	require.ErrorIs(t, (&Plot{Name: "a", DefaultVolumeL: ptr(-5.0)}).Validate(), ErrInvalidArea)
}

func TestWaterAnalysis_Validate(t *testing.T) {
	base := func() WaterAnalysis {
		return WaterAnalysis{Source: WaterWell, EC: 1.1, Ca: 90, Mg: 30, HCO3Mmol: 4}
	}

	w := base()
	require.NoError(t, w.Validate())

	w = base()
	w.EC = -0.1
	require.ErrorIs(t, w.Validate(), ErrInvalidWaterValue)

	w = base()
	w.EC = 25
	require.ErrorIs(t, w.Validate(), ErrInvalidWaterValue)

	w = base()
	w.PH = ptr(15.0)
	require.ErrorIs(t, w.Validate(), ErrInvalidWaterValue)

	w = base()
	w.Na = ptr(-1.0)
	require.ErrorIs(t, w.Validate(), ErrInvalidWaterValue)

	w = base()
	w.Source = 7
	require.ErrorIs(t, w.Validate(), ErrInvalidWaterSource)
}

func TestMergeFertilizer(t *testing.T) {
	def := FertilizerDef{Code: "CaNO3", Kind: 1, Elements: map[string]float64{"Ca": 16.972, "N_NO3": 11.863}, StockGL: ptr(600.0)}

	eff := MergeFertilizer(def, nil)
	require.False(t, eff.Overridden)
	require.Equal(t, def.Elements, eff.Elements)

	o := &FertilizerOverride{Elements: map[string]float64{"Ca": 19, "N_NO3": 14.4, "N_NH4": 1.1}, StockGL: ptr(500.0), Brand: ptr("Yara")}
	eff = MergeFertilizer(def, o)
	require.True(t, eff.Overridden)
	require.Equal(t, 19.0, eff.Elements["Ca"])
	require.Equal(t, 500.0, *eff.StockGL)
	require.Equal(t, 16.972, eff.Default.Elements["Ca"], "default is kept for the reset button")

	// Faqat narx o'zgarsa tarkib standartligicha qoladi.
	eff = MergeFertilizer(def, &FertilizerOverride{PriceUZSPerKg: new(int64)})
	require.Equal(t, def.Elements, eff.Elements)
	require.Equal(t, 600.0, *eff.StockGL)
}

func TestFertilizerOverride_Validate(t *testing.T) {
	salt := FertilizerDef{Code: "KNO3", Kind: 1}
	acid := FertilizerDef{Code: "HNO3_38", Kind: 3}

	require.NoError(t, (&FertilizerOverride{Elements: map[string]float64{"K": 38, "N_NO3": 13.5}}).Validate(salt))
	require.ErrorIs(t, (&FertilizerOverride{Elements: map[string]float64{"Kalium": 38}}).Validate(salt), ErrInvalidElements)
	require.ErrorIs(t, (&FertilizerOverride{Elements: map[string]float64{"K": 0}}).Validate(salt), ErrInvalidElements)
	require.ErrorIs(t, (&FertilizerOverride{Elements: map[string]float64{"K": 70, "N_NO3": 40}}).Validate(salt), ErrInvalidElements, "over 100%")
	require.ErrorIs(t, (&FertilizerOverride{StockGL: ptr(-1.0)}).Validate(salt), ErrInvalidSetting)
	require.ErrorIs(t, (&FertilizerOverride{ConcentrationPct: ptr(38.0)}).Validate(salt), ErrInvalidSetting, "salts have no acid strength")

	require.NoError(t, (&FertilizerOverride{ConcentrationPct: ptr(56.0), DensityGML: ptr(1.35)}).Validate(acid))
	require.ErrorIs(t, (&FertilizerOverride{DensityGML: ptr(2.5)}).Validate(acid), ErrInvalidSetting)
}

func TestParamDef_Check(t *testing.T) {
	d := ParamDef{Key: "residual_hco3_mmol", Value: 0.5, Min: 0, Max: 3, FarmOverridable: true}
	require.NoError(t, d.Check(1))
	require.ErrorIs(t, d.Check(3.1), ErrInvalidSetting)

	d.FarmOverridable = false
	require.ErrorIs(t, d.Check(1), ErrParameterNotOverridable)

	eff := MergeParam(d, ptr(1.2))
	require.Equal(t, 1.2, eff.Value)
	require.Equal(t, 0.5, eff.Default)
	require.True(t, eff.Overridden)
}
