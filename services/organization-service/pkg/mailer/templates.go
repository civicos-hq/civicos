package mailer

import (
	"fmt"
	"strings"
	"time"
)

// wat is Nigeria's timezone. A fixed offset is correct here rather than a
// tzdata lookup: West Africa Time is UTC+1 year-round with no daylight
// saving, so there is nothing to look up, and this avoids depending on
// tzdata being present in whatever container the service runs in.
var wat = time.FixedZone("WAT", 60*60)

// Receipt is everything the donation receipt needs to say.
//
// All amounts are integer minor units, matching the ledger. Nothing here is
// recomputed for display — the figures are exactly what was recorded, so a
// donor comparing the receipt with the campaign page cannot find a
// discrepancy that only exists in the email.
type Receipt struct {
	DonorName        string
	CampaignTitle    string
	CampaignURL      string
	OrganizationName string
	Reference        string
	Currency         string
	GrossMinor       int64
	PlatformFeeMinor int64
	NetMinor         int64
	PlatformFeeBps   int64
	SettledAt        time.Time
}

// DonationReceiptEmail renders the subject/HTML/text triple for a settled
// donation.
//
// Deliberately NOT called a tax receipt, and it says so. CivicOS is not the
// merchant of record — Paystack settles directly to the organization, and
// the organization is the recipient of the gift. Implying this document
// could be filed for tax relief would be a claim CivicOS is not entitled to
// make on another entity's behalf, and a donor acting on it could be misled
// into a filing they cannot support.
func DonationReceiptEmail(r Receipt) (subject, html, text string) {
	name := strings.TrimSpace(r.DonorName)
	if name == "" {
		name = "there"
	}
	gross := FormatMoney(r.GrossMinor, r.Currency)
	fee := FormatMoney(r.PlatformFeeMinor, r.Currency)
	net := FormatMoney(r.NetMinor, r.Currency)
	when := r.SettledAt.In(wat).Format("2 January 2006 at 15:04 WAT")
	rate := formatBps(r.PlatformFeeBps)

	subject = fmt.Sprintf("Your %s donation to %s", gross, r.OrganizationName)

	text = fmt.Sprintf(
		"Hi %s,\n\n"+
			"Thank you. Your donation has been confirmed.\n\n"+
			"Amount:        %s\n"+
			"To:            %s\n"+
			"Campaign:      %s\n"+
			"Date:          %s\n"+
			"Reference:     %s\n\n"+
			"Where it went\n"+
			"%s\n\n"+
			"Your money went directly to %s through Paystack. CivicOS never\n"+
			"held it.\n\n"+
			"Follow the campaign: %s\n\n"+
			"Keep this email as your record of the donation. It is not a tax\n"+
			"receipt — %s is the recipient of your gift, so any tax\n"+
			"documentation would come from them.\n\n"+
			"— CivicOS",
		name, gross, r.OrganizationName, r.CampaignTitle, when, r.Reference,
		splitTable(gross, rate, fee, r.OrganizationName, net),
		r.OrganizationName, r.CampaignURL, r.OrganizationName,
	)

	html = fmt.Sprintf(`<!doctype html>
<html lang="en">
  <body style="margin:0;padding:0;background:#f4f6fb;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;color:#0f172a;">
    <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#f4f6fb;padding:40px 16px;">
      <tr>
        <td align="center">
          <table role="presentation" width="560" cellpadding="0" cellspacing="0" style="max-width:560px;background:#ffffff;border-radius:12px;box-shadow:0 4px 12px rgba(15,23,42,0.06);overflow:hidden;">
            <tr>
              <td style="padding:32px 32px 0;">
                <p style="margin:0;font-size:12px;letter-spacing:0.16em;text-transform:uppercase;color:#174d95;font-weight:700;">CivicOS</p>
              </td>
            </tr>
            <tr>
              <td style="padding:18px 32px 4px;">
                <h1 style="margin:0;font-family:Georgia,'Times New Roman',serif;font-size:24px;line-height:1.25;color:#0f172a;">Thank you, %s.</h1>
              </td>
            </tr>
            <tr>
              <td style="padding:6px 32px 20px;">
                <p style="margin:0;font-size:15px;line-height:1.55;color:#334155;">Your donation to <strong>%s</strong> has been confirmed.</p>
              </td>
            </tr>
            <tr>
              <td style="padding:0 32px 8px;">
                <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="border:1px solid #e2e8f0;border-radius:10px;">
                  <tr>
                    <td style="padding:18px 20px 6px;">
                      <p style="margin:0;font-size:28px;font-weight:700;color:#0f172a;">%s</p>
                      <p style="margin:4px 0 0;font-size:13px;color:#64748b;">%s</p>
                    </td>
                  </tr>
                  <tr>
                    <td style="padding:10px 20px 18px;">
                      <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="font-size:13px;color:#334155;">
                        <tr><td style="padding:5px 0;color:#64748b;">Campaign</td><td align="right" style="padding:5px 0;">%s</td></tr>
                        <tr><td style="padding:5px 0;color:#64748b;">Reference</td><td align="right" style="padding:5px 0;font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:12px;">%s</td></tr>
                      </table>
                    </td>
                  </tr>
                </table>
              </td>
            </tr>
            <tr>
              <td style="padding:18px 32px 6px;">
                <p style="margin:0 0 8px;font-size:12px;letter-spacing:0.12em;text-transform:uppercase;color:#64748b;font-weight:700;">Where it went</p>
                <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="font-size:14px;color:#334155;">
                  <tr><td style="padding:6px 0;">You gave</td><td align="right" style="padding:6px 0;">%s</td></tr>
                  <tr><td style="padding:6px 0;color:#64748b;">CivicOS platform fee (%s)</td><td align="right" style="padding:6px 0;color:#64748b;">&minus;%s</td></tr>
                  <tr><td style="padding:10px 0 0;border-top:1px solid #e2e8f0;font-weight:600;">Reached %s</td><td align="right" style="padding:10px 0 0;border-top:1px solid #e2e8f0;font-weight:600;">%s</td></tr>
                </table>
              </td>
            </tr>
            <tr>
              <td style="padding:16px 32px 4px;">
                <p style="margin:0;font-size:13px;line-height:1.6;color:#475569;">Your money went directly to %s through Paystack. CivicOS never held it.</p>
              </td>
            </tr>
            <tr>
              <td style="padding:16px 32px 26px;">
                <a href="%s" style="display:inline-block;background:#1f6ed4;color:#ffffff;text-decoration:none;padding:12px 18px;border-radius:8px;font-size:14px;font-weight:600;">Follow this campaign</a>
              </td>
            </tr>
            <tr>
              <td style="padding:0 32px 30px;">
                <p style="margin:0;font-size:12px;line-height:1.6;color:#94a3b8;">Keep this email as your record of the donation. It is not a tax receipt &mdash; %s is the recipient of your gift, so any tax documentation would come from them.</p>
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
  </body>
</html>`,
		htmlEscape(name), htmlEscape(r.OrganizationName),
		gross, when,
		htmlEscape(r.CampaignTitle), htmlEscape(r.Reference),
		gross, rate, fee, htmlEscape(r.OrganizationName), net,
		htmlEscape(r.OrganizationName),
		htmlEscape(r.CampaignURL),
		htmlEscape(r.OrganizationName),
	)
	return subject, html, text
}

// splitTable renders the "where it went" block with the money column
// aligned. Organization names vary in length, so the padding has to be
// computed rather than baked into the format string — a receipt with a
// money column that staggers about looks careless, and this is the one
// document where a donor is being asked to trust our arithmetic.
func splitTable(gross, rate, fee, orgName, net string) string {
	rows := [][2]string{
		{"You gave", gross},
		{fmt.Sprintf("CivicOS platform fee (%s)", rate), "-" + fee},
		{fmt.Sprintf("Reached %s", orgName), net},
	}

	labelW, amountW := 0, 0
	for _, row := range rows {
		if n := len([]rune(row[0])); n > labelW {
			labelW = n
		}
		if n := len([]rune(row[1])); n > amountW {
			amountW = n
		}
	}

	var b strings.Builder
	for i, row := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		// Runes, not bytes: ₦ is multi-byte, and %-*s pads by byte count.
		labelPad := strings.Repeat(" ", labelW-len([]rune(row[0])))
		amountPad := strings.Repeat(" ", amountW-len([]rune(row[1])))
		fmt.Fprintf(&b, "  %s%s   %s%s", row[0], labelPad, amountPad, row[1])
	}
	return b.String()
}

// FormatMoney renders integer minor units as major units with a thousands
// separator and exactly two decimal places.
//
// Two decimals always, never rounded to whole units: a ₦62.50 platform fee
// displayed as ₦63 would not reconcile against the ₦62 actually recorded,
// and a receipt whose arithmetic does not add up is worse than no receipt.
func FormatMoney(minor int64, currency string) string {
	neg := minor < 0
	if neg {
		minor = -minor
	}
	major, cents := minor/100, minor%100

	var b strings.Builder
	s := fmt.Sprintf("%d", major)
	for i, d := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(d)
	}

	sign := ""
	if neg {
		sign = "-"
	}
	return fmt.Sprintf("%s%s%s.%02d", sign, currencySymbol(currency), b.String(), cents)
}

func currencySymbol(code string) string {
	switch strings.ToUpper(code) {
	case "NGN":
		return "₦"
	case "USD":
		return "$"
	case "GBP":
		return "£"
	case "EUR":
		return "€"
	default:
		return strings.ToUpper(code) + " "
	}
}

// formatBps renders basis points as a percentage without trailing noise:
// 250 → "2.5%", 300 → "3%", 25 → "0.25%".
//
// Integer arithmetic throughout. The rest of this system refuses floats
// anywhere near money, and a rate that renders as "2.4999999%" on the one
// receipt a donor actually reads would undo a lot of careful work upstream.
func formatBps(bps int64) string {
	whole, frac := bps/100, bps%100
	switch {
	case frac == 0:
		return fmt.Sprintf("%d%%", whole)
	case frac%10 == 0:
		return fmt.Sprintf("%d.%d%%", whole, frac/10)
	default:
		return fmt.Sprintf("%d.%02d%%", whole, frac)
	}
}

// htmlEscape is deliberately applied to every interpolated value. A campaign
// title and an organization name are both user-supplied, and this template is
// assembled with fmt.Sprintf rather than html/template.
func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")
	return r.Replace(s)
}
