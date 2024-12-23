# Шардирование диалогов

## Задание

Необходимо реализовать масштабируемую подсистему диалогов.
Требования:
- Обеспечить горизонтальное масштабирование хранилищ на запись с помощью шардинга.
- Предусмотреть:
    - Возможность решардинга
    - (опционально) “Эффект Леди Гаги” (один пользователь пишет сильно больше среднего)
    - Наиболее эффективную схему

## Реализация

### Требования:

Диалог пользователей состоит из следующих элементов:
- ID пользователя-отправителя
- ID пользователя-получателя
- текст сообщения
- дата отправки сообщения

Поддерживаются только диалоги между двумя пользователями (в т.ч., диалог пользователя с самим собой).

### Наиболее эффективной схемой шардирования выглядит следующая:
- Для каждой пары пользователей генерировать отдельный уникальный `dialog_id`, который будет означать ID диалога между ними.
- Удобнее `dialog_id` генерировать на уровне приложения, чтобы не задействовать для этого БД. Например, можно упорядочивать id пользователей, склеивать их id в строку, считать хэш по данной строке (например, murmur128) и использовать его в качестве `dialog_id`.
- Поле `dialog_id` использовать в качестве ключа шардирования.

Впоследствии могут потребоваться дополнительные оптимизации:
- горзионтальное масштабирование (увеличение количества реплик)
- архивация старых сообщений в отдельную таблицу или хранилище
- выполнение решардинга при необходимости

Необходимо следить за равномерным распределением данных. Как вариант - можно для этого использовать `smart sharding`. Т.е. развернуть контроллер, который будет следить за загруженностью шардов и, возможно, выполнять решардинг при необходимости (либо направлять соответствующие уведомления). 

### Плюсы данного решения:
- Запросы на получение диалога между двумя пользователями будут всегда выполняться на одном шарде. 
- Решается проблема "эффекта Леди Гаги", когда у пользователя много диалогов с разными пользователями, т.к. таблица шардируется по диалогам достаточно равномерно.
- Генерация `dialog_id` на уровне приложения позволяет не задействовать для этого БД, что улучшает производительность.
- Использование хэширования при генерации `dialog_id` приводит к более равномерному распределению значений.

### Ограничения данного решения:
- Вероятность коллизий в `dialog_id`. Чтобы её обработать, возможно, потребуется создавать дополнительную таблицу `dialogs` с полями `dialog_id`, `user_id_1`, `user_id_2` и проверять по ней значения. Тогда потребуется связать таблицу с сообщениями с таблицей диалогов `dialogs` (т.е. `dialogs` тоже шардировать по ключу `dialog_id` и сделать эти таблицы `co-locate`). Ещё в таком случае стоит подумать над использованием кэша, чтобы минимизировать запросы к БД.
- При получении всех диалогов конкретного пользователя (если возникнет такая необходимость) запрос может уходить на несколько шардов.
- Отсутствие `foreign key` по `user_id` в таблице с сообщениями. Если возникнет необходимость, нужно отдельно проверять существование пользователей. В данном ДЗ предполагаем, что приходят только существующие ID.

### Алгоритм

1) Поднимаем окружение и сервис через `make up-sharded` из [Makefile](../Makefile).

2) Для шардирования будем использовать Citus для PostgreSQL (см. [docker-compose-db-sharded.yaml](../deployments/docker-compose-db-sharded.yaml)). В рамках ДЗ один master, один manager, два workers (на первом этапе).

3) Таблица сообщений `messages` - см. [messages.sql](../backend/migrations/20241220180000_messages.sql). Т.к. в Citus есть ограничение `Distributed relations cannot have UNIQUE, EXCLUDE, or PRIMARY KEY constraints that do not include the partition column (with an equality operator if EXCLUDE).`, первичный ключ таблицы составной и состоит из `dialog_id` (ID диалога, будущий ключ шардирования) и `message_id` (ID сообщения).

4) Заходим в контейнер master
```
docker exec -it deployments_master psql -U otus -d backend
```

5) Делаем таблицу `messages` распределённой (шардированной по полю `dialog_id`). Можно вместо `create_distributed_table` использовать `create_distributed_table_concurrently`, чтобы шардирование таблицы выполнялось конкуретно и не блокировало чтение и запись.
```postgresql
backend=# SELECT create_distributed_table('messages', 'dialog_id');
NOTICE:  Copying data from local table...
NOTICE:  copying the data has completed
DETAIL:  The local data in the table is no longer visible, but is still on disk.
HINT:  To remove the local data, run: SELECT truncate_local_data_after_distributing_table($$public.messages$$)
 create_distributed_table
--------------------------

(1 row)
```

6) Удаляем лишние локальные данные
```postgresql
backend=# SELECT truncate_local_data_after_distributing_table($$public.messages$$);
 truncate_local_data_after_distributing_table
----------------------------------------------

(1 row)
```

7) Проверяем информацию по распределённым таблицам. Количество шардов по умолчанию - 32.
```postgresql
backend=# SELECT * FROM citus_tables;
 table_name | citus_table_type | distribution_column | colocation_id | table_size | shard_count | table_owner | access_method
------------+------------------+---------------------+---------------+------------+-------------+-------------+---------------
 messages   | distributed      | dialog_id           |             1 | 576 kB     |          32 | otus        | heap
(1 row)
```

8) Заполняем таблицу данными (совершаем вызовы `POST /dialog/{user_id}/send`).

9) Посмотрим план запроса select всех данных. Видим, что запрос пойдёт на все шарды:
```postgresql
backend=# EXPLAIN (VERBOSE ON) SELECT count(*) FROM messages;
                                                QUERY PLAN
----------------------------------------------------------------------------------------------------------
 Aggregate  (cost=250.00..250.02 rows=1 width=8)
   Output: COALESCE((pg_catalog.sum(remote_scan.count))::bigint, '0'::bigint)
   ->  Custom Scan (Citus Adaptive)  (cost=0.00..0.00 rows=100000 width=8)
         Output: remote_scan.count
         Task Count: 32
         Tasks Shown: One of 32
         ->  Task
               Query: SELECT count(*) AS count FROM public.messages_102008 messages WHERE true
               Node: host=localhost port=5432 dbname=backend
               ->  Aggregate  (cost=12.38..12.38 rows=1 width=8)
                     Output: count(*)
                     ->  Seq Scan on public.messages_102008 messages  (cost=0.00..11.90 rows=190 width=0)
                           Output: dialog_id, message_id, content, from_user_id, to_user_id, send_time
(13 rows)
```

10) Посмотрим план запроса select одного диалога (по `dialog_id`). Видим, что запрос пойдёт только на один шард:
```postgresql
backend=# EXPLAIN (VERBOSE ON) SELECT * FROM messages WHERE dialog_id='6784669214258ead072fa16272343262';
                                                QUERY PLAN
----------------------------------------------------------------------------------------------------------
 Custom Scan (Citus Adaptive)  (cost=0.00..0.00 rows=0 width=0)
   Output: remote_scan.dialog_id, remote_scan.message_id, remote_scan.content, remote_scan.from_user_id, remote_scan.to_user_id, remote_scan.send_time
   Task Count: 1
   Tasks Shown: All
   ->  Task
         Query: SELECT dialog_id, message_id, content, from_user_id, to_user_id, send_time FROM public.messages_102014 messages WHERE ((dialog_id)::text OPERATOR(pg_catalog.=) '6784669214258ead072fa16272343262'::text)
         Node: host=localhost port=5432 dbname=backend
         ->  Index Scan using messages_pk_102014 on public.messages_102014 messages  (cost=0.14..8.16 rows=1 width=392)
               Output: dialog_id, message_id, content, from_user_id, to_user_id, send_time
               Index Cond: ((messages.dialog_id)::text = '6784669214258ead072fa16272343262'::text)
(10 rows)
```

### Процесс решардинга без даунтайма

Добавим ещё одного воркера и выполним решардинг данных без дайнтайма.  

1) В [docker-compose-db-sharded.yaml](../deployments/docker-compose-db-sharded.yaml) раскомментируем `dbWorker3` и соответствующий вольюм `worker3-sharded-data` и перезапустим приложение через `make down-sharded` и `make up-sharded` из [Makefile](../Makefile).  
Как вариант, можно было бы добавить новую ноду без перезапуска такой командой: `SELECT * from citus_add_node('node-name', 5432);`

2) Заходим в контейнер master
```
docker exec -it deployments_master psql -U otus -d backend
```

3) Проверяем, что координатор видит новые шарды:
```postgresql
backend=# SELECT master_get_active_worker_nodes();
 master_get_active_worker_nodes
--------------------------------
 (deployments-dbWorker2-1,5432)
 (deployments-dbWorker1-1,5432)
 (deployments-dbWorker3-1,5432)
(3 rows)
```

4) Проверяем, что данные лежат всё ещё на двух узлах `dbWorker1` и `dbWorker2`:
```postgresql
backend=# SELECT nodename, count(*)
FROM citus_shards
GROUP BY nodename;
        nodename         | count
-------------------------+-------
 deployments-dbWorker1-1 |    16
 deployments-dbWorker2-1 |    16
(2 rows)
```

5) Т.к. данные не переехали на новые узлы, надо запустить перебалансировку. Если хотим, что рабалансировка осуществлялась параллельно, можно поменять параметр:
```postgresql
ALTER SYSTEM SET citus.max_background_task_executors_per_node = 2;
SELECT pg_reload_conf();
```

6) Для начала установим `wal_level` = `logical`, чтобы узлы могли переносить данные (текущий уровень `wal_level` имеет значение `replica`):
```postgresql
backend=# show wal_level;
 wal_level
-----------
 replica
(1 row)

backend=# ALTER system SET wal_level = logical;
ALTER SYSTEM
backend=# SELECT run_command_on_workers('alter system set wal_level = logical');
             run_command_on_workers
-------------------------------------------------
 (deployments-dbWorker1-1,5432,t,"ALTER SYSTEM")
 (deployments-dbWorker2-1,5432,t,"ALTER SYSTEM")
 (deployments-dbWorker3-1,5432,t,"ALTER SYSTEM")
(3 rows)
```

7) Перезапускаем все узлы в кластере, чтобы применить изменения wal_level, через `make down-sharded` и `make up-sharded` из [Makefile](../Makefile).

8) Проверяем на мастере и воркерах, что wal_level изменился:
```postgresql
backend=# show wal_level;
 wal_level
-----------
 logical
(1 row)
```

9) Запускаем ребалансировку на мастере:
```
docker exec -it deployments_master psql -U otus -d backend
```
```postgresql
backend=# SELECT citus_rebalance_start();
NOTICE:  Scheduled 10 moves as job 1
DETAIL:  Rebalance scheduled as background job
HINT:  To monitor progress, run: SELECT * FROM citus_rebalance_status();
 citus_rebalance_start
-----------------------
                     1
(1 row)
```

10) Следим за статусом ребалансировки, дожидаемся окончания:
```postgresql
backend=# SELECT * FROM citus_rebalance_status();
 job_id |  state   | job_type  |           description           |          started_at           |          finished_at          |                     details
--------+----------+-----------+---------------------------------+-------------------------------+-------------------------------+--------------------------------------------------
      1 | finished | rebalance | Rebalance all colocation groups | 2024-12-23 10:44:20.489172+00 | 2024-12-23 10:44:43.394281+00 | {"tasks": [], "task_state_counts": {"done": 10}}
(1 row)
```

11) Проверяем, что данные равномерно распределились по шардам:
```postgresql
backend=# SELECT nodename, count(*)
FROM citus_shards
GROUP BY nodename;
        nodename         | count
-------------------------+-------
 deployments-dbWorker1-1 |    11
 deployments-dbWorker2-1 |    11
 deployments-dbWorker3-1 |    10
(3 rows)
```

## Запуск приложения
В [Makefile](../Makefile):
- `make up-sharded` - поднять окружение (citus (1 master + 2 (3) workers + 1 manager), кеши, очередь), автоматически применить миграции, поднять сервис
- `make down-sharded` - потушить окружение и сервис

