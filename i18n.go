package main

import (
	"fmt"
	"strings"
)

// Supported interface languages. Russian is the default.
const (
	LangRU = "ru"
	LangEN = "en"
)

// IsLang reports whether the code is a supported language.
func IsLang(code string) bool { return code == LangRU || code == LangEN }

// translations holds every UI string, keyed by language and then by message key.
// Keeping them in one map is enough for two languages and keeps the templates
// free of hard-coded text.
var translations = map[string]map[string]string{
	LangRU: {
		"app.name":       "Траты",
		"nav.overview":   "Обзор",
		"nav.reports":    "Отчёты",
		"nav.categories": "Категории",
		"nav.settings":   "Настройки",

		"action.add_expense": "Добавить расход",
		"action.close":       "Закрыть",
		"action.cancel":      "Отмена",
		"action.delete":      "Удалить",
		"action.save":        "Сохранить",
		"action.back":        "Назад",

		"date.today":     "Сегодня",
		"date.yesterday": "Вчера",

		"overview.today":       "Сегодня, %s",
		"overview.today_empty": "Сегодня трат пока нет",
		"overview.week":        "Эта неделя",
		"overview.week_empty":  "На этой неделе расходов пока нет. Неделя считается с понедельника.",
		"overview.recent":      "Последние расходы",
		"overview.all":         "Все",

		"donut.other": "Остальные",

		"first_run.title":      "Здесь появятся ваши расходы",
		"first_run.text":       "Нажмите «+» и запишите первую трату — категорию можно создать прямо там.",
		"empty.expenses_title": "Расходов пока нет",
		"empty.expenses_text":  "Запишите первый — это займёт несколько секунд.",
		"empty.chart":          "Нет данных для графика",

		"form.new_title":                "Новый расход",
		"form.edit_title":               "Расход",
		"form.amount":                   "Сумма",
		"form.category":                 "Категория",
		"form.category_new":             "Новая",
		"form.category_new_placeholder": "Название новой категории",
		"form.first_category_hint":      "Первая категория создастся вместе с расходом. Дальше её можно выбрать одним касанием.",
		"form.date":                     "Дата",
		"form.date_other":               "Другая дата",
		"form.note":                     "Комментарий",
		"form.optional":                 "— необязательно",
		"form.note_placeholder":         "Например, ужин с друзьями",
		"form.delete_expense":           "Удалить расход",
		"form.errors":                   "Проверьте выделенные поля.",

		"error.amount_empty":   "Введите сумму",
		"error.amount_zero":    "Введите сумму больше нуля",
		"error.amount_shape":   "Сумма — это число, например 12,50",
		"error.amount_big":     "Слишком большая сумма",
		"error.category_empty": "Выберите категорию",
		"error.category_name":  "Введите название категории",
		"error.category_dup":   "Такая категория уже есть",
		"error.date_future":    "Дата не может быть позже сегодняшней",
		"error.date_shape":     "Введите дату",
		"error.range_order":    "Дата окончания раньше даты начала",
		"error.note_long":      "Комментарий слишком длинный",
		"error.name_long":      "Название слишком длинное",

		"confirm.delete_expense_title": "Удалить расход?",
		"confirm.delete_expense_text":  "Вернуть его будет нельзя.",
		"confirm.delete_category":      "Удалить категорию «%s»?",
		"confirm.delete_category_text": "Категория пропадёт из выбора при добавлении расхода.",

		"toast.expense_added":    "Расход добавлен",
		"toast.expense_saved":    "Изменения сохранены",
		"toast.expense_deleted":  "Расход удалён",
		"toast.category_added":   "Категория создана",
		"toast.category_deleted": "Категория удалена",

		"category.title":        "Категория",
		"category.no_note":      "Без комментария",
		"category.period_week":  "Эта неделя",
		"category.period_month": "Этот месяц",
		"category.empty_period": "В этой категории за выбранный период расходов нет",

		"categories.title":           "Категории",
		"categories.new":             "Новая категория",
		"categories.new_placeholder": "Например, Спорт",
		"categories.add":             "Добавить",
		"categories.yours":           "Ваши категории",
		"categories.empty_title":     "Категорий пока нет",
		"categories.empty_text":      "Создайте первую — или прямо в форме расхода, когда будете записывать трату.",
		"categories.in_use":          "Нельзя удалить: категория используется в расходах (%d). Сначала измените или удалите их.",
		"categories.note":            "Удалить можно категорию без расходов. Если расходы есть, сначала измените или удалите их.",

		"reports.title":       "Отчёты",
		"reports.export":      "Экспортировать CSV",
		"reports.week":        "Эта неделя",
		"reports.month":       "Этот месяц",
		"reports.prev_month":  "Прошлый месяц",
		"reports.year":        "Этот год",
		"reports.from":        "С",
		"reports.to":          "По",
		"reports.show":        "Показать",
		"reports.total":       "Всего",
		"reports.count":       "Расходов",
		"reports.by_category": "По категориям",
		"reports.list":        "Расходы",
		"reports.empty_title": "За %s расходов нет",
		"reports.empty_text":  "Выберите другие даты или один из готовых периодов выше.",

		"settings.title":         "Настройки",
		"settings.currency":      "Валюта",
		"settings.currency_note": "Новые и уже записанные расходы показываются в выбранной валюте. Суммы не пересчитываются.",
		"settings.language":      "Язык",
		"settings.theme":         "Тема",
		"settings.theme_light":   "Светлая",
		"settings.theme_dark":    "Тёмная",
		"settings.theme_system":  "Системная",
		"settings.theme_note":    "Системная тема повторяет настройку устройства и меняется вместе с ней.",

		"currency.EUR": "Евро",
		"currency.USD": "Доллар США",
		"currency.RUB": "Российский рубль",
		"currency.GBP": "Фунт стерлингов",

		"csv.date":     "Дата",
		"csv.amount":   "Сумма",
		"csv.currency": "Валюта",
		"csv.category": "Категория",
		"csv.comment":  "Комментарий",

		"error.not_found": "Страница не найдена",
		"error.server":    "Что-то пошло не так. Попробуйте ещё раз.",
		"error.go_home":   "На главную",
	},
	LangEN: {
		"app.name":       "Traty",
		"nav.overview":   "Overview",
		"nav.reports":    "Reports",
		"nav.categories": "Categories",
		"nav.settings":   "Settings",

		"action.add_expense": "Add expense",
		"action.close":       "Close",
		"action.cancel":      "Cancel",
		"action.delete":      "Delete",
		"action.save":        "Save",
		"action.back":        "Back",

		"date.today":     "Today",
		"date.yesterday": "Yesterday",

		"overview.today":       "Today, %s",
		"overview.today_empty": "Nothing spent today",
		"overview.week":        "This week",
		"overview.week_empty":  "No expenses this week yet. Weeks start on Monday.",
		"overview.recent":      "Recent expenses",
		"overview.all":         "See all",

		"donut.other": "Other",

		"first_run.title":      "Your expenses will appear here",
		"first_run.text":       "Tap + to add your first one — you can create a category right there.",
		"empty.expenses_title": "No expenses yet",
		"empty.expenses_text":  "Adding one takes a few seconds.",
		"empty.chart":          "Nothing to chart yet",

		"form.new_title":                "New expense",
		"form.edit_title":               "Expense",
		"form.amount":                   "Amount",
		"form.category":                 "Category",
		"form.category_new":             "New",
		"form.category_new_placeholder": "New category name",
		"form.first_category_hint":      "Your first category will be created with this expense. Next time it’s one tap away.",
		"form.date":                     "Date",
		"form.date_other":               "Other date",
		"form.note":                     "Note",
		"form.optional":                 "— optional",
		"form.note_placeholder":         "e.g. Dinner with friends",
		"form.delete_expense":           "Delete expense",
		"form.errors":                   "Please check the highlighted fields.",

		"error.amount_empty":   "Enter an amount",
		"error.amount_zero":    "Enter an amount greater than zero",
		"error.amount_shape":   "Use a number like 12,50",
		"error.amount_big":     "That amount is too large",
		"error.category_empty": "Choose a category",
		"error.category_name":  "Name the category",
		"error.category_dup":   "That category already exists",
		"error.date_future":    "The date can’t be in the future",
		"error.date_shape":     "Enter a date",
		"error.range_order":    "The end date is before the start date",
		"error.note_long":      "The note is too long",
		"error.name_long":      "The name is too long",

		"confirm.delete_expense_title": "Delete this expense?",
		"confirm.delete_expense_text":  "This can’t be undone.",
		"confirm.delete_category":      "Delete “%s”?",
		"confirm.delete_category_text": "It won’t be available for new expenses.",

		"toast.expense_added":    "Expense added",
		"toast.expense_saved":    "Changes saved",
		"toast.expense_deleted":  "Expense deleted",
		"toast.category_added":   "Category created",
		"toast.category_deleted": "Category deleted",

		"category.title":        "Category",
		"category.no_note":      "No note",
		"category.period_week":  "This week",
		"category.period_month": "This month",
		"category.empty_period": "No expenses in this category for the selected period",

		"categories.title":           "Categories",
		"categories.new":             "New category",
		"categories.new_placeholder": "e.g. Sport",
		"categories.add":             "Add",
		"categories.yours":           "Your categories",
		"categories.empty_title":     "No categories yet",
		"categories.empty_text":      "Create one here — or right in the expense form.",
		"categories.in_use":          "Can’t delete: this category is used by %d expenses. Change or delete them first.",
		"categories.note":            "A category can be deleted once it has no expenses. Change or delete them first.",

		"reports.title":       "Reports",
		"reports.export":      "Export CSV",
		"reports.week":        "This week",
		"reports.month":       "This month",
		"reports.prev_month":  "Last month",
		"reports.year":        "This year",
		"reports.from":        "From",
		"reports.to":          "To",
		"reports.show":        "Show",
		"reports.total":       "Total",
		"reports.count":       "Expenses",
		"reports.by_category": "By category",
		"reports.list":        "Expenses",
		"reports.empty_title": "No expenses for %s",
		"reports.empty_text":  "Pick other dates or one of the periods above.",

		"settings.title":         "Settings",
		"settings.currency":      "Currency",
		"settings.currency_note": "New and existing expenses are shown in this currency. Amounts are not converted.",
		"settings.language":      "Language",
		"settings.theme":         "Theme",
		"settings.theme_light":   "Light",
		"settings.theme_dark":    "Dark",
		"settings.theme_system":  "System",
		"settings.theme_note":    "System follows your device’s setting.",

		"currency.EUR": "Euro",
		"currency.USD": "US dollar",
		"currency.RUB": "Russian rouble",
		"currency.GBP": "Pound sterling",

		"csv.date":     "Date",
		"csv.amount":   "Amount",
		"csv.currency": "Currency",
		"csv.category": "Category",
		"csv.comment":  "Note",

		"error.not_found": "Page not found",
		"error.server":    "Something went wrong. Please try again.",
		"error.go_home":   "Go to overview",
	},
}

// T returns the translated string for a key, falling back to Russian and then
// to the key itself so a missing translation is visible but never fatal.
func T(lang, key string) string {
	if m, ok := translations[lang]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	if s, ok := translations[LangRU][key]; ok {
		return s
	}
	return key
}

// Tf formats a translated string that contains verbs like %s or %d.
func Tf(lang, key string, args ...any) string {
	return fmt.Sprintf(T(lang, key), args...)
}

// PluralExpenses renders "3 расхода" / "3 expenses" with the right form.
func PluralExpenses(lang string, n int) string {
	if lang == LangEN {
		if n == 1 {
			return "1 expense"
		}
		return fmt.Sprintf("%d expenses", n)
	}
	return fmt.Sprintf("%d %s", n, pluralRU(n, "расход", "расхода", "расходов"))
}

// ExpenseWord returns just the noun for a count, used under the donut.
func ExpenseWord(lang string, n int) string {
	if lang == LangEN {
		if n == 1 {
			return "expense"
		}
		return "expenses"
	}
	return pluralRU(n, "расход", "расхода", "расходов")
}

// pluralRU picks the Russian plural form by the CLDR one/few/many rules.
func pluralRU(n int, one, few, many string) string {
	if n < 0 {
		n = -n
	}
	mod10, mod100 := n%10, n%100
	switch {
	case mod10 == 1 && mod100 != 11:
		return one
	case mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14):
		return few
	default:
		return many
	}
}

// UpperFirst capitalises the first rune — Russian dates come lower-cased from
// the formatter but start a sentence in some places.
func UpperFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}
