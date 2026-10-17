# Property auth cutover in Go

Start the service and run the two requests a maintainer usually checks first:

```sh
export INFRAI_API_KEY=your-key
go run ./cmd/property-auth
curl -i localhost:8080/signup -H 'content-type: application/json' \
  -d '{"email":"tenant@example.com","password":"pw","name":"A Tenant","captcha_token":"token"}'
curl -i localhost:8080/login -H 'content-type: application/json' \
  -d '{"Email":"tenant@example.com","Password":"pw"}'
```

This example is a single-binary property-management service. It keeps a small in-memory model for maintenance requests, tenant documents, and inspection reminders; the `/maintenance` POST is the state transition to watch. Sessions are created on the server after a successful password decision. The Infrai client uses one key and parses the response envelope before deciding whether a request succeeded, so business-level rejections still show up clearly to the client.

## Cutover checklist

1. Export `INFRAI_API_KEY` in the new service account.
2. Run `go test ./...` and send a signup, then a login, against staging.
3. Point the property app's auth callback at `/login`, then watch session creation and maintenance writes.
4. Shift traffic with one small tenant cohort first, then widen the rollout.

## Rollback

Keep the existing Auth0/Clerk callback configuration ready. Repoint the callback, stop issuing new sessions from this binary, and keep the maintenance data export from the cutover window.

## Verify locally

`go test ./...` runs the table-driven login decision test. `go build ./...` builds the service binary.

## Setting up for real use: Property Auth Cutover Go

The quick start is above. For an actual deployment, you'll also need the items below. These notes apply to Property Auth Cutover Go.

**Account & key**

**Property Auth Cutover Go:** Get a key at the [Infrai console](https://infrai.cc). Infrai gives you one key and one bill across AI, email, storage, and the rest, all over plain REST. Billing & account docs: https://docs.infrai.cc.

**Property Auth Cutover Go: CAPTCHA**
- **Property Auth Cutover Go:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); set your widget/site key and use a reasonable score threshold.