return {
    up = function()
        -- Создаём space для сообщений
        if not box.space.messages then
            box.schema.space.create('messages', {
                format = {
                    {name = 'dialog_id', type = 'string'},
                    {name = 'message_id', type = 'string'},
                    {name = 'content', type = 'string'},
                    {name = 'from_user_id', type = 'string'},
                    {name = 'to_user_id', type = 'string'},
                    {name = 'send_time', type = 'string'}
                }
            })
            box.space.messages:create_index('primary', {
                parts = {'dialog_id', 'message_id'}, 
                unique = true,
                type = "TREE"
            })
        end

        -- Создаём пользовательские функции
        rawset(_G, 'insert_message', function(dialog_id, message_id, content, from_user_id, to_user_id)
            local send_time = os.date('%Y-%m-%dT%H:%M:%S')
            box.space.messages:insert{dialog_id, message_id, content, from_user_id, to_user_id, send_time}
            return send_time
        end)

        rawset(_G, 'batch_insert_messages', function(messages)
            -- Проверяем, что аргумент является массивом
            if type(messages) ~= 'table' then
                error("Expected a table of messages")
            end
        
            -- Вставляем каждое сообщение
            for _, msg in ipairs(messages) do
                local dialog_id = msg.dialog_id
                local message_id = msg.message_id
                local content = msg.content
                local from_user_id = msg.from_user_id
                local to_user_id = msg.to_user_id
                local send_time = os.date('%Y-%m-%dT%H:%M:%S')
        
                -- Выполняем вставку
                box.space.messages:insert{dialog_id, message_id, content, from_user_id, to_user_id, send_time}
            end
        
            return true
        end)

        -- без сортировки по убыванию send_time
        rawset(_G, 'get_messages_by_dialog', function(dialog_id)
            return box.space.messages.index.primary:select(dialog_id, {iterator = 'EQ'})
        end)
    end
}