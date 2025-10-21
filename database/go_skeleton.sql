/*
 Navicat Premium Data Transfer

 Source Server         : localhost
 Source Server Type    : MySQL
 Source Server Version : 80033
 Source Host           : localhost:3306
 Source Schema         : go_skeleton

 Target Server Type    : MySQL
 Target Server Version : 80033
 File Encoding         : 65001

 Date: 21/10/2025 15:17:29
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for cn_sys_admin
-- ----------------------------
DROP TABLE IF EXISTS `cn_sys_admin`;
CREATE TABLE `cn_sys_admin` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `uuid` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '唯一id号',
  `dept_id` bigint unsigned NOT NULL COMMENT '部门ID',
  `nick_name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '昵称',
  `real_name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '真实姓名',
  `desc` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '描述',
  `gender` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '性别 1:男 2:女 0:未知',
  `account` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '账号',
  `password` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '密码',
  `phone` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '手机号',
  `email` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '邮箱',
  `avatar` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '头像',
  `salt` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '密码',
  `role_ids` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '角色IDs',
  `type` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '类型：1：平台 2：商家 3：代理商',
  `is_main` tinyint unsigned NOT NULL DEFAULT '2' COMMENT '是否主账号 1：是 2：不是',
  `is_auth` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '是否认证 1:未认证 2:已认证',
  `mfa_secret` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'mfa密钥',
  `status` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '状态 1：正常 2：禁用',
  `created_by` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '创建者',
  `created_at` int unsigned NOT NULL,
  `updated_by` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '编辑者',
  `updated_at` int unsigned NOT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `account_index` (`account`) USING BTREE,
  KEY `dept_id_index` (`dept_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='后台管理员表';

INSERT INTO `cn_sys_admin` (`id`, `uuid`, `dept_id`, `nick_name`, `real_name`, `desc`, `gender`, `account`, `password`, `phone`, `email`, `avatar`, `salt`, `role_ids`, `type`, `is_main`, `is_auth`, `mfa_secret`, `status`, `created_by`, `created_at`, `updated_by`, `updated_at`) VALUES (1, '1859045070490324993', 1, 'admin', 'admin', '', 2, 'admin', 'acce31f66e319f31f7b3c603cb76dd3ee1abd6bde53fcecef7fc61a35186138f', '13595026776', '', '', '02e443a7-9ff6-8b81-1823-b141b318', '1', 1, 1, 2, 'FWQQ5ESYDYXGV5HAXDFA27P3L4JU7DRQ', 1, '', 1688191036, '', 1752639506);

-- ----------------------------
-- Table structure for cn_sys_resource
-- ----------------------------
DROP TABLE IF EXISTS `cn_sys_resource`;
CREATE TABLE `cn_sys_resource` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '名称',
  `alias` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '别名',
  `desc` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '描述',
  `f_url` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '前端路由',
  `b_url` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '后端接口',
  `redirect` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '重定向地址',
  `comp_path` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '组件路径',
  `icon` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '菜单icon',
  `c_icon` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '自定义icon(优先)',
  `parent_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '父级ID',
  `path` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'ID路径 1-2-3...',
  `resource_type` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '类型 1：目录 2：菜单 3：操作',
  `is_hidden` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '是否隐藏 1:是 0：不是',
  `is_cache` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '是否缓存 1:是 0：不是',
  `is_external` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '是否外链 1:是 0:不是',
  `always_show` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '总是显示 1:是 0:不是',
  `breadcrumb_show` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '面包屑显示 1:是 0:不是',
  `is_affix` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '是否固定在tab栏 1：是 0：不是',
  `res_type` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '资源类型：1：平台型 2：商家型 3：代理商型 4：通用型',
  `status` tinyint unsigned NOT NULL COMMENT '状态：1：正常 2：停用',
  `sort_order` smallint unsigned NOT NULL DEFAULT '50' COMMENT '排序',
  `created_at` bigint unsigned NOT NULL,
  `updated_at` bigint unsigned NOT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `backend_menu_alias_title_unique` (`alias`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='后台配置资源菜单表';

-- ----------------------------
-- Table structure for cn_sys_role
-- ----------------------------
DROP TABLE IF EXISTS `cn_sys_role`;
CREATE TABLE `cn_sys_role` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `mch_id` bigint unsigned NOT NULL COMMENT '商家ID 0:非商家',
  `name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '角色名称',
  `code` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '角色唯一code',
  `desc` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '角色描述',
  `is_sys` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '是否系统角色 1：是 0：否',
  `role_type` tinyint unsigned NOT NULL COMMENT '角色类型：1：平台类型 2：商家类型 3：代理商类型',
  `status` tinyint unsigned NOT NULL COMMENT '状态：1正常(默认) 2停用',
  `created_at` int unsigned NOT NULL,
  `updated_at` int unsigned NOT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `name` (`name`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色表';

INSERT INTO `cn_sys_role` (`id`, `mch_id`, `name`, `code`, `desc`, `is_sys`, `role_type`, `status`, `created_at`, `updated_at`) VALUES (1, 0, '超级管理员', 'role_superadmin', '超级管理员', 1, 1, 1, 1645932052, 1689672175);

-- ----------------------------
-- Table structure for cn_sys_role_auth
-- ----------------------------
DROP TABLE IF EXISTS `cn_sys_role_auth`;
CREATE TABLE `cn_sys_role_auth` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `role_id` bigint unsigned NOT NULL COMMENT '角色ID',
  `resource_ids` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '菜单id列表 1,2,3...',
  PRIMARY KEY (`id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色权限表';

-- ----------------------------
-- Table structure for cn_sys_casbin_rule
-- ----------------------------
DROP TABLE IF EXISTS `cn_sys_casbin_rule`;
CREATE TABLE `cn_sys_casbin_rule` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `ptype` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `v0` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `v1` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `v2` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `v3` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `v4` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `v5` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `v6` varchar(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `v7` varchar(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `idx_casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色菜单权限表';

INSERT INTO `cn_sys_casbin_rule` (`id`, `ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`, `v6`, `v7`) VALUES (1, 'p', '1', '/backend/auth/*', 'GET,POST', '', '', '', '', '');
INSERT INTO `cn_sys_casbin_rule` (`id`, `ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`, `v6`, `v7`) VALUES (2, 'p', '1', '/backend/attachment/*', 'GET,POST', '', '', '', '', '');
INSERT INTO `cn_sys_casbin_rule` (`id`, `ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`, `v6`, `v7`) VALUES (5, 'p', '1', '/backend/role/*', 'GET,POST', '', '', '', '', '');
INSERT INTO `cn_sys_casbin_rule` (`id`, `ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`, `v6`, `v7`) VALUES (6, 'p', '1', '/backend/menu/*', 'GET,POST', '', '', '', '', '');
INSERT INTO `cn_sys_casbin_rule` (`id`, `ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`, `v6`, `v7`) VALUES (7, 'p', '1', '/backend/permission/*', 'GET,POST', '', '', '', '', '');
INSERT INTO `cn_sys_casbin_rule` (`id`, `ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`, `v6`, `v7`) VALUES (185, 'p', '1', '/backend/admin/*', 'GET,POST', '', '', '', '', '');

-- ----------------------------
-- Table structure for cn_attachment
-- ----------------------------
DROP TABLE IF EXISTS `cn_attachment`;
CREATE TABLE `cn_attachment` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '附件上传的用户id',
  `attach_name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '附件新名称',
  `attach_origin_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '附件原名称',
  `attach_url` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '附件地址',
  `attach_type` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '附件类型 1：图片 2：视频 3：文件',
  `attach_mimetype` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '附件mime类型',
  `attach_extension` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '附件后缀名',
  `attach_size` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '附件大小',
  `status` tinyint unsigned NOT NULL COMMENT '状态 1：正常 0：删除',
  `created_at` int unsigned NOT NULL DEFAULT '0',
  `updated_at` int unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='附件表';

SET FOREIGN_KEY_CHECKS = 1;
