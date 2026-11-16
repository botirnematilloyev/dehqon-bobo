package main

import "context"

// Modullar bir-birini import qilmaydi: har biri o'z portini e'lon qiladi,
// ularni shu yerda bog'laymiz (binary'dagi cmd/api/adapters.go kabi).

// onboarderFunc — user.Module.MarkOnboarded ni farm/service.UserOnboarder
// portiga moslaydi.
type onboarderFunc func(ctx context.Context, userID int64, regionID, districtID *int32, at int64) error

func (f onboarderFunc) MarkOnboarded(ctx context.Context, userID int64, regionID, districtID *int32, at int64) error {
	return f(ctx, userID, regionID, districtID, at)
}
