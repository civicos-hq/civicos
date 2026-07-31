# Community Funding — manual test walkthrough

A full pass through the feature with three real accounts: an organization,
a platform admin, and a citizen. Roughly 20 minutes.

---

## Before you start

**1. Infrastructure**

```bash
docker compose -f infrastructure/docker-compose.yml up -d
```

**2. Environment** — the seed and the donation flow both need Paystack **test** keys.

```bash
set -a && source .env && set +a
```

**3. Services** — five terminals:

```bash
cd services/identity-service     && air
cd services/community-service    && air
cd services/organization-service && air
cd services/api-gateway          && air
pnpm dev                                  # web on :5173
```

organization-service needs these set (they are in `.env.example`):
`SMTP_HOST=localhost`, `SMTP_PORT=1025`, `APP_URL=http://localhost:5173`,
`DONATION_CALLBACK_URL=http://localhost:5173/donations/complete`,
`NATS_URL=nats://localhost:4222`.

**4. Seed the three accounts**

```bash
node scripts/seed-funding-demo.mjs
```

It prints the credentials and the organization's page URL. Re-running it
wipes and recreates the demo data, so you can start over at any point.

| Role               | Email                  | Password            |
| ------------------ | ---------------------- | ------------------- |
| Organization owner | `org@civicos.demo`     | `CivicOS-Demo-2026` |
| Platform admin     | `admin@civicos.demo`   | `CivicOS-Demo-2026` |
| Citizen            | `citizen@civicos.demo` | `CivicOS-Demo-2026` |

The organization (**Zaria Relief Trust**) is seeded already verified and with
payouts connected. Both are prerequisites: without them the donate endpoint
refuses with `ORG_NOT_FUNDING_ELIGIBLE`.

Use three browser profiles (or one normal + two incognito windows) so all
three can be signed in at once.

---

## Act 1 — The organization creates a campaign

Sign in as **org@civicos.demo** and open the organization page printed by the
seed script.

1. Find **Fundraising campaigns** → **New campaign**.
2. Fill it in. The form mirrors the server's rules, so the button stays
   disabled until they are met:
   - Title ≥ 4 characters — _Flood relief for Sabon Gari_
   - Summary ≥ 10 — _Emergency supplies for displaced households._
   - Description ≥ 40 — a paragraph about the flooding
   - Goal — `2000000` (₦2,000,000)
   - First spend-plan item — _Food and water_, amount `1200000`
3. **Create draft.**

**Check:** it appears with status **Draft**, a hint telling you to submit it,
and _no_ public link — a draft has no public page.

### Editing while it is still yours

4. Click **Edit**. The form arrives pre-filled, with the goal shown back in
   naira rather than kobo.
5. In **Spend plan**, note the line showing how much is left to allocate.
   Add a second item — _Temporary shelter_, `800000`.

**Check:** the plan now covers the full goal and says so. Try adding a third
item for more than ₦0 — the Add button stays disabled and tells you what is
left, rather than letting the server reject it.

6. **Save changes.**

---

## Act 2 — The admin reviews

7. Back as the org, click **Submit for review**.

**Check:** status becomes **In review**, and the **Edit** button disappears —
content freezes once it leaves the organization's hands, so a donor is always
giving to the thing they read. (Try `PATCH`ing it via the API if you like: it
returns `409`.)

Now sign in as **admin@civicos.demo** and open the admin console
(`http://localhost:5174`).

8. Go to the campaign review queue and open the campaign.

**Check:** you can see the organization's verification evidence — registration
number, country, official email, named representative, bank verification —
which is what a reviewer needs before approving a fundraiser.

9. Try **Needs changes** first, with a note like _"Please itemise the shelter
   costs."_

**Check:** back in the org window, the campaign shows **Changes needed** with
your note visible, and **Edit** is available again. That note appears _only_
on the org's dashboard, never on any public page.

10. Have the org submit again, then **Approve** it as the admin.
11. As the org, click **Publish**.

**Check:** the campaign is now **Live** with a **View page** link.

---

## Act 3 — A citizen donates

Sign in as **citizen@civicos.demo** and open the campaign's public page.

**Check before donating:** the goal, the full spend plan, and the fee
disclosure — the donate box shows exactly what reaches the organization and
what CivicOS takes (2.5%), _before_ you commit.

12. Choose ₦10,000, enter `citizen@civicos.demo`, add a name, and donate.
13. You land on Paystack's checkout. Pay with a test card:

    | Field  | Value                         |
    | ------ | ----------------------------- |
    | Card   | `4084 0840 8408 4081`         |
    | Expiry | any future date, e.g. `12/30` |
    | CVV    | `408`                         |
    | PIN    | `0000`                        |
    | OTP    | `123456`                      |

14. You are returned to `/donations/complete`.

**Check:** it says the donation is being **confirmed** — not "thank you, it
worked". Landing on that page only means the browser came back; settlement is
decided solely by the signed webhook.

### Settling it locally

Paystack cannot reach `localhost`, so the webhook never arrives in local
development and the donation stays `PENDING`. That is exactly the failure
reconciliation exists for — so use it. As the **admin**:

```bash
ADMIN=$(curl -s -X POST http://localhost:3000/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@civicos.demo","password":"CivicOS-Demo-2026"}' \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['tokens']['accessToken'])")

curl -s -X POST http://localhost:3000/api/v1/admin/donations/reconcile \
  -H "Authorization: Bearer $ADMIN" -H 'Content-Type: application/json' \
  -d '{"pendingGraceMinutes":0}' | python3 -m json.tool
```

**Check the report:** `recovered: 1`, `recoveredMinor` equal to your donation,
and one drift entry of kind `RECOVERED_MISSED_WEBHOOK` — the money is now
banked correctly, but it is still reported, because the delivery path failed
and that is worth knowing.

Run it a second time: `recovered: 0` and no drift. Settlement is idempotent.

**Check the campaign page:** the raised total, the donor list showing the
citizen's name, and the progress bar. Then check **Mailpit**
(`http://localhost:8025`) for the receipt — it shows the split to the kobo
(₦10,000.00 = ₦250.00 fee + ₦9,750.00 to the organization), and says plainly
that it is **not a tax receipt**, because the organization is the recipient of
the gift, not CivicOS.

> Only a signed-in donor is linked to their account. Donating signed-out still
> works — guest giving is deliberate — but a guest cannot receive in-app
> notifications, only the emailed receipt.

---

## Act 4 — The organization accounts for the money

Back as the **org**, on the campaign's public page. Because you administer the
organization, you see a **Manage this campaign** panel below the public
content — reporting against the same figures your donors are reading.

15. Under **Report spending**, add:
    - Milestone: _Food and water_
    - Amount: `6000`
    - What it was spent on: _Bought 200 bags of rice and 400 crates of water_
    - Date: any past date (the picker will not offer a future one)
16. **Publish spending.**

**Check the public section above:** _Where the money went_ now shows received,
reported and not-yet-accounted-for — to the kobo, so the three rows visibly
add up — plus the percentage accounted for and the itemised entry with who
published it. Below it, in plain words: CivicOS does not hold the money and
cannot verify these figures.

17. Under **Post an update**, add a headline and a paragraph, and optionally a
    link to a photo.

**Check:** it appears in _Updates from the organization_, and the citizen gets
a notification (see Act 5).

18. Scroll to the campaign row on your organization dashboard and use
    **Mark complete** on a milestone.

**Check:** the milestone shows as completed on the public spend plan. This is
the one plan change allowed after review — progress reporting, not rewriting
what donors were shown.

---

## Act 5 — What the citizen sees

Back as the **citizen**, on the campaign page.

**Check:**

- The full account of the spending, without needing to ask anyone. That is
  this phase's whole purpose.
- Your donation in the donor list — and **no email addresses anywhere** on the
  page.
- Your notifications (bell icon): a campaign update, and a milestone
  completion. If `NATS_URL` is set on both organization-service and
  community-service, these arrive **live** without a refresh; without it they
  appear on the next page load.

Try the anonymity flag: donate again with **Give anonymously** ticked. Your
name is replaced with _Anonymous_ on the public list — but you still get the
receipt and the notifications, because anonymity governs the public donor
list, not whether you hear what happened to your own money.

---

## Act 6 — Governance

As the **admin**, pause the campaign (reason: _Misuse reported_).

**Check:** the campaign stops accepting donations — the donate form is gone
and a new intent is refused with `CAMPAIGN_NOT_ACCEPTING`. Pausing is the main
lever that still works once funds settle directly to an organization, so it
has to actually stop the money.

**Check:** the organization can _still_ report spending while paused. An
organization under investigation is exactly who should be accounting for what
it already took.

Resume it, and donations open again.

---

## Known limits in local development

- **The webhook never fires locally.** Paystack cannot reach `localhost`. Use
  the reconciliation endpoint above, or expose the gateway with a tunnel
  (`cloudflared tunnel --url http://localhost:3000`) and point the Paystack
  dashboard webhook at `<tunnel>/api/v1/webhooks/paystack`.
- **Deleting** is only possible while a campaign is a draft. Anything already
  submitted is archived instead, so a campaign reviewers or donors have seen
  leaves a trail rather than vanishing.
- **Receipts as PDF** are not implemented; the emailed receipt prints to PDF
  from any client. See the plan doc for the reasoning.

## Resetting

```bash
node scripts/seed-funding-demo.mjs   # wipes and recreates the demo data
```
