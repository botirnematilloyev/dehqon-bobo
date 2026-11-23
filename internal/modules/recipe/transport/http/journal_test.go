package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dehqon-bobo/internal/modules/recipe/domain"
	"dehqon-bobo/internal/modules/recipe/service"
	"dehqon-bobo/internal/platform/i18n"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type fakeJournal struct {
	err      error
	saved    service.SaveInput
	listIn   service.ListInput
	more     bool
	batchIDs []int64
}

func batch(id int64) *domain.Batch {
	g, ml := 255.0, 31.2
	return &domain.Batch{
		ID: id, FarmID: 1, CropCode: "tomato", Stage: domain.StageFruiting, Method: domain.MethodDirect,
		VolumeL: 200, TargetEC: 2.71, Status: domain.BatchPlanned, CalcVersion: "v", CreatedAt: 100,
		Snapshot: json.RawMessage(`{"volume_l":200}`),
		Items: []domain.BatchItem{
			{FertilizerCode: "CaNO3", Tank: domain.TankSingle, AmountG: &g},
			{FertilizerCode: "HNO3_38", Tank: domain.TankSingle, AmountML: &ml, Position: 1},
		},
	}
}

func (f *fakeJournal) Save(_ context.Context, _, _ int64, in service.SaveInput) (*domain.Batch, error) {
	f.saved = in
	if f.err != nil {
		return nil, f.err
	}
	return batch(5), nil
}

func (f *fakeJournal) List(_ context.Context, _, _ int64, in service.ListInput) ([]domain.Batch, bool, error) {
	f.listIn = in
	var out []domain.Batch
	for _, id := range f.batchIDs {
		out = append(out, *batch(id))
	}
	return out, f.more, f.err
}

func (f *fakeJournal) Get(context.Context, int64, int64, int64) (*domain.Batch, error) {
	if f.err != nil {
		return nil, f.err
	}
	return batch(5), nil
}

func (f *fakeJournal) MarkMixed(context.Context, int64, int64, int64) (*domain.Batch, error) {
	if f.err != nil {
		return nil, f.err
	}
	b := batch(5)
	at := int64(200)
	b.Status, b.MixedAt = domain.BatchMixed, &at
	return b, nil
}

func (f *fakeJournal) Delete(context.Context, int64, int64, int64) error { return f.err }

func journalRouter(j Journal) http.Handler {
	h := NewJournalHandler(j, func(context.Context) (int64, bool) { return 1, true })
	r := chi.NewRouter()
	r.Use(i18n.Middleware)
	r.Post("/farms/{farmID}/recipes", h.Save)
	r.Get("/farms/{farmID}/recipes", h.List)
	r.Get("/farms/{farmID}/recipes/{id}", h.Get)
	r.Post("/farms/{farmID}/recipes/{id}/mixed", h.MarkMixed)
	r.Delete("/farms/{farmID}/recipes/{id}", h.Delete)
	return r
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestJournal_SaveAndGet(t *testing.T) {
	f := &fakeJournal{}
	h := journalRouter(f)

	rec := do(h, http.MethodPost, "/farms/1/recipes", `{"crop":"tomato","stage":"fruiting","volume_l":200,"plot_id":11,"acids":["HNO3_38"]}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	require.Equal(t, int64(11), *f.saved.PlotID)
	require.Equal(t, domain.StageFruiting, f.saved.Stage)

	var d BatchDetail
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &d))
	require.Equal(t, "planned", d.Status)
	require.Equal(t, "fruiting", d.Stage)
	require.Equal(t, "direct", d.Method)
	require.JSONEq(t, `{"volume_l":200}`, string(d.Recipe))
	require.Len(t, d.Items, 2)
	require.Equal(t, "single", d.Items[0].Tank)
	require.Equal(t, 255.0, *d.Items[0].Grams)
	require.Nil(t, d.Items[0].ML)
	require.Equal(t, 31.2, *d.Items[1].ML)

	rec = do(h, http.MethodPost, "/farms/1/recipes/5/mixed", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &d))
	require.Equal(t, "mixed", d.Status)
	require.Equal(t, int64(200), *d.MixedAt)

	require.Equal(t, http.StatusNoContent, do(h, http.MethodDelete, "/farms/1/recipes/5", "").Code)
}

func TestJournal_List(t *testing.T) {
	f := &fakeJournal{batchIDs: []int64{9, 8}, more: true}
	rec := do(journalRouter(f), http.MethodGet, "/farms/1/recipes?limit=2&before_id=10&plot_id=3", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.ListInput{Limit: 2, BeforeID: 10, PlotID: ptr(int64(3))}, f.listIn)

	var res BatchListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	require.Len(t, res.Recipes, 2)
	require.Equal(t, int64(8), *res.NextBeforeID)

	f = &fakeJournal{batchIDs: []int64{1}}
	rec = do(journalRouter(f), http.MethodGet, "/farms/1/recipes", "")
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	require.Nil(t, res.NextBeforeID)
	require.Contains(t, rec.Body.String(), `"next_before_id":null`)
}

func ptr[T any](v T) *T { return &v }

func TestJournal_Errors(t *testing.T) {
	cases := []struct {
		name, method, path, body string
		err                      error
		status                   int
		code                     string
	}{
		{"bad limit", http.MethodGet, "/farms/1/recipes?limit=-1", "", nil, 400, "common.invalid_query"},
		{"bad id", http.MethodGet, "/farms/1/recipes/x", "", nil, 400, "common.invalid_id"},
		{"bad json", http.MethodPost, "/farms/1/recipes", "{", nil, 400, "common.invalid_json"},
		{"bad stage", http.MethodPost, "/farms/1/recipes", `{"stage":"x"}`, nil, 400, "recipe.invalid_stage"},
		{"not found", http.MethodGet, "/farms/1/recipes/5", "", domain.ErrBatchNotFound, 404, "recipe.not_found"},
		{"plot", http.MethodPost, "/farms/1/recipes", `{"crop":"tomato","stage":"fruiting"}`, domain.ErrPlotNotFound, 404, "recipe.plot_not_found"},
		{"infeasible", http.MethodPost, "/farms/1/recipes", `{"crop":"tomato","stage":"fruiting"}`, domain.ErrInfeasible, 422, "recipe.infeasible"},
		{"mixed", http.MethodDelete, "/farms/1/recipes/5", "", domain.ErrAlreadyMixed, 409, "recipe.already_mixed"},
		{"farm", http.MethodGet, "/farms/1/recipes", "", domain.ErrFarmNotFound, 404, "recipe.farm_not_found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(journalRouter(&fakeJournal{err: tc.err}), tc.method, tc.path, tc.body)
			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			var e i18n.ErrorBody
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &e))
			require.Equal(t, tc.code, e.Error)
		})
	}
}
