#!/bin/bash
# Jaeger OpenSearch Index Cleanup Script
# Deletes indices older than 2 days

RETENTION_DAYS=1
OPENSEARCH_URL="http://localhost:9200"

echo "=== Jaeger Index Cleanup - $(date) ==="
echo "Retention: ${RETENTION_DAYS} days"
echo ""

# Get date from N days ago
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    CUTOFF_DATE=$(date -v-${RETENTION_DAYS}d +%Y-%m-%d)
else
    # Linux
    CUTOFF_DATE=$(date -d "${RETENTION_DAYS} days ago" +%Y-%m-%d)
fi

echo "Deleting indices older than: ${CUTOFF_DATE}"
echo ""

# Get all jaeger indices
INDICES=$(curl -s "${OPENSEARCH_URL}/_cat/indices/jaeger-*?h=index" | sort)

for index in $INDICES; do
    # Extract date from index name (format: jaeger-span-YYYY-MM-DD or jaeger-service-YYYY-MM-DD)
    INDEX_DATE=$(echo "$index" | grep -oE '[0-9]{4}-[0-9]{2}-[0-9]{2}')

    if [ -n "$INDEX_DATE" ]; then
        # Compare dates
        if [[ "$INDEX_DATE" < "$CUTOFF_DATE" ]]; then
            echo "Deleting old index: $index (date: $INDEX_DATE)"
            curl -s -X DELETE "${OPENSEARCH_URL}/${index}" | jq -r '.acknowledged'
        else
            echo "Keeping index: $index (date: $INDEX_DATE)"
        fi
    fi
done

echo ""
echo "=== Cleanup Complete ==="
echo ""
echo "Current indices:"
curl -s "${OPENSEARCH_URL}/_cat/indices/jaeger-*?v"
