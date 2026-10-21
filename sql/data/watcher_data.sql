-- watcher_data.sql · Watcher 表数据（默认种子）
-- 角色/权限、管理员 admin/Admin@123、默认仓 WH01、审批模板
-- 依赖：sql/schema/watcher_schema.sql

SET NAMES utf8mb4;

-- ========== 角色与权限 ==========
INSERT INTO `roles` (`id`, `code`, `name`, `remark`, `created_at`, `updated_at`) VALUES
  (1, 'admin', '系统管理员', '内置：拥有全部权限点', NOW(3), NOW(3)),
  (2, 'readonly', '只读观察员', '内置：Portal/APP 只读能力', NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `remark` = VALUES(`remark`),
  `updated_at` = NOW(3);

DELETE FROM `casbin_rule`
 WHERE (`ptype` = 'p' AND `v0` IN ('role:admin', 'role:readonly'))
    OR (`ptype` = 'g' AND `v0` = 'user:1' AND `v1` = 'role:admin');

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES
  ('p', 'role:admin', 'portal', 'access', '', '', ''),
  ('p', 'role:admin', 'admin', 'access', '', '', ''),
  ('p', 'role:admin', 'app', 'access', '', '', ''),
  ('p', 'role:admin', 'user', 'view', '', '', ''),
  ('p', 'role:admin', 'user', 'edit', '', '', ''),
  ('p', 'role:admin', 'role', 'view', '', '', ''),
  ('p', 'role:admin', 'role', 'edit', '', '', ''),
  ('p', 'role:admin', 'audit', 'view', '', '', ''),
  ('p', 'role:admin', 'warehouse', 'view', '', '', ''),
  ('p', 'role:admin', 'inbound', 'edit', '', '', ''),
  ('p', 'role:admin', 'inbound', 'confirm', '', '', ''),
  ('p', 'role:admin', 'outbound', 'edit', '', '', ''),
  ('p', 'role:admin', 'outbound', 'confirm', '', '', ''),
  ('p', 'role:admin', 'code', 'issue', '', '', ''),
  ('p', 'role:admin', 'material', 'view', '', '', ''),
  ('p', 'role:admin', 'material', 'edit', '', '', ''),
  ('p', 'role:admin', 'material', 'export', '', '', ''),
  ('p', 'role:admin', 'stock', 'adjust', '', '', ''),
  ('p', 'role:admin', 'approval', 'view', '', '', ''),
  ('p', 'role:admin', 'approval', 'act', '', '', ''),
  ('p', 'role:admin', 'inventory', 'view', '', '', ''),
  ('p', 'role:admin', 'inventory', 'create', '', '', ''),
  ('p', 'role:admin', 'inventory', 'scan', '', '', ''),
  ('p', 'role:admin', 'inventory', 'adjust', '', '', ''),
  ('p', 'role:admin', 'alarm', 'view', '', '', ''),
  ('p', 'role:admin', 'alarm', 'ack', '', '', ''),
  ('p', 'role:admin', 'threshold', 'edit', '', '', ''),
  ('p', 'role:admin', 'transport', 'view', '', '', ''),
  ('p', 'role:admin', 'transport', 'edit', '', '', ''),
  ('p', 'role:admin', 'unit', 'edit', '', '', ''),
  ('p', 'role:admin', 'checkin', 'exec', '', '', ''),
  ('p', 'role:admin', 'track', 'view', '', '', ''),
  ('p', 'role:admin', 'geofence', 'view', '', '', ''),
  ('p', 'role:admin', 'geofence', 'edit', '', '', ''),
  ('p', 'role:admin', 'monitor', 'view', '', '', ''),
  ('p', 'role:admin', 'device', 'edit', '', '', ''),
  ('p', 'role:admin', 'mqtt', 'schema:edit', '', '', ''),
  ('p', 'role:admin', 'openapi', 'config', '', '', ''),
  ('p', 'role:admin', 'inspection', 'view', '', '', ''),
  ('p', 'role:admin', 'inspection', 'create', '', '', ''),
  ('p', 'role:admin', 'inspection', 'exec', '', '', '');

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES
  ('p', 'role:readonly', 'portal', 'access', '', '', ''),
  ('p', 'role:readonly', 'app', 'access', '', '', ''),
  ('p', 'role:readonly', 'material', 'view', '', '', ''),
  ('p', 'role:readonly', 'alarm', 'view', '', '', ''),
  ('p', 'role:readonly', 'monitor', 'view', '', '', ''),
  ('p', 'role:readonly', 'track', 'view', '', '', ''),
  ('p', 'role:readonly', 'geofence', 'view', '', '', ''),
  ('p', 'role:readonly', 'inventory', 'view', '', '', ''),
  ('p', 'role:readonly', 'inspection', 'view', '', '', ''),
  ('p', 'role:readonly', 'warehouse', 'view', '', '', '');

-- ========== 管理员 ==========
INSERT INTO `users` (
  `id`, `username`, `password_hash`, `display_name`, `mobile`, `dept`,
  `status`, `must_change_password`, `ldap_dn`, `created_at`, `updated_at`
) VALUES (
  1,
  'admin',
  '$2a$10$LIRSh/Z0MlKgYTDHAWIIeOKANSLWu8wo3ddWWglBixYRjKZatFyR6',
  '系统管理员',
  '',
  '',
  1,
  0,
  '',
  NOW(3),
  NOW(3)
) ON DUPLICATE KEY UPDATE
  `password_hash` = VALUES(`password_hash`),
  `display_name` = VALUES(`display_name`),
  `status` = 1,
  `updated_at` = NOW(3);

INSERT INTO `user_roles` (`user_id`, `role_id`) VALUES (1, 1)
ON DUPLICATE KEY UPDATE `role_id` = VALUES(`role_id`);

INSERT INTO `user_client_access` (`user_id`, `client`) VALUES
  (1, 'portal'),
  (1, 'admin'),
  (1, 'app')
ON DUPLICATE KEY UPDATE `client` = VALUES(`client`);

INSERT INTO `user_scopes` (`user_id`, `warehouse_id`) VALUES (1, 'WH01')
ON DUPLICATE KEY UPDATE `warehouse_id` = VALUES(`warehouse_id`);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES
  ('g', 'user:1', 'role:admin', '', '', '', '')
ON DUPLICATE KEY UPDATE `v1` = VALUES(`v1`);

-- ========== 默认仓与审批 ==========
INSERT INTO `warehouses` (`id`, `code`, `name`, `address`, `status`, `created_at`, `updated_at`) VALUES
  ('WH01', 'WH01', '默认暂存仓', '', 1, NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `status` = 1,
  `updated_at` = NOW(3);

INSERT INTO `approval_templates` (`id`, `biz_type`, `name`, `levels_json`, `enabled`, `created_at`, `updated_at`) VALUES
  (
    1,
    'stock_adjust',
    '库存调整默认两级',
    '[{"level":1,"roleCodes":["admin"],"strategy":"any"},{"level":2,"roleCodes":["admin"],"strategy":"any"}]',
    1,
    NOW(3),
    NOW(3)
  )
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `levels_json` = VALUES(`levels_json`),
  `enabled` = 1,
  `updated_at` = NOW(3);
