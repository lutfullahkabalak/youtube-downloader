# Frontend (Vue 3 + Vite)

This repository includes a small Vue 3 UI that is **built into static assets** and served by the Go server.

- **Source**: `frontend/`
- **Built output**: `web/dist` (embedded into the Go binary via `//go:embed`)
- **Served at**: `GET /` (same port as the API)

## Local frontend development

From `frontend/`:

```bash
npm install
npm run dev
```

To serve the production UI through the Go server, build the frontend and then run the Go app:

```bash
npm run build
cd ..
go run main.go
```
