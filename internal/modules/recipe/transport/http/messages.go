package http

import "dehqon-bobo/internal/platform/i18n"

const (
	msgFarmNotFound      i18n.Key = "recipe.farm_not_found"
	msgCropNotFound      i18n.Key = "recipe.crop_not_found"
	msgProfileNotFound   i18n.Key = "recipe.profile_not_found"
	msgInvalidStage      i18n.Key = "recipe.invalid_stage"
	msgInvalidMethod     i18n.Key = "recipe.invalid_method"
	msgWaterRequired     i18n.Key = "recipe.water_analysis_required"
	msgInvalidAcid       i18n.Key = "recipe.invalid_acid"
	msgCatalogIncomplete i18n.Key = "recipe.catalog_incomplete"
)

func init() {
	i18n.Register(map[i18n.Key]i18n.Messages{
		msgFarmNotFound: {
			i18n.Uzbek: "Xo‘jalik topilmadi", i18n.Russian: "Хозяйство не найдено", i18n.English: "Farm not found",
		},
		msgCropNotFound: {
			i18n.Uzbek: "Ekin topilmadi", i18n.Russian: "Культура не найдена", i18n.English: "Crop not found",
		},
		msgProfileNotFound: {
			i18n.Uzbek: "Bu ekin va bosqich uchun retsept hali kiritilmagan", i18n.Russian: "Для этой культуры и фазы рецепт ещё не добавлен", i18n.English: "No recipe profile for this crop and stage yet",
		},
		msgInvalidStage: {
			i18n.Uzbek: "Bosqich noto‘g‘ri: seedling, vegetative yoki fruiting", i18n.Russian: "Неверная фаза: seedling, vegetative или fruiting", i18n.English: "Stage must be seedling, vegetative or fruiting",
		},
		msgInvalidMethod: {
			i18n.Uzbek: "Bak hajmini kiriting. A/B usulida idish hajmi va injektor nisbati kerak", i18n.Russian: "Укажите объём бака. Для A/B нужны объём ёмкости и пропорция инжектора", i18n.English: "Set the tank volume. The A/B method needs the stock tank volume and injector ratio",
		},
		msgWaterRequired: {
			i18n.Uzbek: "Avval suv tahlilini kiriting", i18n.Russian: "Сначала добавьте анализ воды", i18n.English: "Add a water analysis first",
		},
		msgInvalidAcid: {
			i18n.Uzbek: "Kislota noto‘g‘ri yoki foizi va zichligi kiritilmagan", i18n.Russian: "Неверная кислота или не указаны концентрация и плотность", i18n.English: "Unknown acid, or its concentration and density are missing",
		},
		msgCatalogIncomplete: {
			i18n.Uzbek: "O‘g‘itlar yoki hisob sozlamalari to‘liq emas. Agronomga murojaat qiling", i18n.Russian: "Не хватает удобрений или параметров расчёта. Обратитесь к агроному", i18n.English: "A fertilizer or calc parameter is missing from the catalog",
		},
	})
}
