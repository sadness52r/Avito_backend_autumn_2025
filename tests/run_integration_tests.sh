#!/bin/bash

set -a
source .env
set +a

echo "  Starting Integration Tests with environment:"
echo "  DB: ${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
echo "  App: http://localhost:${PORT}"

docker-compose -f docker-compose.test.yml up --build -d

echo "⏳ Waiting for services to be ready..."
until curl -s http://localhost:${PORT}/stats/system > /dev/null; do
    sleep 2
done

echo "✅ Services ready, running tests..."

cd integration
go test -v -timeout=5m
TEST_RESULT=$?

cd ..
docker-compose -f docker-compose.test.yml down

exit $TEST_RESULT