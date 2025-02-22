# Отказоустойчивость приложений

## Задание

Необходимо уменьшить число точек отказа в приложении.  
Требования:
- Поднять несколько слейвов PostgreSQL.
- Реализовать соединение со слейвами PostgreSQL через haproxy.
- Поднять несколько приложений и обеспечить их балансировку через nginx.
- Воспроизвести нагрузку.
- Под нагрузкой с помощью "kill -9" отключить один из слейвов PostgreSQL. Убедиться, что система осталась работоспособной.
- Под нагрузкой с помощью "kill -9" отключить один из инстансов бэкенда. Убедиться, что система осталась работоспособной.

## Реализация

В качестве основы для настройки репликации, мониторингов и выполнения НТ использовано домашнее задание №3 "Репликация" ([отчёт по ДЗ](../replication/report.md)).

- Настроена потоковая репликация Postgres с 1 мастером и 2 репликами по инструкции из [ДЗ №3](../replication/report.md#потоковая-репликация-с-1-мастером-и-2-репликами). Файлы развёртывания: [docker-compose-db-master.yaml](../deployments/docker-compose-db-master.yaml) и [docker-compose-db-replicas.yaml](../deployments/docker-compose-db-replicas.yaml).
- Поднят второй экземпляр приложения `backend-social-server-2`. Файл развёртывания: [docker-compose-balancing.yaml](../deployments/docker-compose-balancing.yaml). В связи с тем, что теперь несколько экземпляров будут подключаться к БД, потребуется изменить настройки пула соединений Postgres (чтобы не было переполнения). Для этого в конфиги и переменные окружения вынесены дополнительные параметры `DB_MASTER_MAX_CONNS` и `DB_REPLICAS_MAX_CONNS`.
- Настроен `haproxy` для соединения с репликами Postgres. Файл развёртывания: [docker-compose-balancing.yaml](../deployments/docker-compose-balancing.yaml). Конфигурация: [haproxy.cfg](../deployments/balancing/haproxy/haproxy.cfg)
- Настроен `nginx` для соединения с экземплярами сервиса `backend-social-server`. Файл развёртывания: [docker-compose-balancing.yaml](../deployments/docker-compose-balancing.yaml). Конфигурация: [nginx.conf](../deployments/balancing/nginx/nginx.conf)
- Добавлен дополнительный файл с переменными окружения [.env_balancing](../deployments/.env_balancing), где в качестве url реплик использован url `haproxy` (`DB_REPLICAS_URI=postgres://otus:password@haproxy:5432/backend`), а также настроено количество соединений в пуле с учётом наличия нескольких экземпляров.
- Для отслеживания нагрузки использованы мониторинги из ДЗ №3. Файл развёртывания: [docker-compose-monitoring.yaml](../deployments/docker-compose-monitoring.yaml).

## Нагрузочное тестирование

Для НТ использован алгоритм из ДЗ №3 с небольшими корректировками.  
План НТ - [balancing_read_test.jmx](balancing_read_test.jmx)  
Общее описание плана:
- в начале теста необходимо зарегистрировать нового пользователя и получить токен для последующего выполнения запроса `/user/search`. Поэтому для регистрации (`/user/register`) и получения токена (`/login`) использована `setUp` группа потоков с 1 потоком.
- само НТ методов `/user/search` и `/user/get/{userId}` выполняется в другой группе потоков с настраиваемым количеством потоков.  
В запросе `/user/get/{userId}` используется id пользователя, полученный на шаге `setUp`.  
В запросе `/user/search` используется токен, полученный на шаге `setUp`. Параметры метода `first_name` и `last_name` генерируются случайным образом и состоят из 3-х символов русского алфавита с первой заглавной буквой:
```
first_name=${__changeCase(${__RandomString(3,абвгдеёжзийклмнопрстуфхцчшщъыьэюя,firstName)},CAPITALIZE,)}
last_name=${__changeCase(${__RandomString(3,абвгдеёжзийклмнопрстуфхцчшщъыьэюя,lastName)},CAPITALIZE,)}
```
Количество потоков - 1000, скорость "разгона" - 10s, продолжительность - 60s, таймаут на соединение - 1s, таймаут на получение ответа - 5s.

В качестве url в данном НТ использован url `nginx`, т.е. `localhost:4088`.

### Отключение реплики Postgres

В процессе НТ была отключена реплика Postgres через `docker kill deployments-dbReplica2-1`.  
Примеры логов `haproxy` до отключения `dbReplica2`:
```
172.18.0.13:60970 [22/Feb/2025:11:46:45.996] pgsql_front pgsql_back/replica2 1/0/19436 541820 -- 85/85/84/0/0 0/0
172.18.0.15:37740 [22/Feb/2025:11:46:46.744] pgsql_front pgsql_back/replica2 1/0/19 1 -- 81/81/80/48/0 0/0
172.18.0.13:33832 [22/Feb/2025:11:47:05.371] pgsql_front pgsql_back/replica1 1/0/57 1 -- 84/84/83/51/0 0/0
```

После отключения `dbReplica2`:
```
[WARNING]  (8) : Server pgsql_back/replica2 is DOWN, reason: Layer4 timeout, info: " at initial connection step of tcp-check", check duration: 2000ms. 1 active and 0 backup servers left. 34 sessions active, 0 requeued, 0 remaining in queue.
172.18.0.15:58600 [22/Feb/2025:11:47:05.369] pgsql_front pgsql_back/replica2 1/-1/31003 0 SC 76/76/75/15/3 0/0
172.18.0.15:33810 [22/Feb/2025:11:47:36.418] pgsql_front pgsql_back/replica1 1/0/16 1 -- 85/85/84/83/0 0/0
```

До отключения все запросы были успешные. А после отключения `dbReplica2` перестала отвечать.

По графикам `jmeter` видно некоторое количество неуспешных запросов в момент отключения реплики (с 18 по 24 секунду). Далее все запросы успешные.  
![Jmeter](kill_postgres/jmeter.png)  
![Jmeter failed](kill_postgres/jmeter_failed.png)

Графики в графане подтверждают, что одна из реплик была остановлена, при этом запросы на вторую реплику и `haproxy` продолжались.  
![Containers](kill_postgres/containers.png)

Таким образом, при отключении одной из реплик Postgres работоспособность приложения не была нарушена.

### Отключение экземпляра сервиса

В процессе НТ был отключён один экземпляр сервиса через `docker kill deployments-backend-social-server-2-1`.  
Примеры логов `nginx` до отключения `backend-social-server-2`:
```
192.168.65.1 - - [22/Feb/2025:13:04:01 +0000] "GET /user/get/902f0944-9480-4c8c-9088-47d3d74b160b HTTP/1.1" 200 332 "-" "Apache-HttpClient/4.5.14 (Java/21.0.5)" "-" [upstream: 172.18.0.15:8080]
192.168.65.1 - - [22/Feb/2025:13:04:01 +0000] "GET /user/search?first_name=%D0%99%D1%80%D1%82&last_name=%D0%A0%D0%B1%D1%8C HTTP/1.1" 200 3 "-" "Apache-HttpClient/4.5.14 (Java/21.0.5)" "-" [upstream: 172.18.0.13:8080]
192.168.65.1 - - [22/Feb/2025:13:04:01 +0000] "GET /user/get/902f0944-9480-4c8c-9088-47d3d74b160b HTTP/1.1" 200 332 "-" "Apache-HttpClient/4.5.14 (Java/21.0.5)" "-" [upstream: 172.18.0.13:8080]
192.168.65.1 - - [22/Feb/2025:13:04:01 +0000] "GET /user/search?first_name=%D0%91%D0%B3%D0%B6&last_name=%D0%97%D0%B8%D1%84 HTTP/1.1" 200 3 "-" "Apache-HttpClient/4.5.14 (Java/21.0.5)" "-" [upstream: 172.18.0.15:8080]
```

После отключения `backend-social-server-2`:
```
2025/02/22 13:04:01 [error] 31#31: *32061 recv() failed (104: Connection reset by peer) while reading response header from upstream, client: 192.168.65.1, server: , request: "GET /user/get/902f0944-9480-4c8c-9088-47d3d74b160b HTTP/1.1", upstream: "http://172.18.0.13:8080/user/get/902f0944-9480-4c8c-9088-47d3d74b160b", host: "localhost:4088"
2025/02/22 13:04:01 [error] 31#31: *56568 recv() failed (104: Connection reset by peer) while reading response header from upstream, client: 192.168.65.1, server: , request: "GET /user/search?first_name=%D0%AA%D0%B3%D1%87&last_name=%D0%9D%D1%85%D1%87 HTTP/1.1", upstream: "http://172.18.0.13:8080/user/search?first_name=%D0%AA%D0%B3%D1%87&last_name=%D0%9D%D1%85%D1%87", host: "localhost:4088"

192.168.65.1 - - [22/Feb/2025:13:04:01 +0000] "GET /user/search?first_name=%D0%AE%D1%8D%D0%BF&last_name=%D0%AB%D1%86%D1%83 HTTP/1.1" 200 3 "-" "Apache-HttpClient/4.5.14 (Java/21.0.5)" "-" [upstream: 172.18.0.15:8080]
192.168.65.1 - - [22/Feb/2025:13:04:01 +0000] "GET /user/get/902f0944-9480-4c8c-9088-47d3d74b160b HTTP/1.1" 200 332 "-" "Apache-HttpClient/4.5.14 (Java/21.0.5)" "-" [upstream: 172.18.0.15:8080]
192.168.65.1 - - [22/Feb/2025:13:04:20 +0000] "GET /user/search?first_name=%D0%97%D1%87%D0%B6&last_name=%D0%96%D0%BA%D1%87 HTTP/1.1" 200 3 "-" "Apache-HttpClient/4.5.14 (Java/21.0.5)" "-" [upstream: 172.18.0.15:8080]
192.168.65.1 - - [22/Feb/2025:13:04:20 +0000] "GET /user/get/902f0944-9480-4c8c-9088-47d3d74b160b HTTP/1.1" 200 332 "-" "Apache-HttpClient/4.5.14 (Java/21.0.5)" "-" [upstream: 172.18.0.15:8080]
```

До отключения все запросы шли на оба экземляра (`172.18.0.15:8080` и `172.18.0.13:8080`). А после отключения `backend-social-server-2` запросы шли только на работающий экземпляр `172.18.0.15:8080`.

По графикам `jmeter` видно, что после отключения одного экземпляра производительность уменьшилась. При этом неуспешных запросов нет. 
![Jmeter](kill_service/jmeter.png)  

Графики в графане подтверждают, что один экземпляр был остановлен, при этом запросы на второй экземпляр и `nginx` продолжались.  
![Containers](kill_service/containers.png)

Таким образом, при отключении одного экземпляра сервиса работоспособность приложения не была нарушена.

## Запуск приложения
В [Makefile](../Makefile):
- `make up-balancing` - поднять окружение (БД Postgres master и реплики, tarantool, кеш, очередь, мониторинги, haproxy, nginx), автоматически применить миграции, поднять несколько реплик сервиса
- `make down-balancing` - потушить окружение и сервис

