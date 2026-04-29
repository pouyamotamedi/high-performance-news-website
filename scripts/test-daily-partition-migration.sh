#!/bin/bash

echo "=== Testing Daily Partition Migration ==="
echo

echo "1. Checking current partition structure BEFORE migration..."
echo "Monthly partitions:"
sudo -u newsapp psql -d newsdb -c "SELECT tablename FROM pg_tables WHERE tablename ~ '^articles_\d{4}_\d{2}$' ORDER BY tablename;"

echo
echo "Daily partitions:"
sudo -u newsapp psql -d newsdb -c "SELECT tablename FROM pg_tables WHERE tablename ~ '^articles_\d{4}_\d{2}_\d{2}$' ORDER BY tablename;"

echo
echo "2. Checking data count in existing partitions..."
sudo -u newsapp psql -d newsdb -c "
SELECT 
    schemaname,
    tablename,
    (xpath('/row/c/text()', query_to_xml(format('SELECT COUNT(*) as c FROM %I.%I', schemaname, tablename), false, true, '')))[1]::text::int as row_count
FROM pg_tables 
WHERE tablename LIKE 'articles_%' 
AND schemaname = 'public'
ORDER BY tablename;
"

echo
echo "3. Running the migration..."
echo "This will convert monthly partitions to daily partitions..."
read -p "Press Enter to continue with migration..."

echo
echo "=== Migration Complete ==="