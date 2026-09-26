#!/usr/bin/env bash
set -e

echo "🚀 Starting GitHub Setup for Event-Driven Go Ingestion Service..."

# -----------------------------------------------------------------------------
# 0. Pre-create Labels
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

  local exists
  exists=$(gh api repos/:owner/:repo/milestones --jq ".[] | select(.title==\"$title\") | .title")

  if [ -z "$exists" ]; then
    gh api repos/:owner/:repo/milestones \
      -f title="$title" \
      -f description="$desc" >/dev/null
  fi
}

M1_TITLE="Milestone 1: Ingestion Infrastructure & Bucket Notifications"
M2_TITLE="Milestone 2: Upload Channels & Extractor"
M3_TITLE="Milestone 3: Ingestion Containerization & Testing"

ensure_milestone "$M1_TITLE" "Configure MinIO-to-RabbitMQ event publishing and build S3 storage helper client."
ensure_milestone "$M2_TITLE" "Build HTTP API endpoint, email attachment scraper, and Go PDF metadata inspector."
ensure_milestone "$M3_TITLE" "Containerize services and run end-to-end event integration tests."

echo "  [✓] Milestones verified."

# -----------------------------------------------------------------------------
# 2. Create Issues for Milestone 1
# -----------------------------------------------------------------------------
echo "📌 Creating Issues for Milestone 1..."

gh issue create \
  --title "[Ingestion] Implement S3/MinIO client for raw PDF uploads" \
  --milestone "$M1_TITLE" \
  --label "enhancement,ingestion,storage" \
  --body "### Description
Create a thread-safe client package in Go to upload raw PDF invoices into MinIO storage.

### Tasks
- [ ] Create \`pkg/storage/s3.go\` using the official MinIO Go SDK (\`minio-go/v7\`).
- [ ] Add \`UploadInvoice(ctx, objectKey, reader, fileSize)\` method.
- [ ] Implement automatic bucket initialization (\`invoices-raw\`) on application startup if it doesn't exist.
- [ ] Write unit tests using a mocked MinIO client or local Docker test container."

gh issue create \
  --title "[Ingestion] Configure MinIO bucket notifications to RabbitMQ" \
  --milestone "$M1_TITLE" \
  --label "enhancement,ingestion,rabbitmq,storage" \
  --body "### Description
Configure MinIO bucket notifications so uploading a \`.pdf\` file to \`invoices-raw\` automatically publishes an \`s3:ObjectCreated:Put\` event to RabbitMQ.

### Tasks
- [ ] Configure \`MINIO_NOTIFY_AMQP\` environment variables in \`docker-compose.yml\`.
- [ ] Create setup script/hook (\`deploy/minio_init.sh\`) to bind bucket event \`arn:minio:sqs::primary:amqp\` on \`s3:ObjectCreated:*\` with \`.pdf\` suffix.
- [ ] Validate that uploading a sample PDF to MinIO pushes an S3 event payload to the \`invoices.exchange\` exchange on RabbitMQ."

# -----------------------------------------------------------------------------
# 3. Create Issues for Milestone 2
# -----------------------------------------------------------------------------
echo "📌 Creating Issues for Milestone 2..."

gh issue create \
  --title "[Ingestion] Create Go PDF metadata & validation helper" \
  --milestone "$M2_TITLE" \
  --label "enhancement,ingestion,pdf" \
  --body "### Description
Build a fast Go PDF helper to validate files before pushing them to MinIO.

### Tasks
- [ ] Create \`pkg/pdf/parser.go\` using a native Go library (\`ledongthuc/pdf\` or \`pdfcpu\`).
- [ ] Implement magic bytes check to reject non-PDF uploads before storage.
- [ ] Extract basic metadata (page count, digital text flag) to pass as S3 object tags."

gh issue create \
  --title "[Ingestion] Build HTTP POST /invoices/upload endpoint" \
  --milestone "$M2_TITLE" \
  --label "feature,ingestion,api" \
  --body "### Description
Implement a web API endpoint allowing users/systems to upload single or batch invoice PDFs directly to MinIO.

### Tasks
- [ ] Implement HTTP server under \`cmd/api/main.go\` using Gin or standard \`net/http\`.
- [ ] Create endpoint \`POST /api/v1/invoices/upload\`.
- [ ] Generate an \`audit_run_id\` UUIDv4 per file.
- [ ] Upload file directly to MinIO at \`invoices-raw/YYYY/MM/DD/{audit_run_id}.pdf\`.
- [ ] Return \`202 Accepted\` response with \`{ \"audit_run_id\": \"...\", \"file_uri\": \"...\" }\`."

gh issue create \
  --title "[Ingestion] Implement IMAP daemon to scrape invoice email attachments" \
  --milestone "$M2_TITLE" \
  --label "feature,ingestion,email" \
  --body "### Description
Build a background worker in Go that monitors an AP inbox (e.g., \`invoices@company.com\`) and uploads PDF attachments to MinIO.

### Tasks
- [ ] Build daemon under \`cmd/email-listener/main.go\` using \`emersion/go-imap\`.
- [ ] Poll or IDLE on INBOX for unread messages.
- [ ] Extract attached PDF files from email MIME bodies.
- [ ] Save attachments directly to MinIO bucket \`invoices-raw/\` (triggering automatic RabbitMQ event).
- [ ] Mark processed emails as read or move to \`Processed\` folder."

# -----------------------------------------------------------------------------
# 4. Create Issues for Milestone 3
# -----------------------------------------------------------------------------
echo "📌 Creating Issues for Milestone 3..."

gh issue create \
  --title "[Ingestion] Add multi-stage Dockerfile for Go services" \
  --milestone "$M3_TITLE" \
  --label "devops,docker,ingestion" \
  --body "### Description
Create minimal, production-ready container builds for the API and Email listener services.

### Tasks
- [ ] Create \`services/ingestion-go/Dockerfile\` using multi-stage build (\`golang:1.22-alpine\` builder, \`alpine:latest\` runtime).
- [ ] Add \`ingestion-api\` and \`ingestion-email\` service definitions to root \`docker-compose.yml\`.
- [ ] Connect environment variables to link with \`invoice_minio:9000\` and \`invoice_rabbitmq:5672\`."

gh issue create \
  --title "[Ingestion] Add end-to-end S3 event integration test" \
  --milestone "$M3_TITLE" \
  --label "testing,ingestion" \
  --body "### Description
Verify that an uploaded PDF in MinIO automatically generates an S3 event payload on RabbitMQ.

### Tasks
- [ ] Create integration test under \`services/ingestion-go/tests/e2e_test.go\`.
- [ ] Upload sample PDF via \`POST /api/v1/invoices/upload\`.
- [ ] Assert file creation in MinIO bucket \`invoices-raw\`.
- [ ] Listen to RabbitMQ queue \`invoice_created\` and verify the \`s3:ObjectCreated:Put\` event payload matches the uploaded file key."

echo "🎉 Milestones and Issues updated and created successfully!"