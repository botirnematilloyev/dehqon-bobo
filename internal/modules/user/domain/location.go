package domain

type Region struct {
	ID   int32
	Name map[string]string
}

type District struct {
	ID       int32
	RegionID int32
	Name     map[string]string
}
