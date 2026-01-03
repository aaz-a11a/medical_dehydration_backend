-- 1) USERS
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    login VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    is_moderator BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- seed demo user if missing (for user_id=1 flow)
INSERT INTO users (login, password_hash, is_moderator)
SELECT 'user1', 'hash-placeholder', FALSE
WHERE NOT EXISTS (SELECT 1 FROM users WHERE login = 'user1');

-- 2) SYMPTOMS (симптомы)
CREATE TABLE IF NOT EXISTS symptoms (
    id SERIAL PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    category VARCHAR(100),
    description TEXT,
    severity VARCHAR(50),
    weight_loss VARCHAR(50),
    fluid_need VARCHAR(50),
    recovery_time VARCHAR(50),
    image_url VARCHAR(200),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- DELETE ALL EXISTING SYMPTOMS AND INSERT ONLY 5 NEEDED ONES
DELETE FROM symptoms;

-- seed exactly 5 symptoms
INSERT INTO symptoms (title, category, description, severity, weight_loss, fluid_need, recovery_time, image_url, is_active) VALUES
('Жажда', 'Ранние признаки', 'Неутолимое желание пить воду, часто сопровождающееся сухостью во рту', 'Легкая (1-2%)', '1-2% массы тела', '30-50 мл/кг', '2-4 часа', '1.png', true),
('Эластичность кожи', 'Объективные признаки', 'Снижение тургора кожи — кожная складка расправляется медленно', 'Средняя (3-6%)', '3-6% массы тела', '50-70 мл/кг', '6-12 часов', '2.png', true),
('Судороги', 'Тяжелые признаки', 'Непроизвольные болезненные сокращения мышц при потере электролитов', 'Тяжелая (7-9%)', '7-9% массы тела', '70-100 мл/кг', '12-24 часа', '3.png', true),
('Состояние глазных яблок', 'Объективные признаки', 'Глаза выглядят запавшими, с темными кругами, снижение слезоотделения', 'Тяжелая (7-9%)', '7-9% массы тела', '70-100 мл/кг', '12-24 часа', '4.png', true),
('Диурез', 'Объективные признаки', 'Снижение объема мочи, концентрированная моча', 'Средняя (3-6%)', '3-6% массы тела', '50-70 мл/кг', '6-12 часов', '5.png', true);

-- 3) REQUESTS (заявки)
CREATE TABLE IF NOT EXISTS dehydration_requests (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    formed_at TIMESTAMP,
    completed_at TIMESTAMP,
    moderator_id INTEGER,
    patient_weight DECIMAL(5,2),
    dehydration_percent DECIMAL(4,2),
    fluid_deficit DECIMAL(6,2),
    doctor_comment TEXT,
    CONSTRAINT dehydration_requests_status_check
        CHECK (status IN ('черновик', 'удален', 'удалён', 'сформирован', 'завершен', 'отклонен'))
);

-- FK to users without cascade
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_dehydration_requests_user_id'
    ) THEN
        ALTER TABLE dehydration_requests
        ADD CONSTRAINT fk_dehydration_requests_user_id
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT;
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_dehydration_requests_moderator_id'
    ) THEN
        ALTER TABLE dehydration_requests
        ADD CONSTRAINT fk_dehydration_requests_moderator_id
        FOREIGN KEY (moderator_id) REFERENCES users(id) ON DELETE RESTRICT;
    END IF;
END$$;

-- No more than one draft per user
CREATE UNIQUE INDEX IF NOT EXISTS one_draft_per_user ON dehydration_requests (user_id) WHERE status = 'черновик';

-- 4) M-M REQUEST_SYMPTOMS
CREATE TABLE IF NOT EXISTS request_symptoms (
    request_id INTEGER NOT NULL,
    symptom_id INTEGER NOT NULL,
    intensity INTEGER CHECK (intensity BETWEEN 1 AND 10),
    is_main BOOLEAN DEFAULT FALSE,
    comment TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id, symptom_id),
    FOREIGN KEY (request_id) REFERENCES dehydration_requests(id) ON DELETE RESTRICT,
    FOREIGN KEY (symptom_id) REFERENCES symptoms(id) ON DELETE RESTRICT
);

-- +goose Down
DROP TABLE IF EXISTS request_symptoms;
DROP INDEX IF EXISTS one_draft_per_user;
DROP TABLE IF EXISTS dehydration_requests;
DROP TABLE IF EXISTS symptoms;
DROP TABLE IF EXISTS users;

