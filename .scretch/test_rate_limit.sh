#!/bin/bash

echo "Testing Rate Limiting on Health Check endpoint..."
echo "This endpoint has lenient rate limiting (100 req/sec, burst 200)"
echo "Making 10 rapid requests to /hc:"

for i in {1..10}; do
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}\n" -k https://localhost:9876/hc)
    echo "Request $i: $response"
done

echo ""
echo "Testing Rate Limiting on Login endpoint (should be more restrictive)..."
echo "This endpoint has strict rate limiting (5 req/sec, burst 10)"
echo "Making 15 rapid requests to /login (should get rate limited):"

for i in {1..15}; do
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}\n" -k -X POST \
        -H "Content-Type: application/json" \
        -d '{"username": "test", "password": "test"}' \
        https://localhost:9876/login)
    echo "Request $i: $response"
done