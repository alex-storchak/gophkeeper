CREATE TABLE IF NOT EXISTS users
(
    id            BIGSERIAL PRIMARY KEY,
    username      VARCHAR(255)             NOT NULL UNIQUE,
    password_hash VARCHAR(255)             NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS data_types
(
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(50)  NOT NULL UNIQUE,
    description VARCHAR(255) NOT NULL DEFAULT ''
);

INSERT INTO data_types (code, description)
VALUES ('credentials', 'Пара логин/пароль'),
       ('card', 'Данные банковской карты'),
       ('text', 'Произвольные текстовые данные'),
       ('binary', 'Произвольные бинарные данные')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS data_entries
(
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT                   NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    data_type_id   INT                      NOT NULL REFERENCES data_types (id),
    title          VARCHAR(512)             NOT NULL,
    encrypted_data BYTEA                    NOT NULL,
    salt           BYTEA                    NOT NULL,
    metadata       TEXT                     NOT NULL DEFAULT '',
    created_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, title)
);

CREATE INDEX idx_data_entries_user_id ON data_entries (user_id);
CREATE INDEX idx_data_entries_user_title ON data_entries (user_id, title);