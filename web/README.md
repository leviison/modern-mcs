# modern-mcs web

React + TypeScript admin UI scaffold for `modern-mcs` backend.

## Runtime requirement
- Node.js `20.19+` (or `22.12+`) is required for the current Vite toolchain.
- For `nvm` users: `nvm use` (uses `web/.nvmrc`).

## Features scaffolded
- Login/logout
- Change password
- SQL profile list/create/update/delete
- Session list/revoke (admin)
- Migration status/apply

## Run
```bash
cd web
cp .env.example .env
npm install
npm run dev
```

By default this targets `VITE_API_BASE=http://localhost:8080`.
