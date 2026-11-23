package notifier

import "dehqon-bobo/internal/platform/i18n"

// OTP xabarlari so'rov tilida (ctx dagi Accept-Language; bo'lmasa uz).
const (
	msgSubject     i18n.Key = "notify.otp_subject"
	msgYourCode    i18n.Key = "notify.otp_your_code"
	msgIntro       i18n.Key = "notify.otp_intro"
	msgValidFor    i18n.Key = "notify.otp_valid_5_min"
	msgNeverShare  i18n.Key = "notify.otp_never_share"
	msgIgnoreIfNot i18n.Key = "notify.otp_ignore"
)

func init() {
	i18n.Register(map[i18n.Key]i18n.Messages{
		msgSubject: {
			i18n.Uzbek: "Tasdiqlash kodi", i18n.Russian: "Код подтверждения", i18n.English: "Verification code",
		},
		msgYourCode: {
			i18n.Uzbek: "Tasdiqlash kodingiz", i18n.Russian: "Ваш код подтверждения", i18n.English: "Your verification code",
		},
		msgIntro: {
			i18n.Uzbek: "Hisobingizga kirish uchun quyidagi kodni kiriting.", i18n.Russian: "Введите этот код, чтобы войти в аккаунт.", i18n.English: "Enter this code to sign in to your account.",
		},
		msgValidFor: {
			i18n.Uzbek: "Kod 5 daqiqa amal qiladi.", i18n.Russian: "Код действует 5 минут.", i18n.English: "The code is valid for 5 minutes.",
		},
		msgNeverShare: {
			i18n.Uzbek: "Kodni hech kimga bermang — xodimlarimiz ham hech qachon so‘ramaydi.", i18n.Russian: "Никому не сообщайте код — наши сотрудники никогда его не спрашивают.", i18n.English: "Never share the code — our staff will never ask for it.",
		},
		msgIgnoreIfNot: {
			i18n.Uzbek: "Agar bu so‘rovni siz yubormagan bo‘lsangiz, xatni e’tiborsiz qoldiring.", i18n.Russian: "Если вы не запрашивали код, просто проигнорируйте это письмо.", i18n.English: "If you did not request this, ignore this email.",
		},
	})
}
