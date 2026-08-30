package evidence

import (
	"errors"
	"html"
	"net/url"
	"strings"
)

var errEmailPresentationInvalid = errors.New("email presentation is invalid")

type emailFact struct {
	Label string
	Value string
}

type emailPresentationInput struct {
	BrandName      string
	Preheader      string
	Heading        string
	Intro          string
	BodyPlain      string
	BodyHTML       string
	ActionLabel    string
	ActionURL      string
	Facts          []emailFact
	SupportContact string
}

type renderedEmailPresentation struct {
	PlainText string
	HTML      string
}

func renderEmailPresentation(input emailPresentationInput) (renderedEmailPresentation, error) {
	input.Preheader = strings.TrimSpace(input.Preheader)
	input.BrandName = strings.TrimSpace(input.BrandName)
	input.Heading = strings.TrimSpace(input.Heading)
	input.Intro = strings.TrimSpace(input.Intro)
	input.BodyPlain = strings.TrimSpace(input.BodyPlain)
	input.BodyHTML = strings.TrimSpace(input.BodyHTML)
	input.ActionLabel = strings.TrimSpace(input.ActionLabel)
	input.ActionURL = strings.TrimSpace(input.ActionURL)
	input.SupportContact = strings.TrimSpace(input.SupportContact)
	if !boundedEmailText(input.BrandName, 200, false) || !boundedEmailText(input.Heading, 300, true) || !boundedEmailText(input.Preheader, 500, false) || !boundedEmailText(input.Intro, 2000, false) || !boundedEmailText(input.BodyPlain, 20000, false) || !boundedEmailText(input.SupportContact, 500, false) {
		return renderedEmailPresentation{}, errEmailPresentationInvalid
	}
	if (input.ActionLabel == "") != (input.ActionURL == "") || !boundedEmailText(input.ActionLabel, 120, false) {
		return renderedEmailPresentation{}, errEmailPresentationInvalid
	}
	if input.ActionURL != "" {
		parsed, err := url.Parse(input.ActionURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || strings.ContainsAny(input.ActionURL, "\r\n") {
			return renderedEmailPresentation{}, errEmailPresentationInvalid
		}
	}
	if len(input.Facts) > 12 {
		return renderedEmailPresentation{}, errEmailPresentationInvalid
	}
	for _, fact := range input.Facts {
		if !boundedEmailText(strings.TrimSpace(fact.Label), 100, true) || !boundedEmailText(strings.TrimSpace(fact.Value), 500, true) {
			return renderedEmailPresentation{}, errEmailPresentationInvalid
		}
	}

	plain := []string{input.Heading}
	if input.Intro != "" {
		plain = append(plain, input.Intro)
	}
	if input.BodyPlain != "" {
		plain = append(plain, input.BodyPlain)
	}
	for _, fact := range input.Facts {
		plain = append(plain, strings.TrimSpace(fact.Label)+": "+strings.TrimSpace(fact.Value))
	}
	if input.ActionURL != "" {
		plain = append(plain, input.ActionLabel+": "+input.ActionURL)
	}
	if input.SupportContact != "" {
		plain = append(plain, "Support: "+input.SupportContact)
	}

	var body strings.Builder
	brandName := input.BrandName
	if brandName == "" {
		brandName = "ClearSight"
	}
	body.WriteString(`<!doctype html><html><body style="margin:0;padding:0;background:#f3f5f7;color:#17212b;font-family:Arial,Helvetica,sans-serif;">`)
	if input.Preheader != "" {
		body.WriteString(`<div style="display:none;max-height:0;overflow:hidden;opacity:0;color:transparent;">` + html.EscapeString(input.Preheader) + `</div>`)
	}
	body.WriteString(`<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="width:100%;background:#f3f5f7;"><tr><td align="center" style="padding:24px 12px;"><table role="presentation" width="600" cellspacing="0" cellpadding="0" style="width:100%;max-width:600px;background:#ffffff;border:1px solid #d9e0e7;border-radius:12px;"><tr><td style="padding:24px 28px 12px;color:#145c4a;font-size:15px;font-weight:700;letter-spacing:.02em;">` + html.EscapeString(brandName) + `</td></tr><tr><td style="padding:8px 28px 28px;">`)
	body.WriteString(`<h1 style="margin:0 0 14px;font-size:26px;line-height:1.25;color:#17212b;">` + html.EscapeString(input.Heading) + `</h1>`)
	if input.Intro != "" {
		body.WriteString(`<p style="margin:0 0 18px;font-size:16px;line-height:1.55;color:#344454;">` + html.EscapeString(input.Intro) + `</p>`)
	}
	if input.BodyHTML != "" {
		body.WriteString(`<div style="font-size:15px;line-height:1.55;color:#344454;">` + input.BodyHTML + `</div>`)
	}
	if len(input.Facts) != 0 {
		body.WriteString(`<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="margin:18px 0;border-top:1px solid #e4e9ee;">`)
		for _, fact := range input.Facts {
			body.WriteString(`<tr><td style="padding:9px 8px 9px 0;font-size:13px;color:#637282;vertical-align:top;">` + html.EscapeString(strings.TrimSpace(fact.Label)) + `</td><td style="padding:9px 0 9px 8px;font-size:14px;font-weight:600;color:#17212b;text-align:right;vertical-align:top;">` + html.EscapeString(strings.TrimSpace(fact.Value)) + `</td></tr>`)
		}
		body.WriteString(`</table>`)
	}
	if input.ActionURL != "" {
		body.WriteString(`<p style="margin:22px 0 14px;"><a data-primary-action="true" href="` + html.EscapeString(input.ActionURL) + `" style="display:inline-block;padding:13px 20px;background:#145c4a;color:#ffffff;text-decoration:none;font-size:16px;font-weight:700;border-radius:8px;">` + html.EscapeString(input.ActionLabel) + `</a></p>`)
		body.WriteString(`<p style="margin:0 0 18px;font-size:12px;line-height:1.5;color:#637282;">If the button does not open, copy this secure link into your browser:<br><span style="word-break:break-all;">` + html.EscapeString(input.ActionURL) + `</span></p>`)
	}
	if input.SupportContact != "" {
		body.WriteString(`<p style="margin:18px 0 0;padding-top:16px;border-top:1px solid #e4e9ee;font-size:13px;line-height:1.5;color:#637282;">Need help? Contact ` + html.EscapeString(input.SupportContact) + `.</p>`)
	}
	body.WriteString(`</td></tr></table></td></tr></table></body></html>`)
	return renderedEmailPresentation{PlainText: strings.Join(plain, "\n\n"), HTML: body.String()}, nil
}

func boundedEmailText(value string, limit int, required bool) bool {
	if required && value == "" {
		return false
	}
	return len(value) <= limit && !strings.ContainsAny(value, "\r\n\x00")
}
