docker run -it —rm -p 8020:8020 shurshun/check_rkn

curl -X POST -d '["8.8.8.8", "8.8.4.4"]' http://localhost:8020/check_ips

{"34.246.38.204":true,"47.91.106.69":true}