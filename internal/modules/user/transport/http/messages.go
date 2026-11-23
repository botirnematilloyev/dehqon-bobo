package http

import "dehqon-bobo/internal/platform/i18n"

// User/auth moduli xabarlari. Kalit — API javobidagi `error` kodi.
const (
	msgInvalidOTP          i18n.Key = "auth.invalid_otp"
	msgUnsupportedProvider i18n.Key = "auth.unsupported_provider"
	msgInvalidLoginCode    i18n.Key = "auth.invalid_login_code"
	msgSessionInvalid      i18n.Key = "auth.session_invalid"
	msgMissingToken        i18n.Key = "auth.missing_token"
	msgInvalidToken        i18n.Key = "auth.invalid_token"
	msgAccountExists       i18n.Key = "auth.account_exists"
	msgCodeSixDigits       i18n.Key = "auth.code_must_be_6_digits"
	msgIdentifierRequired  i18n.Key = "auth.identifier_required"
	msgRefreshRequired     i18n.Key = "auth.refresh_token_required"
	msgInitDataInvalid     i18n.Key = "auth.invalid_init_data"
	msgInitDataExpired     i18n.Key = "auth.init_data_expired"
	msgInitDataRequired    i18n.Key = "auth.init_data_required"

	msgUserNotFound      i18n.Key = "user.not_found"
	msgUsernameTaken     i18n.Key = "user.username_taken"
	msgUsernameInvalid   i18n.Key = "user.username_invalid"
	msgUsernameReserved  i18n.Key = "user.username_reserved"
	msgLocaleInvalid     i18n.Key = "user.locale_invalid"
	msgLocationInvalid   i18n.Key = "user.location_invalid"
	msgFullNameEmpty     i18n.Key = "user.full_name_empty"
	msgFullNameTooLong   i18n.Key = "user.full_name_too_long"
	msgRegionIDInvalid   i18n.Key = "location.region_id_invalid"
	msgDistrictIDInvalid i18n.Key = "location.district_id_invalid"
)

// validationError — so'rov tekshiruvi xatosi; kalitni o'zida olib yuradi.
type validationError i18n.Key

func (e validationError) Error() string { return string(e) }

// validationKey — Validate() qaytargan xatoning kaliti.
func validationKey(err error) i18n.Key {
	if ve, ok := err.(validationError); ok {
		return i18n.Key(ve)
	}
	return i18n.ErrInvalidQuery
}

func init() {
	i18n.Register(map[i18n.Key]i18n.Messages{
		msgInvalidOTP: {
			i18n.Uzbek: "Kod noto‘g‘ri yoki muddati o‘tgan", i18n.Russian: "Неверный или просроченный код", i18n.English: "Invalid or expired OTP code",
		},
		msgUnsupportedProvider: {
			i18n.Uzbek: "Bu usul bilan kirish qo‘llanmaydi", i18n.Russian: "Этот способ входа не поддерживается", i18n.English: "Unsupported provider",
		},
		msgInvalidLoginCode: {
			i18n.Uzbek: "Telegram kodi noto‘g‘ri yoki muddati o‘tgan. Botdan yangi kod oling", i18n.Russian: "Код из Telegram неверный или устарел. Получите новый код в боте", i18n.English: "Invalid or expired login code",
		},
		msgSessionInvalid: {
			i18n.Uzbek: "Sessiya tugagan. Qaytadan kiring", i18n.Russian: "Сессия истекла. Войдите снова", i18n.English: "Session is no longer valid",
		},
		msgMissingToken: {
			i18n.Uzbek: "Tizimga kiring", i18n.Russian: "Необходимо войти в систему", i18n.English: "Missing or invalid Authorization header",
		},
		msgInvalidToken: {
			i18n.Uzbek: "Kirish muddati tugagan. Qaytadan kiring", i18n.Russian: "Срок входа истёк. Войдите снова", i18n.English: "Invalid token",
		},
		msgAccountExists: {
			i18n.Uzbek: "Bunday akkaunt allaqachon bor", i18n.Russian: "Такой аккаунт уже существует", i18n.English: "Account already exists",
		},
		msgCodeSixDigits: {
			i18n.Uzbek: "Kod 6 ta raqamdan iborat bo‘lishi kerak", i18n.Russian: "Код должен состоять из 6 цифр", i18n.English: "Code must be 6 digits",
		},
		msgIdentifierRequired: {
			i18n.Uzbek: "Telefon, email yoki Telegram ID kiriting", i18n.Russian: "Укажите телефон, email или Telegram ID", i18n.English: "Identifier is required",
		},
		msgRefreshRequired: {
			i18n.Uzbek: "refresh_token majburiy", i18n.Russian: "refresh_token обязателен", i18n.English: "Refresh token is required",
		},
		msgInitDataInvalid: {
			i18n.Uzbek: "Telegram ma’lumoti tasdiqlanmadi. Ilovani bot orqali qayta oching", i18n.Russian: "Данные Telegram не прошли проверку. Откройте приложение через бота заново", i18n.English: "Telegram init data failed verification. Reopen the app from the bot",
		},
		msgInitDataExpired: {
			i18n.Uzbek: "Kirish muddati o‘tgan. Ilovani bot orqali qayta oching", i18n.Russian: "Сессия устарела. Откройте приложение через бота заново", i18n.English: "Init data has expired. Reopen the app from the bot",
		},
		msgInitDataRequired: {
			i18n.Uzbek: "init_data majburiy", i18n.Russian: "init_data обязателен", i18n.English: "init_data is required",
		},
		msgUserNotFound: {
			i18n.Uzbek: "Foydalanuvchi topilmadi", i18n.Russian: "Пользователь не найден", i18n.English: "User not found",
		},
		msgUsernameTaken: {
			i18n.Uzbek: "Bu nom band. Boshqasini tanlang", i18n.Russian: "Это имя занято. Выберите другое", i18n.English: "Username is already taken",
		},
		msgUsernameInvalid: {
			i18n.Uzbek: "Nom 3–32 belgi: kichik lotin harflari, raqamlar, - va _", i18n.Russian: "Имя: 3–32 символа — строчные латинские буквы, цифры, - и _", i18n.English: "Username must be 3-32 characters: lowercase letters, digits, - and _",
		},
		msgUsernameReserved: {
			i18n.Uzbek: "Bu nomni tanlab bo‘lmaydi", i18n.Russian: "Это имя зарезервировано", i18n.English: "This username is reserved",
		},
		msgLocaleInvalid: {
			i18n.Uzbek: "Til noto‘g‘ri: uz, uz_cyr yoki ru", i18n.Russian: "Неверный язык: uz, uz_cyr или ru", i18n.English: "Invalid locale: uz, uz_cyr or ru",
		},
		msgLocationInvalid: {
			i18n.Uzbek: "Viloyat yoki tuman noto‘g‘ri tanlangan", i18n.Russian: "Неверно выбраны область или район", i18n.English: "Invalid region or district",
		},
		msgFullNameEmpty: {
			i18n.Uzbek: "Ism bo‘sh bo‘lmasligi kerak", i18n.Russian: "Имя не может быть пустым", i18n.English: "Full name cannot be empty",
		},
		msgFullNameTooLong: {
			i18n.Uzbek: "Ism juda uzun (100 belgigacha)", i18n.Russian: "Имя слишком длинное (до 100 символов)", i18n.English: "Full name is too long",
		},
		msgRegionIDInvalid: {
			i18n.Uzbek: "Viloyat ID si noto‘g‘ri", i18n.Russian: "Неверный ID области", i18n.English: "region_id must be a positive integer",
		},
		msgDistrictIDInvalid: {
			i18n.Uzbek: "Tuman ID si noto‘g‘ri", i18n.Russian: "Неверный ID района", i18n.English: "district_id must be a positive integer",
		},
	})
}
