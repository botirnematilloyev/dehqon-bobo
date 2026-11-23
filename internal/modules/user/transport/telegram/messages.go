package telegram

import (
	"dehqon-bobo/internal/platform/i18n"
	tgapi "dehqon-bobo/internal/platform/telegram"
)

// Bot matnlari foydalanuvchining Telegram tilida (language_code). Matnlarda
// %s kabi o'rin yo'q: uz-Cyrl transliteratsiyasi uni buzardi — kod alohida
// qo'shiladi.
const (
	msgStart       i18n.Key = "bot.start"
	msgUnknown     i18n.Key = "bot.unknown_command"
	msgTooMany     i18n.Key = "bot.too_many_attempts"
	msgError       i18n.Key = "bot.error"
	msgExpired     i18n.Key = "bot.code_expired"
	msgCodeLabel   i18n.Key = "bot.code_label"
	msgCodeValid   i18n.Key = "bot.code_valid_for"
	msgButtonLogin i18n.Key = "bot.button_login"
	msgButtonRenew i18n.Key = "bot.button_renew"

	msgStartApp   i18n.Key = "bot.start_app"
	msgButtonOpen i18n.Key = "bot.button_open_app"
	msgCmdStart   i18n.Key = "bot.command_start"
	msgCmdLogin   i18n.Key = "bot.command_login"
	msgMenuButton i18n.Key = "bot.menu_button"
)

func init() {
	i18n.Register(map[i18n.Key]i18n.Messages{
		msgStart: {
			i18n.Uzbek: "Assalomu alaykum! Dehqon Bobo ilovasiga kirish uchun /login yuboring.", i18n.Russian: "Здравствуйте! Чтобы войти в приложение Dehqon Bobo, отправьте /login.",
		},
		msgUnknown: {
			i18n.Uzbek: "Noma’lum buyruq. /login yuboring.", i18n.Russian: "Неизвестная команда. Отправьте /login.",
		},
		msgTooMany: {
			i18n.Uzbek: "🔒 Juda ko‘p urinish. Bir ozdan keyin qayta urinib ko‘ring.", i18n.Russian: "🔒 Слишком много попыток. Попробуйте чуть позже.",
		},
		msgError: {
			i18n.Uzbek: "Xatolik yuz berdi. Keyinroq urinib ko‘ring.", i18n.Russian: "Произошла ошибка. Попробуйте позже.",
		},
		msgExpired: {
			i18n.Uzbek: "🔒 Kod muddati tugadi. Yangi kod olish uchun tugmani bosing.", i18n.Russian: "🔒 Срок действия кода истёк. Нажмите кнопку, чтобы получить новый.",
		},
		msgCodeLabel: {
			i18n.Uzbek: "Kod", i18n.Russian: "Код",
		},
		msgCodeValid: {
			i18n.Uzbek: "Kod 2 daqiqa amal qiladi.", i18n.Russian: "Код действует 2 минуты.",
		},
		msgButtonLogin: {
			i18n.Uzbek: "Kirish ↗", i18n.Russian: "Войти ↗",
		},
		msgStartApp: {
			i18n.Uzbek: "Assalomu alaykum! Ilovani ochish uchun tugmani bosing.", i18n.Russian: "Здравствуйте! Нажмите кнопку, чтобы открыть приложение.",
		},
		msgButtonOpen: {
			i18n.Uzbek: "🌱 Ilovani ochish", i18n.Russian: "🌱 Открыть приложение",
		},
		msgCmdStart: {
			i18n.Uzbek: "Ilovani ochish", i18n.Russian: "Открыть приложение",
		},
		msgCmdLogin: {
			i18n.Uzbek: "Brauzer uchun kirish kodi", i18n.Russian: "Код входа для браузера",
		},
		msgMenuButton: {
			i18n.Uzbek: "Dehqon Bobo", i18n.Russian: "Dehqon Bobo",
		},
		msgButtonRenew: {
			i18n.Uzbek: "🔄 Yangilash", i18n.Russian: "🔄 Обновить",
		},
	})
}

// localeOf — Telegram ilovasi tili; noma'lum yoki bo'sh bo'lsa o'zbekcha.
func localeOf(u *tgapi.User) i18n.Locale {
	if u == nil {
		return i18n.Default
	}
	return i18n.Negotiate(u.LanguageCode)
}
