CREATE INDEX IF NOT EXISTS idx_submissions_user_task_accepted 
ON submissions (user_id, task_id) 
WHERE status = 'ACCEPTED';

CREATE INDEX IF NOT EXISTS idx_submissions_user_id_created_at 
ON submissions (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_task_tests_task_id 
ON task_tests (task_id);