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
			i18n.Uzbek: "O‘g‘it topilmadi", i18n.Russian: "Удобрение не найдено",
		},
		msgParameterNotFound: {
			i18n.Uzbek: "Parametr topilmadi", i18n.Russian: "Параметр не найден",
		},
		msgWaterDefaultNotFound: {
			i18n.Uzbek: "Bu suv manbai uchun qiymat yo‘q", i18n.Russian: "Для этого источника воды нет значений",
		},
		msgInvalidElements: {
			i18n.Uzbek: "O‘g‘it tarkibi noto‘g‘ri: noma’lum element yoki foizlar yig‘indisi 100 dan oshgan", i18n.Russian: "Неверный состав удобрения: неизвестный элемент или сумма процентов больше 100",
		},
		msgOutOfRange: {
			i18n.Uzbek: "Qiymat ruxsat etilgan oraliqdan tashqarida", i18n.Russian: "Значение вне допустимого диапазона",
		},
		msgInvalidValue: {
			i18n.Uzbek: "Qiymat noto‘g‘ri", i18n.Russian: "Неверное значение",
		},
		msgInvalidWaterSource: {
			i18n.Uzbek: "Suv manbai noto‘g‘ri", i18n.Russian: "Неверный источник воды",
		},
		msgInvalidRegion: {
			i18n.Uzbek: "Viloyat noto‘g‘ri", i18n.Russian: "Неверная область",
		},
		msgValueRequired: {
			i18n.Uzbek: "value maydoni majburiy", i18n.Russian: "Поле value обязательно",
		},
		msgCropNotFound: {
			i18n.Uzbek: "Ekin topilmadi", i18n.Russian: "Культура не найдена",
		},
		msgCropProfileNotFound: {
			i18n.Uzbek: "Ekin profili topilmadi", i18n.Russian: "Профиль культуры не найден",
		},
		msgInvalidStage: {
			i18n.Uzbek: "Bosqich noto‘g‘ri: seedling, vegetative yoki fruiting", i18n.Russian: "Неверная фаза: seedling, vegetative или fruiting",
		},
		msgIonImbalance: {
			i18n.Uzbek: "Anionlar (NO3, P, Cl) kationlardan ko‘p — bunday eritmani tuzlardan yig‘ib bo‘lmaydi. K, Ca yoki Mg ni oshiring yoki NO3 ni kamaytiring", i18n.Russian: "Анионов (NO3, P, Cl) больше, чем катионов — такой раствор нельзя собрать из солей. Увеличьте K, Ca или Mg либо уменьшите NO3",
		},
		msgProfileLocked: {
			i18n.Uzbek: "Tasdiqlangan profil o‘zgartirilmaydi. Yangi qoralama yarating", i18n.Russian: "Утверждённый профиль нельзя изменить. Создайте новый черновик",
		},
	})
}
