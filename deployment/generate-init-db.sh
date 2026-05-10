#!/bin/bash

#############################################
# Generate/Update init-db.sql from production database
# Exports the complete schema from a running site
# Usage: ./generate-init-db.sh <site-name>
# Example: ./generate-init-db.sh cryptonlisys
#############################################

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <site-name>"
    echo "Example: $0 cryptonlisys"
    echo ""
    echo "This exports the database schema from a running site"
    echo "and saves it as deployment/init-db.sql"
    exit 1
fi

SITE_NAME=$1
CONTAINER="${SITE_NAME}_postgres"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="/opt/${SITE_NAME}-website/deployment/.env"

if [ ! -f "$ENV_FILE" ]; then
    echo "Error: .env file not found at $ENV_FILE"
    exit 1
fi

source "$ENV_FILE"

DB_USER="${SITE_NAME}app"
DB_NAME="${SITE_NAME}db"

echo "Exporting schema from ${CONTAINER}..."
echo "Database: ${DB_NAME}"
echo "User: ${DB_USER}"

docker exec -e PGPASSWORD=$DB_PASSWORD $CONTAINER pg_dump \
    -U $DB_USER \
    -d $DB_NAME \
    --schema-only \
    --no-owner \
    --no-privileges \
    > "$SCRIPT_DIR/init-db.sql"

LINES=$(wc -l < "$SCRIPT_DIR/init-db.sql")
echo ""
echo "Done! Generated: $SCRIPT_DIR/init-db.sql ($LINES lines)"
echo ""
echo "To commit:"
echo "  git add deployment/init-db.sql"
echo "  git commit -m 'Update init-db.sql from production'"
echo "  git push origin master"
