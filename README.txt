docker run -it —rm -p 8020:8020 shurshun/check_rkn

curl -X POST -d '["34.246.38.204", "47.91.106.69"]' http://localhost:8020/check_ips

{"34.246.38.204":true,"47.91.106.69":true}