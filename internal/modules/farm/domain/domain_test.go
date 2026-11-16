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

func TestEstimateWater(t *testing.T) {
	for _, src := range []WaterSource{WaterWell, WaterCanal, WaterTap, WaterFilter} {
		w, err := EstimateWater(src)
		require.NoError(t, err)
		require.True(t, w.IsEstimate)
		require.Equal(t, src, w.Source)
		require.NoError(t, w.Validate(), "estimates must pass their own validation")
	}

	well, _ := EstimateWater(WaterWell)
	filtered, _ := EstimateWater(WaterFilter)
	require.Greater(t, well.HCO3Mmol, filtered.HCO3Mmol, "well water is harder than RO water")

	_, err := EstimateWater(0)
	require.ErrorIs(t, err, ErrInvalidWaterSource)
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
