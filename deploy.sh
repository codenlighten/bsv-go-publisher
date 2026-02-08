ssh root@134.209.4.149
cd /root/bsv-go-publisher
git pull origin main
echo "SYNC_WAIT_TIMEOUT=5s" >> .env
docker-compose build
docker-compose restart bsv-publisher