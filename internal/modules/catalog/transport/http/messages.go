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
	})
}
