package hpg

// hpgDefaultSalts — HPG saltCompositions standarti (testlar uchun; bu paket
// ichki testlari calctest'ni import qila olmaydi — import sikli).
func hpgDefaultSalts() Salts {
	return Salts{
		CaNO3Ca: 16.972, CaNO3NO3: 11.863, CaNO3NH4: 0,
		KNO3K: 38.672, KNO3NO3: 13.854,
		NH4NO3NH4: 17.499, NH4NO3NO3: 17.499,
		MgSO4Mg: 9.861, MgSO4S: 13.01,
		KH2PO4K: 28.731, KH2PO4P: 22.761,
		K2SO4K: 44.874, K2SO4S: 18.401,
		MgNO3Mg: 9.479, MgNO3NO3: 10.925,
		CaCl2Ca: 18.294, CaCl2Cl: 32.366,
	}
}
