-- watcher_schema.sql · Watcher 表结构（全量）
-- MySQL 8+ · utf8mb4
-- 顺序：IAM → 业务 → 历史

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ========== IAM ==========
CREATE TABLE IF NOT EXISTS `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `username` VARCHAR(64) NOT NULL,
  `password_hash` VARCHAR(255) NOT NULL DEFAULT '',
  `display_name` VARCHAR(64) NOT NULL DEFAULT '',
  `mobile` VARCHAR(32) NOT NULL DEFAULT '',
  `dept` VARCHAR(128) NOT NULL DEFAULT '',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '1=启用',
  `must_change_password` TINYINT(1) NOT NULL DEFAULT 0,
  `ldap_dn` VARCHAR(512) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `roles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `code` VARCHAR(64) NOT NULL,
  `name` VARCHAR(64) NOT NULL DEFAULT '',
  `remark` VARCHAR(255) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_roles_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `user_roles` (
  `user_id` BIGINT UNSIGNED NOT NULL,
  `role_id` BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (`user_id`, `role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `user_client_access` (
  `user_id` BIGINT UNSIGNED NOT NULL,
  `client` VARCHAR(16) NOT NULL COMMENT 'portal|admin|app',
  PRIMARY KEY (`user_id`, `client`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `user_scopes` (
  `user_id` BIGINT UNSIGNED NOT NULL,
  `warehouse_id` VARCHAR(64) NOT NULL,
  PRIMARY KEY (`user_id`, `warehouse_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `refresh_tokens` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `token_hash` VARCHAR(128) NOT NULL,
  `expires_at` DATETIME(3) NULL,
  `revoked` TINYINT(1) NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_refresh_tokens_token_hash` (`token_hash`),
  KEY `idx_refresh_tokens_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `audit_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `actor_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `action` VARCHAR(64) NOT NULL DEFAULT '',
  `resource_type` VARCHAR(64) NOT NULL DEFAULT '',
  `resource_id` VARCHAR(64) NOT NULL DEFAULT '',
  `client` VARCHAR(16) NOT NULL DEFAULT '',
  `ip` VARCHAR(64) NOT NULL DEFAULT '',
  `detail_json` TEXT NULL,
  `created_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  KEY `idx_audit_logs_actor_id` (`actor_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `casbin_rule` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `ptype` VARCHAR(100) NULL,
  `v0` VARCHAR(100) NULL,
  `v1` VARCHAR(100) NULL,
  `v2` VARCHAR(100) NULL,
  `v3` VARCHAR(100) NULL,
  `v4` VARCHAR(100) NULL,
  `v5` VARCHAR(100) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ========== 业务 ==========
CREATE TABLE IF NOT EXISTS `warehouses` (
  `id` VARCHAR(64) NOT NULL,
  `code` VARCHAR(64) NOT NULL DEFAULT '',
  `name` VARCHAR(128) NOT NULL DEFAULT '',
  `address` VARCHAR(255) NOT NULL DEFAULT '',
  `status` TINYINT NOT NULL DEFAULT 1,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_warehouses_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `materials` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `code` VARCHAR(64) NOT NULL DEFAULT '',
  `name` VARCHAR(128) NOT NULL DEFAULT '',
  `waste_code` VARCHAR(64) NOT NULL DEFAULT '',
  `category` VARCHAR(64) NOT NULL DEFAULT '',
  `unit` VARCHAR(16) NOT NULL DEFAULT '',
  `spec` VARCHAR(128) NOT NULL DEFAULT '',
  `status` TINYINT NOT NULL DEFAULT 1,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_materials_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `stock_balances` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `warehouse_id` VARCHAR(64) NOT NULL DEFAULT '',
  `material_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `qty` DECIMAL(18,4) NOT NULL DEFAULT 0,
  `locked_qty` DECIMAL(18,4) NOT NULL DEFAULT 0,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_wh_mat` (`warehouse_id`,`material_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `stock_ledgers` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `warehouse_id` VARCHAR(64) NOT NULL DEFAULT '',
  `material_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `delta` DECIMAL(18,4) NOT NULL DEFAULT 0,
  `balance_after` DECIMAL(18,4) NOT NULL DEFAULT 0,
  `biz_type` VARCHAR(32) NOT NULL DEFAULT '',
  `biz_id` VARCHAR(64) NOT NULL DEFAULT '',
  `unique_code` VARCHAR(64) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  KEY `idx_stock_ledgers_warehouse_id` (`warehouse_id`),
  KEY `idx_stock_ledgers_material_id` (`material_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `unique_codes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `code` VARCHAR(64) NOT NULL DEFAULT '',
  `material_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `warehouse_id` VARCHAR(64) NOT NULL DEFAULT '',
  `status` VARCHAR(16) NOT NULL DEFAULT '',
  `inbound_order_id` BIGINT UNSIGNED NULL,
  `outbound_order_id` BIGINT UNSIGNED NULL,
  `issued_at` DATETIME(3) NULL,
  `stocked_at` DATETIME(3) NULL,
  `signed_at` DATETIME(3) NULL,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_unique_codes_code` (`code`),
  KEY `idx_unique_codes_material_id` (`material_id`),
  KEY `idx_unique_codes_warehouse_id` (`warehouse_id`),
  KEY `idx_unique_codes_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `inbound_orders` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_no` VARCHAR(64) NOT NULL DEFAULT '',
  `warehouse_id` VARCHAR(64) NOT NULL DEFAULT '',
  `status` VARCHAR(16) NOT NULL DEFAULT '',
  `remark` VARCHAR(255) NOT NULL DEFAULT '',
  `manifest_no` VARCHAR(64) NOT NULL DEFAULT '',
  `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_inbound_orders_order_no` (`order_no`),
  KEY `idx_inbound_orders_warehouse_id` (`warehouse_id`),
  KEY `idx_inbound_orders_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `inbound_order_items` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `inbound_order_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `material_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `qty` DECIMAL(18,4) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_inbound_order_items_inbound_order_id` (`inbound_order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `inbound_confirmations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `inbound_order_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `unique_code` VARCHAR(64) NOT NULL DEFAULT '',
  `operator_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `lat` DOUBLE NULL,
  `lng` DOUBLE NULL,
  `photo_key` VARCHAR(255) NOT NULL DEFAULT '',
  `client_request_id` VARCHAR(64) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_inbound_confirmations_client_request_id` (`client_request_id`),
  KEY `idx_inbound_confirmations_inbound_order_id` (`inbound_order_id`),
  KEY `idx_inbound_confirmations_unique_code` (`unique_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `outbound_orders` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_no` VARCHAR(64) NOT NULL DEFAULT '',
  `warehouse_id` VARCHAR(64) NOT NULL DEFAULT '',
  `status` VARCHAR(16) NOT NULL DEFAULT '',
  `remark` VARCHAR(255) NOT NULL DEFAULT '',
  `manifest_no` VARCHAR(64) NOT NULL DEFAULT '',
  `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_outbound_orders_order_no` (`order_no`),
  KEY `idx_outbound_orders_warehouse_id` (`warehouse_id`),
  KEY `idx_outbound_orders_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `outbound_order_items` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `outbound_order_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `material_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `qty` DECIMAL(18,4) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_outbound_order_items_outbound_order_id` (`outbound_order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `outbound_confirmations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `outbound_order_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `unique_code` VARCHAR(64) NOT NULL DEFAULT '',
  `action` VARCHAR(16) NOT NULL DEFAULT '' COMMENT 'lock|receive',
  `operator_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `lat` DOUBLE NULL,
  `lng` DOUBLE NULL,
  `photo_key` VARCHAR(255) NOT NULL DEFAULT '',
  `client_request_id` VARCHAR(64) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_outbound_confirmations_client_request_id` (`client_request_id`),
  KEY `idx_outbound_confirmations_outbound_order_id` (`outbound_order_id`),
  KEY `idx_outbound_confirmations_unique_code` (`unique_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `idempotency_records` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `key_hash` VARCHAR(128) NOT NULL DEFAULT '',
  `response` TEXT NULL,
  `created_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_idempotency_records_key_hash` (`key_hash`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `alarms` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `warehouse_id` VARCHAR(64) NOT NULL DEFAULT '',
  `rule_code` VARCHAR(64) NOT NULL DEFAULT '',
  `level` VARCHAR(16) NOT NULL DEFAULT '',
  `title` VARCHAR(255) NOT NULL DEFAULT '',
  `status` VARCHAR(16) NOT NULL DEFAULT '',
  `metric` VARCHAR(64) NOT NULL DEFAULT '',
  `value` DOUBLE NOT NULL DEFAULT 0,
  `threshold` DOUBLE NOT NULL DEFAULT 0,
  `device_id` VARCHAR(64) NOT NULL DEFAULT '',
  `raised_at` DATETIME(3) NULL,
  `acked_at` DATETIME(3) NULL,
  `acked_by` BIGINT UNSIGNED NULL,
  `closed_at` DATETIME(3) NULL,
  `detail_json` TEXT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_alarms_warehouse_id` (`warehouse_id`),
  KEY `idx_alarms_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `transport_units` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `plate_no` VARCHAR(64) NOT NULL DEFAULT '',
  `type` VARCHAR(32) NOT NULL DEFAULT '',
  `device_id` VARCHAR(64) NOT NULL DEFAULT '',
  `status` TINYINT NOT NULL DEFAULT 1,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_transport_units_plate_no` (`plate_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `transport_tasks` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `task_no` VARCHAR(64) NOT NULL DEFAULT '',
  `outbound_order_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `unit_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `driver_user_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `warehouse_id` VARCHAR(64) NOT NULL DEFAULT '',
  `status` VARCHAR(16) NOT NULL DEFAULT '',
  `remark` VARCHAR(255) NOT NULL DEFAULT '',
  `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_transport_tasks_task_no` (`task_no`),
  KEY `idx_transport_tasks_outbound_order_id` (`outbound_order_id`),
  KEY `idx_transport_tasks_unit_id` (`unit_id`),
  KEY `idx_transport_tasks_driver_user_id` (`driver_user_id`),
  KEY `idx_transport_tasks_warehouse_id` (`warehouse_id`),
  KEY `idx_transport_tasks_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `transport_checkins` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `task_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `node_type` VARCHAR(16) NOT NULL DEFAULT '',
  `unique_code` VARCHAR(64) NOT NULL DEFAULT '',
  `lat` DOUBLE NULL,
  `lng` DOUBLE NULL,
  `accuracy` DOUBLE NULL,
  `photo_key` VARCHAR(255) NOT NULL DEFAULT '',
  `operator_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `client_request_id` VARCHAR(64) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_transport_checkins_client_request_id` (`client_request_id`),
  KEY `idx_transport_checkins_task_id` (`task_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `inventory_tasks` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL DEFAULT '',
  `warehouse_id` VARCHAR(64) NOT NULL DEFAULT '',
  `scope_json` TEXT NULL,
  `status` VARCHAR(16) NOT NULL DEFAULT '',
  `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  KEY `idx_inventory_tasks_warehouse_id` (`warehouse_id`),
  KEY `idx_inventory_tasks_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `inventory_lines` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `task_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `unique_code` VARCHAR(64) NOT NULL DEFAULT '',
  `material_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `scanned` TINYINT(1) NOT NULL DEFAULT 0,
  `scanned_at` DATETIME(3) NULL,
  `operator_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_inventory_lines_task_id` (`task_id`),
  KEY `idx_inventory_lines_unique_code` (`unique_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `approval_templates` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `biz_type` VARCHAR(64) NOT NULL DEFAULT '',
  `name` VARCHAR(128) NOT NULL DEFAULT '',
  `levels_json` TEXT NULL,
  `enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  KEY `idx_approval_templates_biz_type` (`biz_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `approval_orders` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `template_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `biz_type` VARCHAR(64) NOT NULL DEFAULT '',
  `biz_id` VARCHAR(64) NOT NULL DEFAULT '',
  `status` VARCHAR(16) NOT NULL DEFAULT '',
  `current_level` INT NOT NULL DEFAULT 0,
  `applicant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  KEY `idx_approval_orders_template_id` (`template_id`),
  KEY `idx_approval_orders_biz_type` (`biz_type`),
  KEY `idx_approval_orders_biz_id` (`biz_id`),
  KEY `idx_approval_orders_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `approval_actions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `level` INT NOT NULL DEFAULT 0,
  `actor_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `action` VARCHAR(16) NOT NULL DEFAULT '',
  `comment` VARCHAR(512) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  KEY `idx_approval_actions_order_id` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `stock_adjustments` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `warehouse_id` VARCHAR(64) NOT NULL DEFAULT '',
  `material_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `delta` DECIMAL(18,4) NOT NULL DEFAULT 0,
  `reason` VARCHAR(255) NOT NULL DEFAULT '',
  `status` VARCHAR(16) NOT NULL DEFAULT '',
  `approval_order_id` BIGINT UNSIGNED NULL,
  `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  KEY `idx_stock_adjustments_warehouse_id` (`warehouse_id`),
  KEY `idx_stock_adjustments_material_id` (`material_id`),
  KEY `idx_stock_adjustments_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `inspection_plans` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL DEFAULT '',
  `cycle_cron` VARCHAR(64) NOT NULL DEFAULT '',
  `enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `inspection_routes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `plan_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `name` VARCHAR(128) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_inspection_routes_plan_id` (`plan_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `inspection_points` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `route_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `seq` INT NOT NULL DEFAULT 0,
  `name` VARCHAR(128) NOT NULL DEFAULT '',
  `lat` DOUBLE NOT NULL DEFAULT 0,
  `lng` DOUBLE NOT NULL DEFAULT 0,
  `radius_m` DECIMAL(10,2) NOT NULL DEFAULT 50,
  `require_photo` TINYINT(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_inspection_points_route_id` (`route_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `inspection_tasks` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `plan_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `route_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `assignee_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `status` VARCHAR(16) NOT NULL DEFAULT '',
  `due_at` DATETIME(3) NULL,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  KEY `idx_inspection_tasks_plan_id` (`plan_id`),
  KEY `idx_inspection_tasks_route_id` (`route_id`),
  KEY `idx_inspection_tasks_assignee_id` (`assignee_id`),
  KEY `idx_inspection_tasks_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `inspection_checkins` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `task_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `point_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `status` VARCHAR(16) NOT NULL DEFAULT '',
  `lat` DOUBLE NULL,
  `lng` DOUBLE NULL,
  `photo_key` VARCHAR(255) NOT NULL DEFAULT '',
  `remark` VARCHAR(512) NOT NULL DEFAULT '',
  `client_request_id` VARCHAR(64) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_inspection_checkins_client_request_id` (`client_request_id`),
  KEY `idx_inspection_checkins_task_id` (`task_id`),
  KEY `idx_inspection_checkins_point_id` (`point_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `threshold_rules` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `warehouse_id` VARCHAR(64) NOT NULL DEFAULT '',
  `metric` VARCHAR(64) NOT NULL DEFAULT '',
  `upper` DOUBLE NULL,
  `lower` DOUBLE NULL,
  `extra_json` TEXT NULL,
  `version` INT NOT NULL DEFAULT 1,
  `enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  KEY `idx_threshold_rules_warehouse_id` (`warehouse_id`),
  KEY `idx_threshold_rules_metric` (`metric`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `geofences` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `warehouse_id` VARCHAR(64) NOT NULL DEFAULT '',
  `name` VARCHAR(128) NOT NULL DEFAULT '',
  `shape_type` VARCHAR(16) NOT NULL DEFAULT '',
  `geo_json` TEXT NULL,
  `hysteresis_m` DECIMAL(10,2) NOT NULL DEFAULT 0,
  `schedule_json` TEXT NULL,
  `version` INT NOT NULL DEFAULT 1,
  `enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  KEY `idx_geofences_warehouse_id` (`warehouse_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ========== 历史 ==========
CREATE TABLE IF NOT EXISTS `hist_telemetry` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `warehouse_id` VARCHAR(64) NOT NULL DEFAULT '',
  `device_id` VARCHAR(64) NOT NULL DEFAULT '',
  `metric` VARCHAR(64) NOT NULL DEFAULT '',
  `value` DOUBLE NOT NULL DEFAULT 0,
  `ts` DATETIME(3) NULL,
  `msg_id` VARCHAR(64) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_hist_telemetry_msg_id` (`msg_id`),
  KEY `idx_hist_telemetry_warehouse_id` (`warehouse_id`),
  KEY `idx_hist_telemetry_device_id` (`device_id`),
  KEY `idx_hist_telemetry_metric` (`metric`),
  KEY `idx_hist_telemetry_ts` (`ts`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `hist_locations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `warehouse_id` VARCHAR(64) NOT NULL DEFAULT '',
  `device_id` VARCHAR(64) NOT NULL DEFAULT '',
  `unit_id` VARCHAR(64) NOT NULL DEFAULT '',
  `lat` DOUBLE NOT NULL DEFAULT 0,
  `lng` DOUBLE NOT NULL DEFAULT 0,
  `speed` DOUBLE NOT NULL DEFAULT 0,
  `ts` DATETIME(3) NULL,
  `msg_id` VARCHAR(64) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_hist_locations_msg_id` (`msg_id`),
  KEY `idx_hist_locations_warehouse_id` (`warehouse_id`),
  KEY `idx_hist_locations_device_id` (`device_id`),
  KEY `idx_hist_locations_ts` (`ts`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `hist_alarm_snapshots` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `alarm_id` VARCHAR(64) NOT NULL DEFAULT '',
  `status` VARCHAR(16) NOT NULL DEFAULT '',
  `payload` TEXT NULL,
  `ts` DATETIME(3) NULL,
  `msg_id` VARCHAR(64) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_hist_alarm_snapshots_msg_id` (`msg_id`),
  KEY `idx_hist_alarm_snapshots_alarm_id` (`alarm_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


-- ========== 扩展：MQTT Schema / 监管配置 ==========
CREATE TABLE IF NOT EXISTS `mqtt_schema_profiles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `code` VARCHAR(64) NOT NULL DEFAULT '',
  `name` VARCHAR(128) NOT NULL DEFAULT '',
  `topic_pattern` VARCHAR(255) NOT NULL DEFAULT '',
  `mapping_json` TEXT NULL,
  `enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_mqtt_schema_profiles_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `integration_configs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `code` VARCHAR(64) NOT NULL DEFAULT '',
  `name` VARCHAR(128) NOT NULL DEFAULT '',
  `config_json` TEXT NULL,
  `enabled` TINYINT(1) NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_integration_configs_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `devices` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `device_code` VARCHAR(64) NOT NULL DEFAULT '',
  `name` VARCHAR(128) NOT NULL DEFAULT '',
  `type` VARCHAR(32) NOT NULL DEFAULT '',
  `warehouse_id` VARCHAR(64) NOT NULL DEFAULT '',
  `schema_profile_id` BIGINT UNSIGNED NULL,
  `auth_type` VARCHAR(32) NOT NULL DEFAULT 'password',
  `status` VARCHAR(16) NOT NULL DEFAULT 'active',
  `meta_json` TEXT NULL,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_devices_device_code` (`device_code`),
  KEY `idx_devices_warehouse_id` (`warehouse_id`),
  KEY `idx_devices_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `integration_outbox` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `channel` VARCHAR(32) NOT NULL DEFAULT '',
  `event_type` VARCHAR(64) NOT NULL DEFAULT '',
  `biz_type` VARCHAR(64) NOT NULL DEFAULT '',
  `biz_id` VARCHAR(64) NOT NULL DEFAULT '',
  `payload_json` MEDIUMTEXT NULL,
  `status` VARCHAR(16) NOT NULL DEFAULT 'pending',
  `attempts` INT NOT NULL DEFAULT 0,
  `max_attempts` INT NOT NULL DEFAULT 8,
  `next_retry_at` DATETIME(3) NULL,
  `last_error` VARCHAR(512) NOT NULL DEFAULT '',
  `delivered_at` DATETIME(3) NULL,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  PRIMARY KEY (`id`),
  KEY `idx_integration_outbox_channel` (`channel`),
  KEY `idx_integration_outbox_event_type` (`event_type`),
  KEY `idx_integration_outbox_biz_type` (`biz_type`),
  KEY `idx_integration_outbox_biz_id` (`biz_id`),
  KEY `idx_integration_outbox_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
SET FOREIGN_KEY_CHECKS = 1;
