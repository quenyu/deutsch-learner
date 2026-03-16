# Frontend

SvelteKit scaffold for the Deutsch Learner interface.

## Design implementation

- Black + orange primary identity.
- Calm, premium visual rhythm with restrained support colors.
- Mobile-first layout and accessible controls.

## Pages (initial)

- `/resources`: catalog with filters and save actions.
- `/resources/[slug]`: detailed metadata page with outbound action.
- `/saved`: API-backed saved queue.

## Runtime env

- `PUBLIC_API_BASE_URL` (default: `http://localhost:8080`)
- `PUBLIC_USER_ID` (default: demo user `11111111-1111-1111-1111-111111111111`)

## Commands

```bash
npm install
npm run dev
npm run test:e2e
```
