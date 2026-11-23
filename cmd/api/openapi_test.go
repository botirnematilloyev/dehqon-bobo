package main

import (
	"context"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"

	"dehqon-bobo/internal/modules/catalog"
	"dehqon-bobo/internal/modules/farm"
	"dehqon-bobo/internal/modules/recipe"
	"dehqon-bobo/internal/modules/user"
	"dehqon-bobo/internal/platform/config"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestOpenAPIMatchesRouter — docs/openapi.yaml dagi har bir yo'l+metod
// routerda bor va aksincha. Yangi endpoint qo'shilib, spec unutilsa yiqiladi.
func TestOpenAPIMatchesRouter(t *testing.T) {
	pass := func(next http.Handler) http.Handler { return next }
	uid := func(context.Context) (int64, bool) { return 1, true }

	userModule := user.New(nil, nil, config.JWTConfig{Secret: "test-secret-at-least-32-characters!!"},
		config.NotifierConfig{}, config.TelegramConfig{})
	catalogModule := catalog.New(nil, catalog.Deps{Auth: pass, RequireAgronomist: pass, UserIDFrom: uid})
	farmModule := farm.New(nil, farm.Deps{Auth: pass, UserIDFrom: uid})
	recipeModule := recipe.New(nil, recipe.Deps{Auth: pass, UserIDFrom: uid})

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		mountAPI(r, userModule, farmModule, catalogModule, recipeModule)
	})

	routes := map[string]bool{}
	require.NoError(t, chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		route = strings.TrimPrefix(route, "/api/v1")
		route = strings.ReplaceAll(route, "/*", "")
		if len(route) > 1 {
			route = strings.TrimSuffix(route, "/")
		}
		routes[method+" "+route] = true
		return nil
	}))

	raw, err := os.ReadFile("../../docs/openapi.yaml")
	require.NoError(t, err)
	var doc struct {
		Paths      map[string]map[string]any `yaml:"paths"`
		Components map[string]map[string]any `yaml:"components"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &doc))

	spec := map[string]bool{}
	for path, ops := range doc.Paths {
		for method := range ops {
			if method == "parameters" {
				continue
			}
			spec[strings.ToUpper(method)+" "+path] = true
		}
	}

	require.Len(t, routes, len(spec), "marshrutlar soni")
	require.Greater(t, len(routes), 50, "router bo'sh emas (hozir 58 ta)")

	var missingInSpec, missingInRouter []string
	for k := range routes {
		if !spec[k] {
			missingInSpec = append(missingInSpec, k)
		}
	}
	for k := range spec {
		if !routes[k] {
			missingInRouter = append(missingInRouter, k)
		}
	}
	sort.Strings(missingInSpec)
	sort.Strings(missingInRouter)
	require.Empty(t, missingInSpec, "routerda bor, openapi.yaml da yo'q")
	require.Empty(t, missingInRouter, "openapi.yaml da bor, routerda yo'q")

	// Har bir $ref mavjud komponentga ishora qiladi.
	var anyDoc any
	require.NoError(t, yaml.Unmarshal(raw, &anyDoc))
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, val := range x {
				if k == "$ref" {
					ref := val.(string)
					parts := strings.Split(strings.TrimPrefix(ref, "#/components/"), "/")
					require.Len(t, parts, 2, ref)
					_, ok := doc.Components[parts[0]][parts[1]]
					require.True(t, ok, "yo'q komponent: %s", ref)
				}
				walk(val)
			}
		case []any:
			for _, val := range x {
				walk(val)
			}
		}
	}
	walk(anyDoc)
}
