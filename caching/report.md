# Кеширование ленты постов друзей

## Задание

Необходимо реализовать кеширование ленты постов друзей.
Требования:
- Лента постов друзей формируется на уровне кешей
- В ленте держать последние 1000 обновлений друзей
- Работает инвалидация кеша
- Обновление лент работает через очередь
- Есть возможность перестройки кешей из СУБД

## Реализация

Вариантов реализации кеширования ленты постов друзей много. Всё зависит от требований, от того, какая предполагается нагрузка, какое допустимое время ответа, сколько памяти готовы выделить под кеш, предполагается ли наличие "звёзд" (пользователей, которые имеют много подписчиков) и т.д.

Лента постов друзей должна формироваться следующим образом: получаем список друзей пользователя, затем получаем список постов каждого друга, сортируем их в порядке убывания даты создания, ограничиваем количество при необходимости (например, по условию это 1000). Лента может обновляться при следующих событиях: добавление / удаление друга, создание / обновление / удаление поста.

1) Если это необходимо, можно реализовать "прогрев" кеша, чтобы приложение стартовало с уже заполненным кешом. Возможно, там должны быть не все пользователи, а, например, активные в последние 2 дня. В ходе прогрева кеша поведение системы тоже может быть разным - возвращать ошибку, возвращать пустой ответ, не принимать запросы.  
2) Кеш можно организовать следующим образом: ключ - ID пользователя, значение - лента из 1000 постов друзей (отсортированных по убыванию даты создания поста). В ленте можно хранить как только ID постов (а за содержимым ходить в БД), так и все посты целиком. Также в отдельном кеше можно хранить соответствие пользователя и всех его "подписчиков" (т.е. тех, кто добавил его в друзья и у кого в ленте должны отображаться его посты). Если необходимо, можно сделать кеш с соответствием пользователя и его постов, чтобы минимизировать запросы БД.  
3) Есть несколько вариантов реализации обновления кеша.  
- можно удалять значение ключа в кеше, если произошло какое-то обновление, а заполнять новое значение только тогда, когда придёт новый запрос на получение ленты (т.е. обновлять при чтении). Т.е., например, при добавлении нового друга только инвалидировать кеш с лентой пользователя.  
- можно сразу обновлять новым значением (т.е. обновлять при записи). При этом можно заново запрашивать всю ленту из БД или же для некоторых событий реализовать "умное" обновление (например, при добавлении друга запрашивать только его посты и объединять с текущей лентой, хранящейся в кеше).  
- можно реализовать удаление по TTL - если, например, пользователь не запрашивал свою ленту в течение заданного времени; либо по дате последнего обновления.  
- можно сделать периодическое обновление кеша (по крону), чтобы перезаписывать данными из БД (на случай, если при обновлениях были потеряны какие-то события).  
- можно реализовать возможность отправки события, которое вызывает перестройку или инвалидацию кеша - по одному или по всем пользователям.  
- при отсутствии данных в кеше можно либо обновлять значение, либо считать, что его не существует.

События можно обрабатывать сразу (но здесь есть риск неравномерной нагрузки). Либо реализовать обработку событий через очередь. В этом случае события будут обрабатываться более равномерно, но задержки в обновлениях ленты будут больше.  
Также лучше учесть, что при обновлении одной записи следует её блокировать, чтобы не было одновременно одних и тех же запросов. Тогда запросы на чтение либо могут возвращать старые данные, либо вставать в очередь, чтобы получить свежие данные, когда блокировка будет снята. Если предполагается несколько экземпляров сервиса, то лучше делать распределённые блокировки.   

Событие "добавление друга" означает, что нужно обновить кеш с подписчиками друга (добавить туда текущего пользователя), а также перестроить ленту текущего пользователя, добавив туда посты друга.  
Событие "удаление друга" означает, что нужно обновить кеш с подписчиками друга (удалить оттуда текущего пользователя), а также перестроить ленту текущего пользователя, удалив оттуда посты друга, но добавив другие, чтобы обеспечить требование количества записей в ленте.  
Событие "создание поста" означает, что нужно запросить подписчиков пользователя, добавить в ленту каждого подписчика данный пост, удалив при этом самый старый (при превышении требуемого количества записей в ленте).  
Событие "обновление поста" означает, что нужно запросить подписчиков пользователя, просмотреть ленту каждого подписчика на предмет наличия там поста (возможно - бинарным поиском по дате добавления), обновить при необходимости.  
Событие "удаление поста" означает, что нужно запросить подписчиков пользователя и перестроить ленту каждого подписчика (т.к. при удалении поста данного пользователя нужно добавить другие для обеспечения требования количества записей в ленте). Как вариант - можно сначала проверить ленты подписчиков на предмет наличия там этого поста (например, по попаданию даты создания в интервал).  

Такой алгоритм будет плохо работать в случае наличия "звёзд" в нашей системе. Т.к. если у пользователя будут миллионы подписчиков, то создание нового поста будет приводить к обновлению большого количества данных. Для "звёзд" (тут ещё нужно определить, по какому критерию считать пользователя "звездой") можно использовать другой алгоритм. Создать дополнительный кеш ID пользователя - список "звёзд", на которых он подписан. В кеше с лентой пользователя хранить только посты тех друзей, которые являются обычными пользователями. Когда "звезда" пишет пост, то кеши её подписчиков не обновляются. Но когда пользователь делает запрос своей ленты, то выполняется несколько действий:
- запрашивается его лента с постами обычных друзей
- запрашивается список звёзд, на которых он подписан
- запрашивается последние X постов этих звёзд
- посты звёзд "склеиваются" с лентой постов обычных друзей.

### В данном ДЗ реализованы:
- отправка событий добавление / удаление друга, создание / обновление / удаление поста по rabbitmq
- отправка события, означающего инвалидацию всего кеша, через дополнительный запрос `POST /internal/cache/invalidate`
- кеш с подписчиками пользователя и кеш с постами целиком
- удаление из кеша по TTL после последнего обновления
- перестройка кеша с подписчиками на основе полученных из rabbitmq событий
- перестройка кеша с лентой постов на основе полученных из rabbitmq событий
- перед началом обновления конкретного ключа он блокируется во избежание одновременных запросов на обновление
- при отсутствии данных по пользователю в кеше выполняется запрос на получение данных из БД

## Запуск приложения
В [Makefile](../Makefile):
- `make up` - поднять окружение (БД Postgres master, кеши, очередь), автоматически применить миграции, поднять сервис
- `make down` - потушить окружение и сервис

