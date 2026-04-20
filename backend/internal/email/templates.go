package email

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

// SupportedLanguages lists languages with localized templates. Unknown or empty
// language falls back to DefaultLanguage.
var SupportedLanguages = []string{"ru", "en"}

// DefaultLanguage is used when the caller passes an empty or unsupported lang.
// Target audience is Russian-speaking (Kazakhstan) per user_yelyam memory.
const DefaultLanguage = "ru"

// Template key constants — keep in sync with the migration's CHECK constraint
// on email_outbox.template_key.
const (
	TemplateOTP            = "otp"
	TemplatePasswordReset  = "password_reset"
	TemplateInvite         = "invite"
)

type emailTemplate struct {
	Subject string
	HTML    string
}

// templates is keyed by [templateKey][lang]. HTML bodies use html/template
// parameter interpolation; Subject is treated as a text/template too (simple
// Printf-style would also work but text/template keeps the API uniform).
var templates = map[string]map[string]emailTemplate{
	TemplateOTP: {
		"ru": {
			Subject: "FoodBI — код подтверждения",
			HTML: `<!doctype html><html><body style="font-family:-apple-system,Arial,sans-serif;color:#111;">
<h2>Подтверждение регистрации</h2>
<p>Здравствуйте, {{.FirstName}}!</p>
<p>Ваш код подтверждения:</p>
<p style="font-size:28px;font-weight:bold;letter-spacing:4px;background:#f3f3f3;padding:12px 20px;border-radius:8px;display:inline-block;">{{.Code}}</p>
<p>Код действителен 10 минут. Если вы не регистрировались в FoodBI, просто игнорируйте это письмо.</p>
<p style="color:#888;font-size:12px;">FoodBI — аналитика для ресторанов</p>
</body></html>`,
		},
		"en": {
			Subject: "FoodBI — verification code",
			HTML: `<!doctype html><html><body style="font-family:-apple-system,Arial,sans-serif;color:#111;">
<h2>Confirm your registration</h2>
<p>Hi {{.FirstName}},</p>
<p>Your verification code:</p>
<p style="font-size:28px;font-weight:bold;letter-spacing:4px;background:#f3f3f3;padding:12px 20px;border-radius:8px;display:inline-block;">{{.Code}}</p>
<p>This code expires in 10 minutes. If you did not register with FoodBI, you can safely ignore this email.</p>
<p style="color:#888;font-size:12px;">FoodBI — restaurant analytics</p>
</body></html>`,
		},
	},
	TemplatePasswordReset: {
		"ru": {
			Subject: "FoodBI — сброс пароля",
			HTML: `<!doctype html><html><body style="font-family:-apple-system,Arial,sans-serif;color:#111;">
<h2>Сброс пароля</h2>
<p>Здравствуйте, {{.FirstName}}!</p>
<p>Вы запросили сброс пароля. Чтобы задать новый пароль, перейдите по ссылке ниже:</p>
<p><a href="{{.ResetURL}}" style="display:inline-block;background:#111;color:#fff;padding:10px 20px;border-radius:8px;text-decoration:none;">Задать новый пароль</a></p>
<p>Ссылка действительна 1 час. Если вы не запрашивали сброс — просто игнорируйте это письмо.</p>
<p style="color:#888;font-size:12px;">FoodBI — аналитика для ресторанов</p>
</body></html>`,
		},
		"en": {
			Subject: "FoodBI — password reset",
			HTML: `<!doctype html><html><body style="font-family:-apple-system,Arial,sans-serif;color:#111;">
<h2>Password reset</h2>
<p>Hi {{.FirstName}},</p>
<p>You requested a password reset. Click the link below to set a new password:</p>
<p><a href="{{.ResetURL}}" style="display:inline-block;background:#111;color:#fff;padding:10px 20px;border-radius:8px;text-decoration:none;">Set a new password</a></p>
<p>This link expires in 1 hour. If you did not request a reset, you can safely ignore this email.</p>
<p style="color:#888;font-size:12px;">FoodBI — restaurant analytics</p>
</body></html>`,
		},
	},
	TemplateInvite: {
		"ru": {
			Subject: "Приглашение в FoodBI",
			HTML: `<!doctype html><html><body style="font-family:-apple-system,Arial,sans-serif;color:#111;">
<h2>Вас пригласили в FoodBI</h2>
<p>Компания <strong>{{.CompanyName}}</strong> приглашает вас присоединиться к FoodBI в роли <strong>{{.Role}}</strong>.</p>
<p><a href="{{.AcceptURL}}" style="display:inline-block;background:#111;color:#fff;padding:10px 20px;border-radius:8px;text-decoration:none;">Принять приглашение</a></p>
<p>Ссылка действительна 7 дней.</p>
<p style="color:#888;font-size:12px;">FoodBI — аналитика для ресторанов</p>
</body></html>`,
		},
		"en": {
			Subject: "You have been invited to FoodBI",
			HTML: `<!doctype html><html><body style="font-family:-apple-system,Arial,sans-serif;color:#111;">
<h2>You have been invited to FoodBI</h2>
<p><strong>{{.CompanyName}}</strong> has invited you to join FoodBI as <strong>{{.Role}}</strong>.</p>
<p><a href="{{.AcceptURL}}" style="display:inline-block;background:#111;color:#fff;padding:10px 20px;border-radius:8px;text-decoration:none;">Accept invitation</a></p>
<p>This link expires in 7 days.</p>
<p style="color:#888;font-size:12px;">FoodBI — restaurant analytics</p>
</body></html>`,
		},
	},
}

// normalizeLang returns a supported language, falling back to DefaultLanguage.
func normalizeLang(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	for _, l := range SupportedLanguages {
		if l == lang {
			return lang
		}
	}
	return DefaultLanguage
}

// Render produces (subject, html) for the given template key and language.
// Unknown template keys return an error; unknown languages silently fall back
// to DefaultLanguage. Params map values are exposed as template fields via a
// paramMap wrapper.
func Render(templateKey, lang string, params map[string]any) (string, string, error) {
	lang = normalizeLang(lang)

	byLang, ok := templates[templateKey]
	if !ok {
		return "", "", fmt.Errorf("unknown email template: %s", templateKey)
	}
	tmpl, ok := byLang[lang]
	if !ok {
		tmpl = byLang[DefaultLanguage]
	}

	subj, err := renderString("subject:"+templateKey, tmpl.Subject, params)
	if err != nil {
		return "", "", fmt.Errorf("render subject: %w", err)
	}
	body, err := renderString("body:"+templateKey, tmpl.HTML, params)
	if err != nil {
		return "", "", fmt.Errorf("render body: %w", err)
	}
	return subj, body, nil
}

func renderString(name, src string, params map[string]any) (string, error) {
	t, err := template.New(name).Parse(src)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, params); err != nil {
		return "", err
	}
	return buf.String(), nil
}
