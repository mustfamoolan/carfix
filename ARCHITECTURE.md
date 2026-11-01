# Project Architecture — CarFix Platform (almoq3 Framework)

## 1. Framework Rules & File Structure (CRITICAL)
This project runs on the **almoq3** framework (Go + Fiber + GORM + PostgreSQL + Redis + Zap + React/Vite/shadcn) running on port 8080.
Always inspect existing files in `bootstrap/`, `config/`, `app/`, and `database/` first to match exact package names, imports, and `almoq3` conventions.

### Existing & Target Files (`snake_case` convention):
- **Controllers (`app/controllers/`):**
  - `auth_controller.go`
  - `vinsearch_controller.go`
  - `adminuser_controller.go`
- **Models (`app/models/`):**
  - `user.go`
  - `user_quota.go`
  - `search_log.go`
  - `report_cache.go`
- **Services (`app/services/`):**
  - `vinvalidator_service.go`
  - `carfaxtranslator_service.go`
  - `quotamanager_service.go`
- **Middlewares (`app/middleware/`):**
  - `auth_middleware.go`
  - `quotacheck_middleware.go`
- **Jobs (`app/jobs/`):**
  - `fetchcarfaxreportjob_job.go`
- **Routes (`routes/`):**
  - `api.go` (All API endpoints under `/api/...`)
  - `web.go` (Frontend static serving)

## 2. Core Data & Business Logic Rules
1. **Users (`User`):** Login uses `username` and `password`. Roles: `admin` | `user`. Fields: `id`, `full_name`, `phone`, `username` (unique), `password` (bcrypt hashed), `role`, `is_active` (bool).
2. **Quotas (`UserQuota`):** 1-to-1 with `User` (`user_id` unique FK). Fields: `total_limit`, `used_total`, `daily_limit`, `used_today`, `last_reset_date`, `expires_at`.
   - Remaining Total = `total_limit - used_total`.
   - Remaining Daily = `daily_limit - used_today`.
   - Quota is ONLY deducted when a report is successfully returned.
3. **Search Tracking (`SearchLog`):** Tracks every search attempt. Fields: `id`, `user_id`, `vin`, `vehicle_title`, `result_status` (`success`, `not_found`, `limit_exceeded`, `invalid_vin`, `error`), `fetch_source` (`cache`, `canada_node`, `none`), `duration_ms`, `ip_address`, `user_agent`, `created_at`.
4. **Reports Cache (`ReportCache`):** Primary key is `vin` (varchar 17). Fields: `vehicle_title`, `raw_html_en` (original Carfax HTML/CSS), `translated_html_ar` (Arabic RTL Carfax HTML), `summary_json` (JSONB), `fetched_at`.