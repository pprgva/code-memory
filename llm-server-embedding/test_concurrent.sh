#!/bin/bash
# Test concurrent requests to verify no locking issues

echo "🚀 Test: Concurrent Requests"
echo "============================================================"
echo "Sending 5 requests simultaneously..."
echo ""

API_KEY="DmdPeuln8ox8k8CFfphuKscUW5xNvSPhtNSh-CT2ieI"

# Function to send request
send_request() {
    local id=$1
    local start=$(date +%s.%N)

    curl -s http://127.0.0.1:12000/v1/embeddings \
      -H "Authorization: Bearer $API_KEY" \
      -H "Content-Type: application/json" \
      -d '{"input":"Concurrent request '"$id"'","model":"multilingual-e5-large"}' \
      > /dev/null

    local end=$(date +%s.%N)
    local duration=$(echo "$end - $start" | bc)

    echo "✅ Request $id completed in ${duration}s"
}

# Launch 5 requests in parallel
for i in {1..5}; do
    send_request $i &
done

# Wait for all to complete
wait

echo ""
echo "✅ All 5 concurrent requests completed!"
echo "============================================================"
