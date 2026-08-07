package invitations

import (
	"fmt"
	"strings"
	"time"
)

// roleBlurb describes a permission level in terms of what the person will
// be able to do, not by its enum name. "You have been added as STAFF" tells
// an invitee nothing.
func roleBlurb(role string) string {
	switch role {
	case "OWNER":
		return "full control, including managing the team and payout details"
	case "ADMIN":
		return "publishing announcements, running campaigns and managing the team"
	default:
		return "recording work — updating assigned issues and posting progress updates"
	}
}

// InvitationEmail renders the invitation. Plain text and HTML carry the
// same information: the link has to work in a client that renders neither
// well, and a recipient who cannot see the HTML still needs to know who
// invited them and to what.
func InvitationEmail(
	orgName, inviterName, role string,
	title *string,
	link string,
	expiresAt time.Time,
) (subject, html, text string) {
	subject = fmt.Sprintf("%s invited you to join %s on CivicOS", inviterName, orgName)

	jobLine := ""
	if title != nil && strings.TrimSpace(*title) != "" {
		jobLine = fmt.Sprintf("Your role there: %s.", strings.TrimSpace(*title))
	}
	expires := expiresAt.Format("2 January 2006")

	var t strings.Builder
	fmt.Fprintf(&t, "%s has invited you to join %s on CivicOS.\n\n", inviterName, orgName)
	if jobLine != "" {
		fmt.Fprintf(&t, "%s\n", jobLine)
	}
	fmt.Fprintf(&t, "You will be able to help with %s.\n\n", roleBlurb(role))
	fmt.Fprintf(&t, "Accept the invitation:\n%s\n\n", link)
	fmt.Fprintf(&t, "This link expires on %s.\n\n", expires)
	t.WriteString("If you do not have a CivicOS account yet, you can create one from that page — ")
	t.WriteString("use this same email address so the invitation recognises you.\n\n")
	t.WriteString("If you were not expecting this, you can ignore this email. Nothing happens until you accept.\n")
	text = t.String()

	var h strings.Builder
	h.WriteString(`<div style="font-family:system-ui,-apple-system,Segoe UI,sans-serif;max-width:520px;margin:0 auto;color:#0f172a">`)
	fmt.Fprintf(&h, `<h1 style="font-size:20px;margin:0 0 16px">Join %s on CivicOS</h1>`, htmlEscape(orgName))
	fmt.Fprintf(&h,
		`<p style="margin:0 0 12px;line-height:1.55"><strong>%s</strong> has invited you to join <strong>%s</strong>.</p>`,
		htmlEscape(inviterName), htmlEscape(orgName))
	if jobLine != "" {
		fmt.Fprintf(&h, `<p style="margin:0 0 12px;line-height:1.55">%s</p>`, htmlEscape(jobLine))
	}
	fmt.Fprintf(&h,
		`<p style="margin:0 0 20px;line-height:1.55">You will be able to help with %s.</p>`,
		htmlEscape(roleBlurb(role)))
	fmt.Fprintf(&h,
		`<p style="margin:0 0 20px"><a href="%s" style="display:inline-block;background:#047857;color:#fff;`+
			`padding:11px 20px;border-radius:8px;text-decoration:none;font-weight:600">Accept invitation</a></p>`,
		htmlEscape(link))
	fmt.Fprintf(&h,
		`<p style="margin:0 0 12px;color:#475569;font-size:13px;line-height:1.55">This link expires on %s.</p>`,
		htmlEscape(expires))
	h.WriteString(`<p style="margin:0 0 12px;color:#475569;font-size:13px;line-height:1.55">` +
		`No CivicOS account yet? You can create one from that page — use this same email address so the ` +
		`invitation recognises you.</p>`)
	h.WriteString(`<p style="margin:0;color:#475569;font-size:13px;line-height:1.55">` +
		`If you were not expecting this, ignore this email. Nothing happens until you accept.</p>`)
	// Repeated as text because some clients strip the button entirely.
	fmt.Fprintf(&h,
		`<p style="margin:20px 0 0;color:#94a3b8;font-size:12px;word-break:break-all">%s</p>`,
		htmlEscape(link))
	h.WriteString(`</div>`)
	html = h.String()

	return subject, html, text
}

func htmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	return r.Replace(s)
}
