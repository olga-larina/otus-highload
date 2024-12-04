cat people.v2.csv | docker exec -i deployments-db-1 psql -U otus -d backend -c "
-- Создаем временную таблицу для загрузки данных из CSV
CREATE TEMP TABLE temp_users_raw (
    full_name TEXT,
    birthdate DATE,
    city TEXT
);

-- Загружаем данные из CSV-файла в временную таблицу
COPY temp_users_raw (full_name, birthdate, city)
FROM STDIN
WITH (FORMAT csv, DELIMITER ',', ENCODING 'UTF8');

-- Обрабатываем данные и вставляем их в таблицу users
INSERT INTO users (id, first_name, second_name, city, birthdate, password_hash)
SELECT
    uuid_generate_v1() AS id,
    split_part(full_name, ' ', 2) AS first_name,
    split_part(full_name, ' ', 1) AS second_name, 
    city,
    birthdate,
    'default_hash' AS password_hash
FROM temp_users_raw;

-- Удаляем временную таблицу
DROP TABLE temp_users_raw;
"