package http

import "dehqon-bobo/internal/platform/i18n"

// Farm moduli xabarlari. Kalit — API javobidagi `error` kodi.
const (
	msgFarmNotFound          i18n.Key = "farm.not_found"
	msgPlotNotFound          i18n.Key = "farm.plot_not_found"
	msgWaterAnalysisNotFound i18n.Key = "farm.water_analysis_not_found"
	msgInvalidName           i18n.Key = "farm.invalid_name"
	msgInvalidLocation       i18n.Key = "farm.invalid_location"
	msgInvalidSystemType     i18n.Key = "farm.invalid_system_type"
	msgInvalidMixing         i18n.Key = "farm.invalid_mixing"
	msgInvalidWaterSource    i18n.Key = "farm.invalid_water_source"
	msgInvalidWaterValue     i18n.Key = "farm.invalid_water_value"
	msgInvalidArea           i18n.Key = "farm.invalid_area"
	msgWaterDefaultMissing   i18n.Key = "farm.water_default_missing"
	msgInvalidLimit          i18n.Key = "farm.invalid_limit"
	msgFertilizerNotFound    i18n.Key = "farm.fertilizer_not_found"
	msgParameterNotFound     i18n.Key = "farm.parameter_not_found"
	msgParameterAgronomist   i18n.Key = "farm.parameter_agronomist_only"
	msgInvalidElements       i18n.Key = "farm.invalid_elements"
	msgSettingOutOfRange     i18n.Key = "farm.setting_out_of_range"
	msgValueRequired         i18n.Key = "farm.value_required"
)

func init() {
	i18n.Register(map[i18n.Key]i18n.Messages{
		msgFarmNotFound: {
			i18n.Uzbek: "Xo‘jalik topilmadi", i18n.Russian: "Хозяйство не найдено",
		},
		msgPlotNotFound: {
			i18n.Uzbek: "Issiqxona topilmadi", i18n.Russian: "Теплица не найдена",
		},
		msgWaterAnalysisNotFound: {
			i18n.Uzbek: "Suv tahlili hali kiritilmagan", i18n.Russian: "Анализ воды ещё не добавлен",
		},
		msgInvalidName: {
			i18n.Uzbek: "Nom 1 dan 100 belgigacha bo‘lishi kerak", i18n.Russian: "Название должно содержать от 1 до 100 символов",
		},
		msgInvalidLocation: {
			i18n.Uzbek: "Viloyat yoki tuman noto‘g‘ri tanlangan", i18n.Russian: "Неверно выбраны область или район",
		},
		msgInvalidSystemType: {
			i18n.Uzbek: "Yetishtirish tizimi noto‘g‘ri", i18n.Russian: "Неверная система выращивания",
		},
		msgInvalidMixing: {
			i18n.Uzbek: "Aralashtirish usuli noto‘g‘ri. A/B uchun injektor nisbati va bak hajmini kiriting", i18n.Russian: "Неверный способ смешивания. Для A/B укажите пропорцию инжектора и объём бака",
		},
		msgInvalidWaterSource: {
			i18n.Uzbek: "Suv manbai noto‘g‘ri", i18n.Russian: "Неверный источник воды",
		},
		msgInvalidWaterValue: {
			i18n.Uzbek: "Suv tahlili qiymatlari noto‘g‘ri: EC, Ca, Mg va HCO3 majburiy", i18n.Russian: "Неверные значения анализа воды: EC, Ca, Mg и HCO3 обязательны",
		},
		msgInvalidArea: {
			i18n.Uzbek: "Maydon va hajm musbat son bo‘lishi kerak", i18n.Russian: "Площадь и объём должны быть положительными",
		},
		msgWaterDefaultMissing: {
			i18n.Uzbek: "Bu suv manbai uchun taxminiy tarkib yo‘q. Laboratoriya tahlilini kiriting", i18n.Russian: "Для этого источника нет типового состава. Введите лабораторный анализ",
		},
		msgInvalidLimit: {
			i18n.Uzbek: "limit musbat butun son bo‘lishi kerak", i18n.Russian: "limit должен быть положительным целым числом",
		},
		msgFertilizerNotFound: {
			i18n.Uzbek: "O‘g‘it topilmadi", i18n.Russian: "Удобрение не найдено",
		},
		msgParameterNotFound: {
			i18n.Uzbek: "Parametr topilmadi", i18n.Russian: "Параметр не найден",
		},
		msgParameterAgronomist: {
			i18n.Uzbek: "Bu parametrni faqat agronom o‘zgartira oladi", i18n.Russian: "Этот параметр может изменить только агроном",
		},
		msgInvalidElements: {
			i18n.Uzbek: "O‘g‘it tarkibi noto‘g‘ri: noma’lum element yoki foizlar yig‘indisi 100 dan oshgan", i18n.Russian: "Неверный состав удобрения: неизвестный элемент или сумма процентов больше 100",
		},
		msgSettingOutOfRange: {
			i18n.Uzbek: "Qiymat ruxsat etilgan oraliqdan tashqarida", i18n.Russian: "Значение вне допустимого диапазона",
		},
		msgValueRequired: {
			i18n.Uzbek: "value maydoni majburiy", i18n.Russian: "Поле value обязательно",
		},
	})
}
