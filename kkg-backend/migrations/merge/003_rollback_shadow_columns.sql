-- Rollback only shadow columns and mapping table.
-- This does not delete users created from legacy OJ rows, because those may have
-- been used after the merge. Restore a backup for a full rollback.

DELIMITER //
CREATE PROCEDURE drop_merge_index_if_exists(
  IN p_table_name VARCHAR(128),
  IN p_index_name VARCHAR(128)
)
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = p_table_name
      AND index_name = p_index_name
  ) THEN
    SET @ddl = CONCAT('ALTER TABLE ', p_table_name, ' DROP INDEX ', p_index_name);
    PREPARE stmt FROM @ddl;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END IF;
END//
DELIMITER ;

CALL drop_merge_index_if_exists('question', 'idx_question_auth_user_id');
CALL drop_merge_index_if_exists('question_submit', 'idx_question_submit_auth_user_id');
CALL drop_merge_index_if_exists('question_solution_post', 'idx_question_solution_auth_user_id');
CALL drop_merge_index_if_exists('agent_solution_task', 'idx_agent_solution_trigger_auth_user_id');

DROP PROCEDURE drop_merge_index_if_exists;

ALTER TABLE question DROP COLUMN IF EXISTS auth_user_id;
ALTER TABLE question_submit DROP COLUMN IF EXISTS auth_user_id;
ALTER TABLE question_solution_post DROP COLUMN IF EXISTS auth_user_id;
ALTER TABLE agent_solution_task DROP COLUMN IF EXISTS trigger_auth_user_id;

DROP TABLE IF EXISTS user_identity_map;
