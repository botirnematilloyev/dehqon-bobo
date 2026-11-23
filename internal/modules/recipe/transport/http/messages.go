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

	msgBatchNotFound i18n.Key = "recipe.not_found"
	msgPlotNotFound  i18n.Key = "recipe.plot_not_found"
	msgInfeasible    i18n.Key = "recipe.infeasible"
	msgAlreadyMixed  i18n.Key = "recipe.already_mixed"

	msgInvalidMeasurement i18n.Key = "recipe.invalid_measurement"
	msgInvalidCorrection  i18n.Key = "recipe.invalid_correction"

	msgSeasonNotFound   i18n.Key = "season.not_found"
	msgSeasonOpenExists i18n.Key = "season.open_exists"
	msgSeasonEnded      i18n.Key = "season.ended"
	msgInvalidSeason    i18n.Key = "season.invalid"
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
		msgBatchNotFound: {
			i18n.Uzbek: "Retsept topilmadi", i18n.Russian: "Рецепт не найден", i18n.English: "Recipe not found",
		},
		msgPlotNotFound: {
			i18n.Uzbek: "Issiqxona topilmadi", i18n.Russian: "Теплица не найдена", i18n.English: "Plot not found in this farm",
		},
		msgInfeasible: {
			i18n.Uzbek: "Bu o‘g‘itlar bilan retseptni yig‘ib bo‘lmaydi (manfiy miqdor). Sozlamalarda boshqa tuzni yoqing yoki agronomga murojaat qiling", i18n.Russian: "С этими удобрениями рецепт не собрать (отрицательная навеска). Включите другую соль в настройках или обратитесь к агроному", i18n.English: "This recipe cannot be made with the selected salts (negative amounts)",
		},
		msgAlreadyMixed: {
			i18n.Uzbek: "Eritma allaqachon tayyorlangan — jurnaldan o‘chirilmaydi", i18n.Russian: "Раствор уже приготовлен — из журнала не удаляется", i18n.English: "The solution is already mixed and stays in the journal",
		},
		msgInvalidMeasurement: {
			i18n.Uzbek: "O‘lchov noto‘g‘ri: EC, pH yoki drenaj EC dan kamida bittasini kiriting (EC 0–20, pH 0–14)", i18n.Russian: "Неверное измерение: укажите EC, pH или EC дренажа (EC 0–20, pH 0–14)", i18n.English: "Invalid measurement: give ec_in, ph_in or ec_drain (EC 0–20, pH 0–14)",
		},
		msgInvalidCorrection: {
			i18n.Uzbek: "Bakdagi EC va hajmni kiriting; yakuniy hajm hozirgidan katta bo‘lsin", i18n.Russian: "Укажите EC и объём в баке; итоговый объём должен быть больше текущего", i18n.English: "Give ec_now and volumes; volume_final_l must exceed volume_now_l",
		},
		msgSeasonNotFound: {
			i18n.Uzbek: "Mavsum topilmadi", i18n.Russian: "Сезон не найден", i18n.English: "Season not found",
		},
		msgSeasonOpenExists: {
			i18n.Uzbek: "Bu issiqxonada ochiq mavsum bor. Avval uni yakunlang", i18n.Russian: "В этой теплице уже идёт сезон. Сначала завершите его", i18n.English: "This plot already has an open season. End it first",
		},
		msgSeasonEnded: {
			i18n.Uzbek: "Mavsum yakunlangan", i18n.Russian: "Сезон уже завершён", i18n.English: "The season is already ended",
		},
		msgInvalidSeason: {
			i18n.Uzbek: "Mavsum ma’lumoti noto‘g‘ri: hosil (kg) kiriting, sana boshlanishidan oldin bo‘lmasin", i18n.Russian: "Неверные данные сезона: укажите урожай (кг), дата не раньше начала", i18n.English: "Invalid season values: give yield_kg; the end date cannot precede the start",
		},
	})
}
