## Запуск backend

В [Makefile](Makefile):
- `make up` - поднять окружение (БД Postgres), автоматически применить миграции, поднять сервис
- `make down` - потушить окружение и сервис

## Запуск клиента

В [Makefile](Makefile):
- `make run_backend_client` - собрать и выполнить клиентские запросы