CREATE TABLE IF NOT EXISTS users
(
    id         BIGINT       NOT NULL PRIMARY KEY COMMENT 'Telegram user ID',
    username   VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS habits
(
    id                  BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id             BIGINT          NOT NULL,
    name                VARCHAR(255)    NOT NULL,
    origin_at           DATETIME        NOT NULL COMMENT 'Fixed origin point set by user at creation, never changes',
    last_relapse_at     DATETIME        NOT NULL COMMENT 'Updated on every registered relapse',
    cost_per_relapse    DECIMAL(10, 2)  NOT NULL DEFAULT 0,
    avg_relapses_count  DECIMAL(10, 4)  NOT NULL DEFAULT 1,
    avg_relapses_period ENUM ('day','month','3month','6month','year') NOT NULL DEFAULT 'day',
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_habits_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS relapses
(
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    habit_id    BIGINT   NOT NULL,
    relapsed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_relapses_habit FOREIGN KEY (habit_id) REFERENCES habits (id) ON DELETE CASCADE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;
