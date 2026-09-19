#!/usr/bin/env bash
set -e

echo "🚀 Starting GitHub Setup for Go Ingestion Service..."

# -----------------------------------------------------------------------------
# 0. Pre-create Labels if they don't exist
# -----------------------------------------------------------------------------
echo "📌 Pre-creating GitHub Labels..."

LABELS=(
  "ingestion:1f75cb"
  "setup:a2eeef"
  "storage:0052cc"
  "rabbitmq:006b75"
  "pdf:bfd4f2"
  "api:d4c5f9"
  "email:fbca04"
  "docker:0db7ed"
  "testing:d4c5f9"
  "enhancement:a2eeef"
  "feature:a2eeef"
  "devops:0052cc"
)

for item in "${LABELS[@]}"; do
  NAME="${item%%:*}"
  COLOR="${item#*:}"
  gh label create "$NAME" --color "$COLOR" --force >/dev/null 2>&1 || true
done
echo "  [✓] Labels verified."

# -----------------------------------------------------------------------------
# 1. Ensure Milestones Exist
# -----------------------------------------------------------------------------
echo "📌 Resolving Milestones..."

ensure_milestone() {
  local title="$1"
  local desc="$2"

  # Check if milestone exists
  local exists
  exists=$(gh api repos/:owner/:repo/milestones --jq ".[] | select(.title==\"$title\") | .title")

  # Create if missing
  if [ -z "$exists" ]; then
    gh api repos/:owner/:repo/milestones \
      -f title="$title" \
      -f description="$desc" >/dev/null
  fi
}

M1_TITLE="Milestone 1: Ingestion Infrastructure Setup"
M2_TITLE="Milestone 2: Multi-Channel Ingestion Channels"
M3_TITLE="Milestone 3: Ingestion Containerization & Testing"

ensure_milestone "$M1_TITLE" "Stand up core Go module, RabbitMQ producer, and MinIO/S3 storage client."
ensure_milestone "$M2_TITLE" "Build HTTP API and Email scraping entry points to accept raw files and publish events."
ensure_milestone "$M3_TITLE" "Containerize the service, wire up docker-compose, and add integration tests."

echo "  [✓] Milestones verified."

# -----------------------------------------------------------------------------
# 2. Create Issues for Milestone 1
# -----------------------------------------------------------------------------
echo "📌 Creating Issues for Milestone 1..."

gh issue create \
  --title "[Ingestion] Initialize Go module and core directory structure" \
  --milestone "$M1_TITLE" \
  --label "enhancement,ingestion,setup" \
  --body "### Description
Set up the base Go application environment under \`services/ingestion-go\`.

### Tasks
- [ ] Initialize \`go.mod\` and add base dependencies (\`amqp091-go\`, \`minio-go/v7\`, \`gin-gonic/gin\`).
- [ ] Create package layout (\`cmd/api\`, \`cmd/email-listener\`, \`pkg/rabbitmq\`, \`pkg/storage\`, \`pkg/pdf\`).
- [ ] Implement environment variable configuration loader for connection strings (RabbitMQ URI, MinIO credentials, S3 bucket names)."

gh issue create \
  --title "[Ingestion] Implement S3/MinIO client for raw PDF uploads" \
  --milestone "$M1_TITLE" \
  --label "enhancement,ingestion,storage" \
  --body "### Description
Create a thread-safe client package in Go to store incoming raw PDF files in MinIO before processing.

### Tasks
- [ ] Create \`pkg/storage/s3.go\` using the official AWS/MinIO SDK.
- [ ] Add \`UploadInvoice(ctx, fileName, reader, fileSize)\` method.
- [ ] Implement automatic bucket initialization (\`invoices-raw\`) on application startup if it doesn't exist.
- [ ] Write unit tests with a mocked S3 endpoint or test container."

gh issue create \
  --title "[Ingestion] Create RabbitMQ event publisher" \
  --milestone "$M1_TITLE" \
  --label "enhancement,ingestion,rabbitmq" \
  --body "### Description
Build a resilient AMQP publisher that emits \`InvoiceIngestedEvent\` messages when new invoices arrive.

### Tasks
- [ ] Create \`pkg/rabbitmq/publisher.go\` with connection pooling and auto-reconnect handling.
- [ ] Define exchange \`invoices.exchange\` and routing key \`invoice.ingested\`.
- [ ] Implement JSON serialization matching the \`shared/contracts/rabbitmq_message.json\` contract:
  - \`audit_run_id\` (UUIDv4)
  - \`file_uri\` (S3 path)
  - \`source\` (\`\"EMAIL\"\` | \`\"REST_API\"\`)
  - \`ingested_at\` (ISO8601 Timestamp)
- [ ] Implement publisher acknowledgments (Confirm Mode) to guarantee message delivery."

# -----------------------------------------------------------------------------
# 3. Create Issues for Milestone 2
# -----------------------------------------------------------------------------
echo "📌 Creating Issues for Milestone 2..."

gh issue create \
  --title "[Ingestion] Create Go PDF metadata & text extractor" \
  --milestone "$M2_TITLE" \
  --label "enhancement,ingestion,pdf" \
  --body "### Description
Build a fast Go-native PDF processor to inspect digital PDFs before queuing.

### Tasks
- [ ] Create \`pkg/pdf/parser.go\` using a native library (e.g., \`ledongthuc/pdf\` or \`pdfcpu\`).
- [ ] Implement PDF file validation (check magic bytes to reject corrupt or non-PDF uploads).
- [ ] Extract raw text (if available) to detect pure digital vs. scanned/image PDFs early."

gh issue create \
  --title "[Ingestion] Build HTTP POST /invoices upload endpoint" \
  --milestone "$M2_TITLE" \
  --label "feature,ingestion,api" \
  --body "### Description
Implement a lightweight web API server allowing users or external systems to upload single/batch invoice PDFs via multipart web forms.

### Tasks
- [ ] Implement HTTP server under \`cmd/api/main.go\` using Gin or standard \`net/http\`.
- [ ] Create endpoint \`POST /api/v1/invoices/upload\`.
- [ ] Generate a new \`audit_run_id\` UUID for each file.
- [ ] Upload file to MinIO bucket \`invoices-raw/YYYY/MM/DD/{audit_run_id}.pdf\`.
- [ ] Publish \`InvoiceIngestedEvent\` to RabbitMQ.
- [ ] Return \`202 Accepted\` response with the generated \`audit_run_id\`."

gh issue create \
  --title "[Ingestion] Implement IMAP daemon to scrape invoice email attachments" \
  --milestone "$M2_TITLE" \
  --label "feature,ingestion,email" \
  --body "### Description
Create a background worker service in Go that monitors a designated AP inbox (e.g., \`invoices@company.com\`) for incoming emails with attached PDF invoices.

### Tasks
- [ ] Build daemon under \`cmd/email-listener/main.go\` using an IMAP client library (e.g., \`emersion/go-imap\`).
- [ ] Poll or IDLE on the target INBOX for unread messages.
- [ ] Extract PDF attachments from message MIME parts.
- [ ] Upload extracted attachments to MinIO storage.
- [ ] Publish \`InvoiceIngestedEvent\` event with \`source: \"EMAIL\"\` to RabbitMQ.
- [ ] Mark processed emails as read/seen or move them to a \`Processed\` IMAP folder."

# -----------------------------------------------------------------------------
# 4. Create Issues for Milestone 3
# -----------------------------------------------------------------------------
echo "📌 Creating Issues for Milestone 3..."

gh issue create \
  --title "[Ingestion] Add multi-stage Dockerfile for Go services" \
  --milestone "$M3_TITLE" \
  --label "devops,docker,ingestion" \
  --body "### Description
Create production-ready Docker builds for both the API server and Email listener binaries.

### Tasks
- [ ] Create \`services/ingestion-go/Dockerfile\` with a multi-stage build (use \`golang:1.22-alpine\` for building and minimal \`alpine:latest\` or \`distroless\` for execution).
- [ ] Add \`ingestion-api\` and \`ingestion-email\` service definitions to root \`docker-compose.yml\`.
- [ ] Verify environment variables properly configure communication with Docker services (\`invoice_rabbitmq:5672\`, \`invoice_minio:9000\`)."

gh issue create \
  --title "[Ingestion] Add end-to-end integration test suite" \
  --milestone "$M3_TITLE" \
  --label "testing,ingestion" \
  --body "### Description
Verify that an uploaded PDF correctly lands in MinIO and emits the expected message payload onto RabbitMQ.

### Tasks
- [ ] Create integration test under \`services/ingestion-go/tests/e2e_test.go\`.
- [ ] Test uploading sample digital PDF to \`POST /api/v1/invoices/upload\`.
- [ ] Assert file presence in MinIO bucket.
- [ ] Consume queue from RabbitMQ and validate payload against JSON schema contract."

echo "🎉 All Milestones and Issues created successfully!"