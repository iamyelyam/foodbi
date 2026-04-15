# Spec: Telegram Payment Notification Bot

## Overview
Telegram bot that notifies restaurant managers about failed payment attempts at tables in real-time. If payment succeeds on a subsequent attempt, sends a success follow-up.

## Data Flow
```
Payment System → POST /api/v1/webhooks/payment → payment_attempts table → Telegram Bot → Chat
```

## Functional Requirements

### FR-1: Payment Webhook Endpoint
- `POST /api/v1/webhooks/payment` — receives payment events from external system
- Auth: HMAC signature verification via `X-Webhook-Signature` header + shared secret per company
- Payload: `{ terminal_id, table_number, guest_name, guest_phone, status (failed|success), amount, order_id, timestamp }`
- Stores every attempt in `payment_attempts` table
- On `failed`: immediately notify all subscribed Telegram chats for that terminal
- On `success`: check if there were prior failed attempts for same order — if yes, send success follow-up

### FR-2: Telegram Bot Commands
- `/start` — register chat, request company API key for linking
- `/locations` — list available terminals/locations for the linked company
- `/subscribe <terminal_ids>` — subscribe to notifications from selected terminals (comma-separated)
- `/unsubscribe <terminal_ids>` — unsubscribe from terminals
- `/status` — show current subscriptions
- `/help` — command reference

### FR-3: Notification Messages

**Failed payment:**
```
⚠️ Неуспешная оплата
📍 Терминал: {location_name}
🪑 Стол: {table_number}
👤 {guest_name}
📱 {guest_phone}
💰 {amount} ₸
🕐 {time}
```

**Success after failure:**
```
✅ Оплата прошла успешно
📍 Терминал: {location_name}
🪑 Стол: {table_number}
👤 {guest_name}
💰 {amount} ₸
🕐 {time}
(после {N} неуспешных попыток)
```

### FR-4: Multi-location Filtering
- Each subscription links a telegram_chat_id to one or more terminal_ids
- A chat can subscribe to terminals across different locations
- Terminal ID maps to location via a `terminals` table or direct terminal_id field

## Database Schema

### Table: `payment_attempts`
- id UUID PK
- company_id UUID FK → companies
- terminal_id VARCHAR NOT NULL
- order_id VARCHAR NOT NULL
- table_number VARCHAR NOT NULL
- guest_name VARCHAR
- guest_phone VARCHAR
- amount NUMERIC(12,2) NOT NULL
- status VARCHAR NOT NULL (failed/success)
- attempt_at TIMESTAMPTZ NOT NULL
- created_at TIMESTAMPTZ DEFAULT NOW()
- RLS: company_id = current_tenant

### Table: `telegram_subscriptions`
- id UUID PK
- company_id UUID FK → companies
- telegram_chat_id BIGINT NOT NULL
- terminal_ids TEXT[] NOT NULL
- is_active BOOLEAN DEFAULT true
- created_at TIMESTAMPTZ DEFAULT NOW()
- updated_at TIMESTAMPTZ DEFAULT NOW()
- UNIQUE(company_id, telegram_chat_id)
- RLS: company_id = current_tenant

### Table: `telegram_bot_links`
- id UUID PK
- company_id UUID FK → companies
- telegram_chat_id BIGINT NOT NULL UNIQUE
- linked_at TIMESTAMPTZ DEFAULT NOW()

## Architecture

### New packages:
- `backend/internal/telegram/` — bot service (long-polling), command handlers, message formatting
- `backend/internal/payments/` — webhook handler, payment attempt storage, notification trigger logic

### Integration points:
- `cmd/api/main.go` — add webhook route (outside JWT middleware), add bot startup
- Environment variables: `TELEGRAM_BOT_TOKEN`, `WEBHOOK_SECRET`

## Non-functional
- Bot runs as goroutine inside API server (not separate process)
- Webhook endpoint has rate limiting (100 req/min per terminal)
- Payment attempts older than 90 days auto-cleanup (future)
