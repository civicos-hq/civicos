package mailer

import (
	"fmt"
	"strings"
)

// VerificationEmail renders the subject/HTML/text triple for the email
// verification message. Kept here so the auth service stays content-free.
func VerificationEmail(name, verifyURL string) (subject, html, text string) {
	displayName := strings.TrimSpace(name)
	if displayName == "" {
		displayName = "there"
	}
	subject = "Verify your CivicOS email"

	text = fmt.Sprintf(
		"Hi %s,\n\n"+
			"Welcome to CivicOS. Confirm your email so you can file issues, sign petitions,\n"+
			"and follow your representatives.\n\n"+
			"Verify your email:\n%s\n\n"+
			"This link expires in 24 hours. If you didn't sign up for CivicOS, ignore this email.\n\n"+
			"— CivicOS",
		displayName, verifyURL,
	)

	html = fmt.Sprintf(`<!doctype html>
<html lang="en">
  <body style="margin:0;padding:0;background:#f4f6fb;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;color:#0f172a;">
    <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#f4f6fb;padding:40px 16px;">
      <tr>
        <td align="center">
          <table role="presentation" width="520" cellpadding="0" cellspacing="0" style="max-width:520px;background:#ffffff;border-radius:12px;box-shadow:0 4px 12px rgba(15,23,42,0.06);overflow:hidden;">
            <tr>
              <td style="padding:32px 32px 0;">
                <p style="margin:0;font-size:12px;letter-spacing:0.16em;text-transform:uppercase;color:#174d95;font-weight:700;">CivicOS</p>
              </td>
            </tr>
            <tr>
              <td style="padding:18px 32px 8px;">
                <h1 style="margin:0;font-family:Georgia,'Times New Roman',serif;font-size:24px;line-height:1.25;color:#0f172a;">Confirm your email, %s.</h1>
              </td>
            </tr>
            <tr>
              <td style="padding:8px 32px 16px;">
                <p style="margin:0;font-size:15px;line-height:1.55;color:#334155;">Welcome to CivicOS. Confirm your email so you can file issues, sign petitions, and follow your representatives.</p>
              </td>
            </tr>
            <tr>
              <td style="padding:16px 32px 24px;">
                <a href="%s" style="display:inline-block;background:#1f6ed4;color:#ffffff;text-decoration:none;padding:12px 18px;border-radius:8px;font-size:14px;font-weight:600;">Verify my email</a>
              </td>
            </tr>
            <tr>
              <td style="padding:0 32px 28px;">
                <p style="margin:0;font-size:12px;line-height:1.55;color:#64748b;">If the button doesn't work, paste this link into your browser:<br><a href="%s" style="color:#1f6ed4;word-break:break-all;">%s</a></p>
              </td>
            </tr>
            <tr>
              <td style="padding:0 32px 32px;border-top:1px solid #eef2f7;">
                <p style="margin:20px 0 0;font-size:12px;color:#94a3b8;">This link expires in 24 hours. If you didn't sign up for CivicOS, you can ignore this email.</p>
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
  </body>
</html>`, displayName, verifyURL, verifyURL, verifyURL)

	return subject, html, text
}
