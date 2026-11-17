package service

import (
	"context"
	"math"
	"sort"
	"testing"

	"dehqon-bobo/internal/modules/catalog/domain"

	"github.com/stretchr/testify/require"
)

// ---------- fakeRepo: ekinlar ----------

func (r *fakeRepo) ListCrops(context.Context) ([]domain.Crop, error) { return nil, nil }
func (r *fakeRepo) FindCrop(_ context.Context, code string) (*domain.Crop, error) {
	c, ok := r.crops[code]
	if !ok {
		return nil, domain.ErrCropNotFound
	}
	return &c, nil
}
func (r *fakeRepo) ListCropProfiles(_ context.Context, cropID int32) ([]domain.CropProfile, error) {
	var out []domain.CropProfile
	for _, p := range r.profiles {
		if p.CropID == cropID && p.Status != domain.ProfileArchived {
			out = append(out, p)
		}
	}
	// Repo tartibi: bosqich, tasdiqlangani birinchi, yangisi birinchi.
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Stage != b.Stage {
			return a.Stage < b.Stage
		}
		if a.Status != b.Status {
			return a.Status > b.Status
		}
		return a.ID > b.ID
	})
	return out, nil
}
func (r *fakeRepo) FindCropProfile(_ context.Context, id int64) (*domain.CropProfile, error) {
	p, ok := r.profiles[id]
	if !ok {
		return nil, domain.ErrCropProfileNotFound
	}
	return &p, nil
}
func (r *fakeRepo) CreateCropProfile(_ context.Context, p *domain.CropProfile) error {
	r.nextID++
	p.ID = r.nextID
	r.profiles[p.ID] = *p
	return nil
}
func (r *fakeRepo) UpdateCropProfile(_ context.Context, p *domain.CropProfile) error {
	r.profiles[p.ID] = *p
	return nil
}
func (r *fakeRepo) ArchiveApproved(_ context.Context, cropID int32, stage domain.Stage, now int64) error {
	for id, p := range r.profiles {
		if p.CropID == cropID && p.Stage == stage && p.Status == domain.ProfileApproved {
			p.Status, p.UpdatedAt = domain.ProfileArchived, now
			r.profiles[id] = p
		}
	}
	return nil
}
func (r *fakeRepo) MarkApproved(_ context.Context, id, reviewer, now int64) error {
	p := r.profiles[id]
	p.Status, p.ReviewedBy, p.ReviewedAt = domain.ProfileApproved, &reviewer, &now
	r.profiles[id] = p
	return nil
}

// Sonneveld pomidor standarti, mg/L (migratsiya 000012 bilan bir xil).
func tomatoInput() ProfileInput {
	return ProfileInput{
		Stage: domain.StageFruiting,
		Targets: domain.Targets{
			NH4N: 16.808, NO3N: 224.107, P: 46.461, K: 371.434, Ca: 216.421, Mg: 58.332,
			Fe: 0.8377, Mn: 0.5494, B: 0.3243, Zn: 0.3269, Cu: 0.0477, Mo: 0.048,
		},
		ECTarget: f64(2.6), PHMin: 5.5, PHMax: 6.5,
	}
}

func TestCreateCropProfile_SulfurFromIonBalance(t *testing.T) {
	s, _ := newSvc()
	p, err := s.CreateCropProfile(context.Background(), "tomato", tomatoInput())
	require.NoError(t, err)
	require.Equal(t, domain.ProfileDraft, p.Status)
	// SO4 4.4 mmol/L × 32.065 = 141.086 mg/L — manba bilan mos.
	require.InDelta(t, 141.086, p.Targets.S, 0.05)
}

func TestCreateCropProfile_Validation(t *testing.T) {
	s, _ := newSvc()
	ctx := context.Background()

	_, err := s.CreateCropProfile(ctx, "rice", tomatoInput())
	require.ErrorIs(t, err, domain.ErrCropNotFound)

	in := tomatoInput()
	in.Stage = 4
	_, err = s.CreateCropProfile(ctx, "tomato", in)
	require.ErrorIs(t, err, domain.ErrInvalidStage)

	in = tomatoInput()
	in.Targets.NO3N = 600 // 42.8 meq NO3 > 26.3 meq kation → SO4 < 0
	_, err = s.CreateCropProfile(ctx, "tomato", in)
	require.ErrorIs(t, err, domain.ErrIonImbalance)

	for _, mut := range []func(*ProfileInput){
		func(i *ProfileInput) { i.Targets.Ca = -1 },
		func(i *ProfileInput) { i.Targets.Fe = 50 },
		func(i *ProfileInput) { i.Targets.NO3N = math.NaN() },
		func(i *ProfileInput) { i.PHMin, i.PHMax = 6.5, 5.5 },
		func(i *ProfileInput) { i.ECTarget = f64(12) },
	} {
		in := tomatoInput()
		mut(&in)
		_, err := s.CreateCropProfile(ctx, "tomato", in)
		require.ErrorIs(t, err, domain.ErrInvalidValue)
	}
}

func TestApproveCropProfile_ArchivesPrevious(t *testing.T) {
	s, r := newSvc()
	ctx := context.Background()

	first, err := s.CreateCropProfile(ctx, "tomato", tomatoInput())
	require.NoError(t, err)
	_, err = s.ApproveCropProfile(ctx, agronomist, first.ID)
	require.NoError(t, err)

	second, err := s.CreateCropProfile(ctx, "tomato", tomatoInput())
	require.NoError(t, err)
	got, err := s.ApproveCropProfile(ctx, agronomist, second.ID)
	require.NoError(t, err)
	require.Equal(t, domain.ProfileApproved, got.Status)
	require.Equal(t, agronomist, *got.ReviewedBy)

	require.Equal(t, domain.ProfileArchived, r.profiles[first.ID].Status)
	require.Equal(t, domain.ProfileApproved, r.profiles[second.ID].Status)

	// Tasdiqlangan profil o'zgarmaydi va qayta tasdiqlanmaydi.
	_, err = s.UpdateCropProfile(ctx, second.ID, tomatoInput())
	require.ErrorIs(t, err, domain.ErrProfileLocked)
	_, err = s.ApproveCropProfile(ctx, agronomist, second.ID)
	require.ErrorIs(t, err, domain.ErrProfileLocked)
}

func TestUpdateCropProfile_Draft(t *testing.T) {
	s, r := newSvc()
	ctx := context.Background()
	p, err := s.CreateCropProfile(ctx, "tomato", tomatoInput())
	require.NoError(t, err)

	in := tomatoInput()
	in.Targets.K = 300
	got, err := s.UpdateCropProfile(ctx, p.ID, in)
	require.NoError(t, err)
	require.Equal(t, 300.0, r.profiles[p.ID].Targets.K)
	require.Less(t, got.Targets.S, p.Targets.S, "kam K → kam SO4 kerak")

	_, err = s.UpdateCropProfile(ctx, 999, in)
	require.ErrorIs(t, err, domain.ErrCropProfileNotFound)
}

func TestCropProfile_PrefersApproved(t *testing.T) {
	s, r := newSvc()
	ctx := context.Background()

	_, _, err := s.CropProfile(ctx, "tomato", domain.StageFruiting)
	require.ErrorIs(t, err, domain.ErrCropProfileNotFound)

	draft, err := s.CreateCropProfile(ctx, "tomato", tomatoInput())
	require.NoError(t, err)
	_, got, err := s.CropProfile(ctx, "tomato", domain.StageFruiting)
	require.NoError(t, err)
	require.Equal(t, draft.ID, got.ID, "faqat draft bo'lsa — draft")

	_, err = s.ApproveCropProfile(ctx, agronomist, draft.ID)
	require.NoError(t, err)
	_, err = s.CreateCropProfile(ctx, "tomato", tomatoInput()) // yangi draft
	require.Len(t, r.profiles, 2)
	require.NoError(t, err)
	_, got, err = s.CropProfile(ctx, "tomato", domain.StageFruiting)
	require.NoError(t, err)
	require.Equal(t, draft.ID, got.ID, "tasdiqlangani draftdan oldin")

	_, _, err = s.CropProfile(ctx, "rice", domain.StageFruiting)
	require.ErrorIs(t, err, domain.ErrCropNotFound)
}
