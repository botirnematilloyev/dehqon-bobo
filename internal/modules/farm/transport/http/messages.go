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
			i18n.Uzbek: "Xo‘jalik topilmadi", i18n.Russian: "Хозяйство не найдено", i18n.English: "Farm not found",
		},
		msgPlotNotFound: {
			i18n.Uzbek: "Issiqxona topilmadi", i18n.Russian: "Теплица не найдена", i18n.English: "Plot not found",
		},
		msgWaterAnalysisNotFound: {
			i18n.Uzbek: "Suv tahlili hali kiritilmagan", i18n.Russian: "Анализ воды ещё не добавлен", i18n.English: "No water analysis yet",
		},
		msgInvalidName: {
			i18n.Uzbek: "Nom 1 dan 100 belgigacha bo‘lishi kerak", i18n.Russian: "Название должно содержать от 1 до 100 символов", i18n.English: "Name must be 1-100 characters",
		},
		msgInvalidLocation: {
			i18n.Uzbek: "Viloyat yoki tuman noto‘g‘ri tanlangan", i18n.Russian: "Неверно выбраны область или район", i18n.English: "Invalid region or district",
		},
		msgInvalidSystemType: {
			i18n.Uzbek: "Yetishtirish tizimi noto‘g‘ri", i18n.Russian: "Неверная система выращивания", i18n.English: "system_type must be greenhouse_soil, substrate, hydroponics or open_field",
		},
		msgInvalidMixing: {
			i18n.Uzbek: "Aralashtirish usuli noto‘g‘ri. A/B uchun injektor nisbati va bak hajmini kiriting", i18n.Russian: "Неверный способ смешивания. Для A/B укажите пропорцию инжектора и объём бака", i18n.English: "mixing_method must be direct or ab; ab needs injector_ratio and stock_tank_l",
		},
		msgInvalidWaterSource: {
			i18n.Uzbek: "Suv manbai noto‘g‘ri", i18n.Russian: "Неверный источник воды", i18n.English: "source must be well, canal, tap or filter",
		},
		msgInvalidWaterValue: {
			i18n.Uzbek: "Suv tahlili qiymatlari noto‘g‘ri: EC, Ca, Mg va HCO3 majburiy", i18n.Russian: "Неверные значения анализа воды: EC, Ca, Mg и HCO3 обязательны", i18n.English: "ec, ca, mg and hco3_mmol are required and must be in a realistic range",
		},
		msgInvalidArea: {
			i18n.Uzbek: "Maydon va hajm musbat son bo‘lishi kerak", i18n.Russian: "Площадь и объём должны быть положительными", i18n.English: "Area and volume must be positive",
		},
		msgWaterDefaultMissing: {
			i18n.Uzbek: "Bu suv manbai uchun taxminiy tarkib yo‘q. Laboratoriya tahlilini kiriting", i18n.Russian: "Для этого источника нет типового состава. Введите лабораторный анализ", i18n.English: "No default water composition for this source; enter a lab analysis",
		},
		msgInvalidLimit: {
			i18n.Uzbek: "limit musbat butun son bo‘lishi kerak", i18n.Russian: "limit должен быть положительным целым числом", i18n.English: "limit must be a positive integer",
		},
		msgFertilizerNotFound: {
			i18n.Uzbek: "O‘g‘it topilmadi", i18n.Russian: "Удобрение не найдено", i18n.English: "Fertilizer not found",
		},
		msgParameterNotFound: {
			i18n.Uzbek: "Parametr topilmadi", i18n.Russian: "Параметр не найден", i18n.English: "Parameter not found",
		},
		msgParameterAgronomist: {
			i18n.Uzbek: "Bu parametrni faqat agronom o‘zgartira oladi", i18n.Russian: "Этот параметр может изменить только агроном", i18n.English: "This parameter can only be changed by an agronomist",
		},
		msgInvalidElements: {
			i18n.Uzbek: "O‘g‘it tarkibi noto‘g‘ri: noma’lum element yoki foizlar yig‘indisi 100 dan oshgan", i18n.Russian: "Неверный состав удобрения: неизвестный элемент или сумма процентов больше 100", i18n.English: "Invalid elements: unknown element, percent outside (0, 100] or total over 100",
		},
		msgSettingOutOfRange: {
			i18n.Uzbek: "Qiymat ruxsat etilgan oraliqdan tashqarida", i18n.Russian: "Значение вне допустимого диапазона", i18n.English: "Value outside the allowed range",
		},
		msgValueRequired: {
			i18n.Uzbek: "value maydoni majburiy", i18n.Russian: "Поле value обязательно", i18n.English: "value is required",
		},
	})
}
