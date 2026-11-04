Файл docker-compose.app.yaml
Отсюда брал сервис Uptime Kuma + MariaDB:
https://github.com/meibraransari/uptime-kuma
Внес несколько правок: 1) порт БД не пробрасывается, но если надо посмотреть ее строки через DBeaver, используется socat как временный мост между хост-машиной и контейнером БД, пока контейнер socat работает (используется только в локальном тестовом окружении, не на проде) P.S. – ПЕРЕДУМАЛ СОКАТА НЕТ ДУМАЮ ОН НЕ НУЖЕН; 2) вместо bind mounts используются именованные тома; 3) добавлены TZ=Europe/Moscow, чтобы время в логах (полезно если упал Uptime Kuma), отображалось по собственному ЧП, а не UTC.
Конфигурация, секреты в .env.monitoring (секреты задаются вручную, структуру брать из .env.monitoring.example:
```cp .env.monitoring.example .env.monitoring```
)

Докер компос был взят из https://github.com/docker/awesome-compose/blob/master/prometheus-grafana/compose.yaml
+ Добавлен нетворк (ведь docker-compose.monitoring.yml не имеет смысла, если не запущен docker-compose.app.yml, и должен иметь доступ к нему), depends_on графаны от проеметеуса, волумы для графаны вместо бинд маунтс
cadvisor взят из https://medium.com/@sohammohite/docker-container-monitoring-with-cadvisor-prometheus-and-grafana-using-docker-compose-b47ec78efbc 
экспортер мариядб сгенерил гемини

докерфайл из за проблем бинд маунтс и даггера (надо делать двойной проброс портов)

  команды вызова большие создал мейкфайл

set -a; source .env.app; set +a

docker exec -i -e MYSQL_PWD="$MARIADB_ROOT_PASSWORD" mariadb mariadb -uroot <<SQL
CREATE USER IF NOT EXISTS 'exporter'@'%' IDENTIFIED BY '${EXPORTER_PASSWORD}' WITH MAX_USER_CONNECTIONS 3;
ALTER USER 'exporter'@'%' IDENTIFIED BY '${EXPORTER_PASSWORD}';
GRANT PROCESS, REPLICATION CLIENT, SLAVE MONITOR, SELECT ON *.* TO 'exporter'@'%';
FLUSH PRIVILEGES;

для EXPORTER_USER и EXPORTER_PASSWORD чтобы экспортер не плакал в логи (либо это как-то автоматизировать хз)

используется mysqld-exporter v0.14.0, по идее deprecated, но не нужно писать .my.cnf

через аптайм куму получить апи кей и положить в .env.monitoring

кучу времени всрал на адвайзор чтобы его дешборд работал в итоге пришел к тому что надо было просто найти более новый дешборд

добавил нод экспортер по https://www.metalnikovg.ru/blog/monitoring-linux-hostov-s-prometheus-node-exporter. но без network_mode: host. типа а как тогда с прометеусом в одной сети быть непонятно. т.е. без сетевых интерфейсов хоста достоверных буду. техдолг наверное так то