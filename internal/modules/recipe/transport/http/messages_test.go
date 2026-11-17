package http

import (
	"testing"

	"dehqon-bobo/internal/platform/i18n"

	"github.com/stretchr/testify/require"
)

func TestMessagesRegistered(t *testing.T) {
	for _, k := range []i18n.Key{msgFarmNotFound, msgCropNotFound, msgProfileNotFound, msgInvalidStage, msgInvalidMethod, msgWaterRequired, msgInvalidAcid, msgCatalogIncomplete} {
		require.True(t, i18n.Registered(k), "key %q is not registered", k)
		for _, l := range []i18n.Locale{i18n.Uzbek, i18n.Russian, i18n.English} {
			require.NotEqual(t, string(k), i18n.T(l, k), "key %q has no %s text", k, l)
		}
	}
	require.Empty(t, i18n.Missing())
}
