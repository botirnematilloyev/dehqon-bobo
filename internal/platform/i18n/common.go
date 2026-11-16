package i18n

import "net/http"

// Barcha modullar uchun umumiy kalitlar.
const (
	ErrInternal           Key = "common.internal_error"
	ErrInvalidJSON        Key = "common.invalid_json"
	ErrInvalidID          Key = "common.invalid_id"
	ErrInvalidQuery       Key = "common.invalid_query"
	ErrUnauthorized       Key = "common.unauthorized"
	ErrForbidden          Key = "common.forbidden"
	ErrNotFound           Key = "common.not_found"
	ErrMethodNotAllowed   Key = "common.method_not_allowed"
	ErrTooManyRequests    Key = "common.too_many_requests"
	ErrServiceUnavailable Key = "common.service_unavailable"
)

func init() {
	Register(map[Key]Messages{
		ErrInternal:           {Uzbek: "Serverda xatolik yuz berdi. Birozdan keyin qayta urinib ko‘ring", Russian: "На сервере произошла ошибка. Попробуйте позже", English: "Internal server error"},
		ErrInvalidJSON:        {Uzbek: "So‘rov ma’lumotlari noto‘g‘ri formatda", Russian: "Неверный формат данных запроса", English: "Invalid JSON body"},
		ErrInvalidID:          {Uzbek: "Identifikator noto‘g‘ri", Russian: "Неверный идентификатор", English: "Invalid id"},
		ErrInvalidQuery:       {Uzbek: "So‘rov parametrlari noto‘g‘ri", Russian: "Неверные параметры запроса", English: "Invalid query parameters"},
		ErrUnauthorized:       {Uzbek: "Tizimga kiring", Russian: "Необходимо войти в систему", English: "Unauthorized"},
		ErrForbidden:          {Uzbek: "Bu amalga ruxsatingiz yo‘q", Russian: "Недостаточно прав для этого действия", English: "Forbidden"},
		ErrNotFound:           {Uzbek: "So‘ralgan manzil topilmadi", Russian: "Запрошенный адрес не найден", English: "Not found"},
		ErrMethodNotAllowed:   {Uzbek: "Bu manzil uchun bunday so‘rov turi qo‘llanmaydi", Russian: "Метод не поддерживается для этого адреса", English: "Method not allowed"},
		ErrTooManyRequests:    {Uzbek: "So‘rovlar juda ko‘p. Birozdan keyin urinib ko‘ring", Russian: "Слишком много запросов. Попробуйте позже", English: "Too many requests"},
		ErrServiceUnavailable: {Uzbek: "Xizmat vaqtincha ishlamayapti", Russian: "Сервис временно недоступен", English: "Service unavailable"},
	})
}

// NotFoundHandler va MethodNotAllowedHandler — chi router uchun.
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	WriteError(w, r, http.StatusNotFound, ErrNotFound)
}

func MethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	WriteError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
}
