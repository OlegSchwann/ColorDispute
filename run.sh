#! /usr/bin/env bash
printf "Start.\n"

# запуск nginx - подстановка корня.
sed -i "s~CURRENT_ROOT~$(pwd)~g"  "nginx.conf"
nginx -c "$(pwd)/nginx.conf"
# nginx.conf:5    pid /tmp/nginx.pid;

# запуск сервера
go run "server/main.go" "server/reactor.go" &
# сохраняем его Process Identifier
echo $! > "/tmp/ColorDisput.pid"

firefox --new-window "http://localhost:8080/" ||
google-chrome "http://localhost:8080/" ||
open -a Safari "http://localhost:8080/" ||
printf "Откройте в браузере http://localhost:8080/.\n"

printf "Нажмите любую клавишу, что бы остановить...\n"
read -n1 -s
kill --signal SIGTERM $(cat "/tmp/ColorDisput.pid")
kill --signal SIGTERM $(cat "/tmp/nginx.pid")
rm "/tmp/ColorDisput.pid" "/tmp/nginx.pid"
printf "Finish.\n"
exit 0
