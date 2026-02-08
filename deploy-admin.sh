#!/bin/bash
# Deploy Admin Portal to Production

set -e

echo "🚀 Deploying GovHash Admin Portal"
echo "==================================="

# Step 1: Build frontend container
echo ""
echo "📦 Step 1: Building admin portal container..."
docker-compose build admin-portal

# Step 2: Start the container
echo ""
echo "🔄 Step 2: Starting admin portal..."
docker-compose up -d admin-portal

# Step 3: Wait for container
echo ""
echo "⏳ Step 3: Waiting for container to be ready..."
sleep 5

# Step 4: Verify container is running
echo ""
echo "✅ Step 4: Verifying deployment..."
if docker ps | grep -q bsv_admin_portal; then
    echo "✅ Admin portal container is running!"
    docker ps | grep bsv_admin_portal
else
    echo "❌ Admin portal container failed to start"
    docker logs bsv_admin_portal
    exit 1
fi

# Step 5: Update nginx config
echo ""
echo "📝 Step 5: Updating nginx configuration..."
echo ""
echo "Add this to your nginx config at /etc/nginx/sites-available/api.govhash.org:"
echo ""
cat << 'EOF'

    # Admin Portal
    location /admin {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        
        # Handle SPA routing
        try_files $uri $uri/ /index.html;
    }

EOF

echo ""
echo "Then run: sudo systemctl reload nginx"
echo ""
echo "🎉 Admin portal deployed!"
echo ""
echo "📍 Access at: https://api.govhash.org/admin"
echo "🔐 Login with ADMIN_PASSWORD from .env"
echo ""
