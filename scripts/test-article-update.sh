#!/bin/bash

echo "🧪 Testing Article Update Functionality"
echo "======================================"

# First, let's get a valid JWT token by logging in
echo "Step 1: Getting authentication token..."

# Try to get a token (you'll need to provide actual credentials)
echo "Please provide admin credentials to test update functionality:"
echo "Username: admin"
echo "Password: [your admin password]"

# For now, let's test if the endpoint is accessible
echo "Step 2: Testing update endpoint accessibility..."

# Test with a simple request (will fail due to auth, but we can see the error)
curl -X PATCH "https://a.10top.shop/api/v1/articles/1" \
     -H "Content-Type: application/json" \
     -d '{"title":"Test Update"}' \
     -v 2>&1

echo -e "\n\nStep 3: Check server logs for any errors..."
journalctl -u newsapp -n 10 --no-pager | grep -i "update\|patch\|error" || echo "No recent update logs found"

echo -e "\n\nTo test with proper authentication:"
echo "1. Login to admin panel: https://a.10top.shop/admin/login"
echo "2. Open browser console and get token: localStorage.getItem('auth_token')"
echo "3. Run: curl -X PATCH 'https://a.10top.shop/api/v1/articles/1' -H 'Authorization: Bearer YOUR_TOKEN' -H 'Content-Type: application/json' -d '{\"title\":\"Test Update\"}'"