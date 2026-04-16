# FoodBI Ops Log

## 2026-04-16 21:05 — Full restart after build
- Killed stale processes: `pkill -9 -f foodbi`, `lsof -ti :8080 | xargs kill -9`
- Rebuilt: `go build -o foodbi-api ./cmd/api` ✅
- Started API: PID 58866, port 8080 ✅
- Frontend: `npm run build` + `npx cap sync ios` ✅
- Status: API running, iOS synced

## 2026-04-16 21:20 — Full rebuild after location filter fix
- Killed all stale: pkill -9 -f foodbi + lsof :8080
- Backend rebuilt: go build -o foodbi-api ./cmd/api ✅
- API started and verified: curl dashboard/summary with location_ids ✅
- Frontend: tsc ✅, npm run build ✅, cap sync ios ✅  
- Fix: selectedLocationIds now persists to localStorage as JSON array
