-- Non-destructive Blog + OJ user merge preparation.
-- Run this after a full mysqldump backup.

CREATE TABLE IF NOT EXISTS user_identity_map (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  auth_user_id BIGINT UNSIGNED NOT NULL,
  legacy_oj_user_id BIGINT NULL,
  legacy_oj_account VARCHAR(128) NOT NULL DEFAULT '',
  source VARCHAR(32) NOT NULL DEFAULT 'oj',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_user_identity_auth_source (auth_user_id, source),
  UNIQUE KEY uk_user_identity_oj_user (legacy_oj_user_id),
  KEY idx_user_identity_oj_account (legacy_oj_account)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Create unified users for OJ accounts that do not already exist in Blog users.
-- Hidden/deleted OJ users are preserved as disabled unified users instead of dropped.
-- Existing Blog users are matched by username or email.
INSERT INTO users (username, email, avatar_url, password_hash, role, status, created_at, updated_at)
SELECT
  NULLIF(TRIM(ou.userAccount), '') AS username,
  CASE
    WHEN ou.userAccount LIKE '%@%' THEN TRIM(ou.userAccount)
    ELSE CONCAT(TRIM(ou.userAccount), '@kkgoj.local')
  END AS email,
  COALESCE(ou.userAvatar, '') AS avatar_url,
  CASE
    WHEN COALESCE(ou.userPassword, '') <> '' THEN CONCAT('$legacy_oj_md5$', ou.userPassword)
    ELSE '$legacy_oj_empty$'
  END AS password_hash,
  CASE
    WHEN ou.userRole IN ('user', 'admin', 'super_admin') THEN ou.userRole
    WHEN ou.userRole = 'ban' THEN 'user'
    ELSE 'user'
  END AS role,
  CASE
    WHEN ou.isDelete <> 0 OR ou.userRole = 'ban' THEN 0
    ELSE 1
  END AS status,
  COALESCE(ou.createTime, CURRENT_TIMESTAMP) AS created_at,
  COALESCE(ou.updateTime, CURRENT_TIMESTAMP) AS updated_at
FROM `user` ou
LEFT JOIN users bu
  ON bu.username = TRIM(ou.userAccount)
  OR bu.email = TRIM(ou.userAccount)
  OR bu.email = CONCAT(TRIM(ou.userAccount), '@kkgoj.local')
WHERE TRIM(COALESCE(ou.userAccount, '')) <> ''
  AND bu.id IS NULL;

-- Map every legacy OJ user to the unified users row.
INSERT INTO user_identity_map (auth_user_id, legacy_oj_user_id, legacy_oj_account, source)
SELECT
  bu.id AS auth_user_id,
  ou.id AS legacy_oj_user_id,
  TRIM(ou.userAccount) AS legacy_oj_account,
  'oj' AS source
FROM `user` ou
JOIN users bu
  ON bu.username = TRIM(ou.userAccount)
  OR bu.email = TRIM(ou.userAccount)
  OR bu.email = CONCAT(TRIM(ou.userAccount), '@kkgoj.local')
WHERE TRIM(COALESCE(ou.userAccount, '')) <> ''
ON DUPLICATE KEY UPDATE
  auth_user_id = VALUES(auth_user_id),
  legacy_oj_account = VALUES(legacy_oj_account),
  updated_at = CURRENT_TIMESTAMP;

-- Add shadow unified-user columns to OJ business tables.
DROP PROCEDURE IF EXISTS add_merge_column_if_missing;
DELIMITER //
CREATE PROCEDURE add_merge_column_if_missing(
  IN p_table_name VARCHAR(128),
  IN p_column_name VARCHAR(128),
  IN p_column_definition VARCHAR(255)
)
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = p_table_name
      AND column_name = p_column_name
  ) THEN
    SET @ddl = CONCAT('ALTER TABLE ', p_table_name, ' ADD COLUMN ', p_column_name, ' ', p_column_definition);
    PREPARE stmt FROM @ddl;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END IF;
END//
DELIMITER ;

CALL add_merge_column_if_missing('question', 'auth_user_id', 'BIGINT UNSIGNED NULL');
CALL add_merge_column_if_missing('question_submit', 'auth_user_id', 'BIGINT UNSIGNED NULL');
CALL add_merge_column_if_missing('question_solution_post', 'auth_user_id', 'BIGINT UNSIGNED NULL');
CALL add_merge_column_if_missing('agent_solution_task', 'trigger_auth_user_id', 'BIGINT UNSIGNED NULL');

DROP PROCEDURE add_merge_column_if_missing;

UPDATE question q
JOIN user_identity_map m ON m.legacy_oj_user_id = q.userId
SET q.auth_user_id = m.auth_user_id
WHERE q.auth_user_id IS NULL;

UPDATE question_submit s
JOIN user_identity_map m ON m.legacy_oj_user_id = s.userId
SET s.auth_user_id = m.auth_user_id
WHERE s.auth_user_id IS NULL;

UPDATE question_solution_post sp
JOIN user_identity_map m ON m.legacy_oj_user_id = sp.userId
SET sp.auth_user_id = m.auth_user_id
WHERE sp.auth_user_id IS NULL;

UPDATE agent_solution_task t
JOIN user_identity_map m ON m.legacy_oj_user_id = t.triggerUserId
SET t.trigger_auth_user_id = m.auth_user_id
WHERE t.trigger_auth_user_id IS NULL;

DROP PROCEDURE IF EXISTS add_merge_index_if_missing;
DELIMITER //
CREATE PROCEDURE add_merge_index_if_missing(
  IN p_table_name VARCHAR(128),
  IN p_index_name VARCHAR(128),
  IN p_column_name VARCHAR(128)
)
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = p_table_name
      AND index_name = p_index_name
  ) THEN
    SET @ddl = CONCAT('CREATE INDEX ', p_index_name, ' ON ', p_table_name, ' (', p_column_name, ')');
    PREPARE stmt FROM @ddl;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END IF;
END//
DELIMITER ;

CALL add_merge_index_if_missing('question', 'idx_question_auth_user_id', 'auth_user_id');
CALL add_merge_index_if_missing('question_submit', 'idx_question_submit_auth_user_id', 'auth_user_id');
CALL add_merge_index_if_missing('question_solution_post', 'idx_question_solution_auth_user_id', 'auth_user_id');
CALL add_merge_index_if_missing('agent_solution_task', 'idx_agent_solution_trigger_auth_user_id', 'trigger_auth_user_id');

DROP PROCEDURE add_merge_index_if_missing;
