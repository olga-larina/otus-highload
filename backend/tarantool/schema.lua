box.cfg{
    listen = 3301,
    memtx_memory = tonumber(os.getenv('TARANTOOL_MEMTX_MEMORY')) or 268435456,
    log_level = 6,
    net_msg_max = 4096,
    readahead = 131072,
    iproto_threads = 8,
    too_long_threshold = 1,
    worker_pool_threads = 8,
    memtx_max_tuple_size = 1048576,
    slab_alloc_factor = 1.1
}

-- Создаём таблицу для блокировок миграций
box.once("create_migration_lock", function()
    box.schema.space.create('migration_lock', {
        format = {{name = 'key', type = 'string'}}
    })
    box.space.migration_lock:create_index('primary', {parts = {'key'}})
end)

-- Пример блокировки миграции
local function acquire_lock()
    local success, err = pcall(function()
        box.space.migration_lock:insert{'migration_in_progress'}
    end)
    return success, err
end

local function release_lock()
    box.space.migration_lock:delete{'migration_in_progress'}
end

-- Получаем имя пользователя и пароль из переменных окружения
local admin_user = os.getenv('TARANTOOL_USER') or 'admin'
local admin_pass = os.getenv('TARANTOOL_PASSWORD') or 'password'

-- Создание пользователя
box.once("create_admin_user", function()
    box.schema.user.create(admin_user, {password = admin_pass}) -- Пользователь из ENV
    box.schema.user.grant(admin_user, 'read,write,execute', 'universe') -- Полные права
end)

-- Применение миграций
local function apply_migrations()
    local success, err = acquire_lock()
    if not success then
        print("Migration already in progress. Skipping.")
        return
    end

    local migration_files = {
        '001_create_messages_space.lua'
    }

    for _, file in ipairs(migration_files) do
        print("Applying migration: " .. file)
        local migration = dofile('/opt/tarantool/files/migrations/' .. file)
        migration.up()
    end

    release_lock()
end

apply_migrations()