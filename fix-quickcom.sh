#!/bin/bash
set -e

echo "🔧 Fixing QuickCom package-lock.json issue..."
echo ""

# Check if QuickCom directory exists
if [ ! -d "../QuickCom/backend" ]; then
    echo "❌ QuickCom not found at ../QuickCom/backend"
    echo "   Expected location: /home/rhythm493/Projects/Local/QuickCom/backend"
    exit 1
fi

echo "📍 Found QuickCom at ../QuickCom/backend"
cd ../QuickCom/backend

echo ""
echo "🗑️  Removing outdated package-lock.json..."
rm -f package-lock.json

echo "📦 Regenerating package-lock.json with npm install..."
npm install

echo ""
echo "✅ QuickCom dependencies fixed!"
echo ""
echo "🔙 Returning to pocket-assistant directory..."
cd ../../pocket-assistant

echo ""
echo "🚀 You can now run: docker compose up -d"
echo ""
