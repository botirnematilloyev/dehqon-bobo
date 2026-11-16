package http

import "dehqon-bobo/internal/platform/i18n"

const (
	msgFertilizerNotFound   i18n.Key = "catalog.fertilizer_not_found"
	msgParameterNotFound    i18n.Key = "catalog.parameter_not_found"
	msgWaterDefaultNotFound i18n.Key = "catalog.water_source_not_found"
	msgInvalidElements      i18n.Key = "catalog.invalid_elements"
	msgOutOfRange           i18n.Key = "catalog.value_out_of_range"
	msgInvalidValue         i18n.Key = "catalog.invalid_value"
	msgInvalidWaterSource   i18n.Key = "catalog.invalid_water_source"
	msgInvalidRegion        i18n.Key = "catalog.invalid_region"
	msgValueRequired        i18n.Key = "catalog.value_required"

	msgCropNotFound        i18n.Key = "catalog.crop_not_found"
	msgCropProfileNotFound i18n.Key = "catalog.crop_profile_not_found"
	msgInvalidStage        i18n.Key = "catalog.invalid_stage"
	msgIonImbalance        i18n.Key = "catalog.ion_imbalance"
	msgProfileLocked       i18n.Key = "catalog.crop_profile_locked"
)

func init() {
	i18n.Register(map[i18n.Key]i18n.Messages{
		msgFertilizerNotFound: {
			i18n.Uzbek: "O‘g‘it topilmadi", i18n.Russian: "Удобрение не найдено", i18n.English: "Fertilizer not found",
		},
		msgParameterNotFound: {
			i18n.Uzbek: "Parametr topilmadi", i18n.Russian: "Параметр не найден", i18n.English: "Parameter not found",
		},
		msgWaterDefaultNotFound: {
			i18n.Uzbek: "Bu suv manbai uchun qiymat yo‘q", i18n.Russian: "Для этого источника воды нет значений", i18n.English: "No values for this water source",
		},
		msgInvalidElements: {
			i18n.Uzbek: "O‘g‘it tarkibi noto‘g‘ri: noma’lum element yoki foizlar yig‘indisi 100 dan oshgan", i18n.Russian: "Неверный состав удобрения: неизвестный элемент или сумма процентов больше 100", i18n.English: "Invalid elements: unknown element, percent outside (0, 100] or total over 100",
		},
		msgOutOfRange: {
			i18n.Uzbek: "Qiymat ruxsat etilgan oraliqdan tashqarida", i18n.Russian: "Значение вне допустимого диапазона", i18n.English: "Value outside the allowed range",
		},
		msgInvalidValue: {
			i18n.Uzbek: "Qiymat noto‘g‘ri", i18n.Russian: "Неверное значение", i18n.English: "Invalid value",
		},
		msgInvalidWaterSource: {
			i18n.Uzbek: "Suv manbai noto‘g‘ri", i18n.Russian: "Неверный источник воды", i18n.English: "source must be well, canal, tap or filter",
		},
		msgInvalidRegion: {
			i18n.Uzbek: "Viloyat noto‘g‘ri", i18n.Russian: "Неверная область", i18n.English: "Invalid region_id",
		},
		msgValueRequired: {
			i18n.Uzbek: "value maydoni majburiy", i18n.Russian: "Поле value обязательно", i18n.English: "value is required",
		},
		msgCropNotFound: {
			i18n.Uzbek: "Ekin topilmadi", i18n.Russian: "Культура не найдена", i18n.English: "Crop not found",
		},
		msgCropProfileNotFound: {
			i18n.Uzbek: "Ekin profili topilmadi", i18n.Russian: "Профиль культуры не найден", i18n.English: "Crop profile not found",
		},
		msgInvalidStage: {
			i18n.Uzbek: "Bosqich noto‘g‘ri: seedling, vegetative yoki fruiting", i18n.Russian: "Неверная фаза: seedling, vegetative или fruiting", i18n.English: "Stage must be seedling, vegetative or fruiting",
		},
		msgIonImbalance: {
			i18n.Uzbek: "Anionlar (NO3, P, Cl) kationlardan ko‘p — bunday eritmani tuzlardan yig‘ib bo‘lmaydi. K, Ca yoki Mg ni oshiring yoki NO3 ni kamaytiring", i18n.Russian: "Анионов (NO3, P, Cl) больше, чем катионов — такой раствор нельзя собрать из солей. Увеличьте K, Ca или Mg либо уменьшите NO3", i18n.English: "Anions (NO3, P, Cl) exceed cations, so the solution cannot be made from salts. Raise K, Ca or Mg, or lower NO3",
		},
		msgProfileLocked: {
			i18n.Uzbek: "Tasdiqlangan profil o‘zgartirilmaydi. Yangi qoralama yarating", i18n.Russian: "Утверждённый профиль нельзя изменить. Создайте новый черновик", i18n.English: "Only draft profiles can be edited or approved. Create a new draft",
		},
	})
}
