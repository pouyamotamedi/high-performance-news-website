#!/bin/bash

echo "=== Testing Partition Management Functions ==="
echo

echo "1. Verifying functions were created..."
sudo -u newsapp psql -d newsdb -c "SELECT proname FROM pg_proc WHERE proname IN ('create_daily_partitions', 'drop_old_partitions', 'partition_maintenance') ORDER BY proname;"

echo
echo "2. Testing daily partition creation..."
sudo -u newsapp psql -d newsdb -c "SELECT * FROM create_daily_partitions();"

echo
echo "3. Checking what partitions exist now..."
sudo -u newsapp psql -d newsdb -c "SELECT tablename FROM pg_tables WHERE tablename LIKE 'articles_%' ORDER BY tablename;"

echo
echo "4. Testing partition cleanup (dry run with 365 days to avoid deleting anything)..."
sudo -u newsapp psql -d newsdb -c "SELECT * FROM drop_old_partitions(365);"

echo
echo "5. Testing full maintenance function..."
sudo -u newsapp psql -d newsdb -c "SELECT partition_maintenance();"

echo
echo "=== Test Complete ==="