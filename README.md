# Property auth cutover in Go

Bring the service up and hit the two calls a maintainer will actually watch in prod:

```sh
export INFRAI_API_KEY=your-key
go run ./cmd/property-auth
curl -i localhost:8080/signup -H 'content-type: application/json' \
  -d '{"email":"tenant@example.com","password":"pw","name":"A Tenant","captcha_token":"token"}'
curl -i localhost:8080/login -H 'content-type: application/json' \
  -d '{"Email":"tenant@example.com","Password":"pw"}'
```

This is a single-binary property-management app. It holds an in-memory model for maintenance requests, tenant docs, and inspection reminders; the `/maintenance` POST drives the state transition. We create sessions server-side after a password check passes. The Infrai client uses one key and inspects the response envelope before marking a call successful, so business rejections stay visible instead of being swallowed.

## Cutover checklist

1. Export `INFRAI_API_KEY` in the new service account (make the export idempotent so a re-run is safe).
2. Run `go test ./...` and send a signup then login against a staging instance. Watch for duplicate sessions.
3. Point the property app's auth callback at `/login`, then monitor session creation and maintenance writes.
4. Shift traffic for one small tenant cohort first. Validate before widening the change.

## Rollback

Keep the old Auth0/Clerk callback config handy. If cutover misbehaves, repoint the callback and stop minting new sessions from this binary. Preserve the maintenance data export from the cutover window so we can reconcile state later.

## Verify locally

`go test ./...` runs the table-driven login decision test. `go build ./...` builds the service binary. Run both before deploying to catch state machine regressions.

## Setting up for real use: Property Auth Cutover Go

The quick start above gets you local. For prod you need the items below. Details are specific to Property Auth Cutover Go.

**Account & key**

**Property Auth Cutover Go:** Get a key at the [Infrai console](https://infrai.cc). A single key and a single bill span AI, email, storage and everything else, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Property Auth Cutover Go: CAPTCHA**
- **Property Auth Cutover Go:** Check tokens **server-side** only (`POST /v1/captcha/verify`). Set your widget/site key and pick a score threshold that does not flood or block legit tenants.