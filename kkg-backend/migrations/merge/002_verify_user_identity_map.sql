SELECT 'blog_users' AS name, COUNT(*) AS count FROM users
UNION ALL
SELECT 'legacy_oj_users' AS name, COUNT(*) AS count FROM `user`
UNION ALL
SELECT 'mapped_oj_users' AS name, COUNT(*) AS count FROM user_identity_map WHERE source = 'oj';

SELECT
  ou.id AS legacy_oj_user_id,
  ou.userAccount AS legacy_oj_account,
  m.auth_user_id,
  bu.username,
  bu.email,
  bu.role,
  bu.status
FROM `user` ou
LEFT JOIN user_identity_map m ON m.legacy_oj_user_id = ou.id
LEFT JOIN users bu ON bu.id = m.auth_user_id
ORDER BY ou.id DESC
LIMIT 50;

SELECT 'question_without_auth_user_id' AS name, COUNT(*) AS count FROM question WHERE userId IS NOT NULL AND userId > 0 AND auth_user_id IS NULL
UNION ALL
SELECT 'submit_without_auth_user_id' AS name, COUNT(*) AS count FROM question_submit WHERE userId IS NOT NULL AND userId > 0 AND auth_user_id IS NULL
UNION ALL
SELECT 'solution_without_auth_user_id' AS name, COUNT(*) AS count FROM question_solution_post WHERE userId IS NOT NULL AND userId > 0 AND auth_user_id IS NULL
UNION ALL
SELECT 'agent_task_without_auth_user_id' AS name, COUNT(*) AS count FROM agent_solution_task WHERE triggerUserId IS NOT NULL AND triggerUserId > 0 AND trigger_auth_user_id IS NULL;

