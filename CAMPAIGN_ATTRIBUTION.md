# Campaign attribution and conversion reporting

Frontend companion: Ziki idea https://ziki.dev/ideas/1433.

Apply `db/migrations/000005_campaign_events.up.sql` through the existing migration
workflow before serving the frontend companion. No seed or data reset is needed.
The model is also registered in the existing development AutoMigrate list.

## Contract

The optional `campaign` object on `/auth/register`, `/auth/google`, `/auth/apple`
and `/contact` contains `utm_source` (required for attribution), optional
`utm_medium`, `utm_campaign`, `utm_content`, `utm_term`, a random UUIDv4
`visit_id`, millisecond `captured_at`, and `consent: true`.
Labels accept 1–100 ASCII letters/digits, spaces and `_./:-`; configure campaign
links without personal data. Empty/malformed/expired optional attribution is
ignored without rejecting the business operation. Existing callers are compatible.

First eligible touch wins for 30 days. Direct/unknown traffic is not attributed.
`POST /campaign/visits` records an eligible consented visit; repeated IDs are
ignored. On a confirmed conversion the server inserts a missing visit as well,
so a failed browser visit request does not lose its conversion denominator.
The server freezes the visit's original campaign dimensions.

Account creation records `sign_up` only when a new account is created, including
Google/Apple. Existing-user logins do not count. A SHA-256 key derived from the
server account ID deduplicates account conversions. Successful sales enquiries
record `lead_submitted`, deduplicated to one lead session per attributed visit.
This deliberately measures lead sessions, not unique people or qualified leads.
Accounts/enquiries store a private campaign visit ID for first-party association;
the analytics table does not store names, emails, IPs, full URLs or raw account IDs.
Attribution writes have a two-second timeout. Database errors are logged and do not turn successful signup/contact
requests into failures. A failed conversion write is not automatically replayed;
the first-party visit linkage remains available for reconciliation.

## Authorized reporting

`GET /api/v1/backoffice/dashboard/campaigns?from=2026-10-01&to=2026-10-31`
uses the existing backoffice admin session authentication. Regular members and
anonymous visitors cannot read it. Dates are inclusive UTC calendar dates with
a maximum 90-day range. Responses use `Cache-Control: no-store`.

The existing protected reporting API returns source/medium/campaign, attributed
visits, lead sessions, registrations and two percentages. It does not require a
second dashboard or new CRM. Date selection groups visits first captured during
that interval; their conversions can occur later within the 30-day attribution
window. Rates divide distinct converting visits by all attributed visits in the
cohort. Registrations can exceed visits when multiple accounts share a browser,
but each visit counts once in the rate. Recent cohorts are still maturing.
No ad spend, revenue, ROAS or multi-touch claim is made.

## Consent and retention

The frontend captures and transmits this new measurement only after opt-in.
Withdrawal clears local attribution and stops subsequent events; already accepted
pseudonymous events are retained for reporting. This does not alter the separate
existing GA/GTM configuration or replace the broader consent work in Ziki #72.
Event rows expire after 90 days: reports purge expired rows before returning data.
The existing cron runner also starts `clean-campaign-events` at application startup
and every 24 hours, so cleanup does not depend on report usage. Private visit IDs follow the account/
contact record lifecycle; purged event dimensions cannot be recovered from them.

## Validation

`go test ./internal/campaign ./services/campaign ./pkg/controller/admin
./pkg/controller/auth ./pkg/router` covers optional metadata, expiry, invalid dates,
idempotent inserts, database errors, login exclusion and report access control.
SQL tests use `go-sqlmock` and do not touch any shared database.
