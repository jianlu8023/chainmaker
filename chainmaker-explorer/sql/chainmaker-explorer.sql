-- --------------------------------------------------------
-- 主机:                           192.168.58.110
-- 服务器版本:                        8.4.2 - MySQL Community Server - GPL
-- 服务器操作系统:                      Linux
-- HeidiSQL 版本:                  12.8.0.6908
-- --------------------------------------------------------

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET NAMES utf8 */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;


-- 导出 chainmaker_explorer_dev 的数据库结构
DROP DATABASE IF EXISTS `chainmaker_explorer_dev`;
CREATE DATABASE IF NOT EXISTS `chainmaker_explorer_dev` /*!40100 DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci */ /*!80016 DEFAULT ENCRYPTION='N' */;
USE `chainmaker_explorer_dev`;

-- 导出  表 chainmaker_explorer_dev.cmb_block 结构
DROP TABLE IF EXISTS `cmb_block`;
CREATE TABLE IF NOT EXISTS `cmb_block` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `pre_block_hash` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `chain_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `block_hash` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `block_height` bigint unsigned DEFAULT NULL,
  `block_version` int unsigned DEFAULT NULL,
  `org_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `timestamp` bigint DEFAULT NULL,
  `block_dag` longtext COLLATE utf8mb4_unicode_ci,
  `dag_hash` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tx_count` int DEFAULT NULL,
  `signature` longtext COLLATE utf8mb4_unicode_ci,
  `rw_set_hash` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tx_root_hash` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `proposer_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `proposer_addr` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `consensus_args` longtext COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `chain_id_block_height_index` (`chain_id`,`block_height`),
  KEY `block_hash_index` (`block_hash`),
  KEY `block_height_index` (`block_height`),
  KEY `timestamp_index` (`timestamp`),
  KEY `block_chain_id_index` (`chain_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 正在导出表  chainmaker_explorer_dev.cmb_block 的数据：~0 rows (大约)
DELETE FROM `cmb_block`;

-- 导出  表 chainmaker_explorer_dev.cmb_chain 结构
DROP TABLE IF EXISTS `cmb_chain`;
CREATE TABLE IF NOT EXISTS `cmb_chain` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `chain_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `chain_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `consensus` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tx_timestamp_verify` tinyint(1) DEFAULT NULL,
  `tx_timeout` int DEFAULT NULL,
  `block_tx_capacity` int DEFAULT NULL,
  `block_size` int DEFAULT NULL,
  `block_interval` int DEFAULT NULL,
  `hash_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `auth_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `chain_chain_id_uq_index` (`chain_id`),
  KEY `chain_chain_id_index` (`chain_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 正在导出表  chainmaker_explorer_dev.cmb_chain 的数据：~0 rows (大约)
DELETE FROM `cmb_chain`;

-- 导出  表 chainmaker_explorer_dev.cmb_contract 结构
DROP TABLE IF EXISTS `cmb_contract`;
CREATE TABLE IF NOT EXISTS `cmb_contract` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `chain_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `addr` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `runtime_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `mgmt_params` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `contract_status` int DEFAULT NULL,
  `block_height` bigint unsigned DEFAULT NULL,
  `org_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `create_tx_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `creator` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `creator_addr` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `create_timestamp` bigint DEFAULT NULL,
  `upgrade_user` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `upgrade_timestamp` bigint DEFAULT NULL,
  `tx_num` bigint DEFAULT NULL,
  `timestamp` bigint DEFAULT NULL,
  `contract_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `contract_symbol` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `total_supply` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `chain_id_name_version_index` (`chain_id`,`name`,`version`),
  KEY `chain_id_index` (`chain_id`),
  KEY `name_index` (`name`),
  KEY `create_timestamp_index` (`create_timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 正在导出表  chainmaker_explorer_dev.cmb_contract 的数据：~0 rows (大约)
DELETE FROM `cmb_contract`;

-- 导出  表 chainmaker_explorer_dev.cmb_contract_event 结构
DROP TABLE IF EXISTS `cmb_contract_event`;
CREATE TABLE IF NOT EXISTS `cmb_contract_event` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `chain_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `topic` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tx_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `contract_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `contract_version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `event_data` mediumblob,
  `timestamp` bigint DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 正在导出表  chainmaker_explorer_dev.cmb_contract_event 的数据：~0 rows (大约)
DELETE FROM `cmb_contract_event`;

-- 导出  表 chainmaker_explorer_dev.cmb_node 结构
DROP TABLE IF EXISTS `cmb_node`;
CREATE TABLE IF NOT EXISTS `cmb_node` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `node_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `node_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `org_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `role` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `address` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `block_height` bigint unsigned DEFAULT NULL,
  `status` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `node_id_index` (`node_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 正在导出表  chainmaker_explorer_dev.cmb_node 的数据：~0 rows (大约)
DELETE FROM `cmb_node`;

-- 导出  表 chainmaker_explorer_dev.cmb_node2chain 结构
DROP TABLE IF EXISTS `cmb_node2chain`;
CREATE TABLE IF NOT EXISTS `cmb_node2chain` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `node_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `chain_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `ref_node_id_chain_id_index` (`node_id`,`chain_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 正在导出表  chainmaker_explorer_dev.cmb_node2chain 的数据：~0 rows (大约)
DELETE FROM `cmb_node2chain`;

-- 导出  表 chainmaker_explorer_dev.cmb_org 结构
DROP TABLE IF EXISTS `cmb_org`;
CREATE TABLE IF NOT EXISTS `cmb_org` (
  `chain_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `org_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` int DEFAULT NULL,
  `id` bigint NOT NULL AUTO_INCREMENT,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `chain_id_org_id_index` (`chain_id`,`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 正在导出表  chainmaker_explorer_dev.cmb_org 的数据：~0 rows (大约)
DELETE FROM `cmb_org`;

-- 导出  表 chainmaker_explorer_dev.cmb_subscribe 结构
DROP TABLE IF EXISTS `cmb_subscribe`;
CREATE TABLE IF NOT EXISTS `cmb_subscribe` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `chain_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `org_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `user_key` text COLLATE utf8mb4_unicode_ci,
  `user_cert` text COLLATE utf8mb4_unicode_ci,
  `nodes_list` text COLLATE utf8mb4_unicode_ci,
  `status` int DEFAULT NULL,
  `auth_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `hash_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `node_ca_cert` text COLLATE utf8mb4_unicode_ci,
  `tls` tinyint(1) DEFAULT NULL,
  `tls_host` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `remote` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `chain_id_index` (`chain_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 正在导出表  chainmaker_explorer_dev.cmb_subscribe 的数据：~0 rows (大约)
DELETE FROM `cmb_subscribe`;

-- 导出  表 chainmaker_explorer_dev.cmb_transaction 结构
DROP TABLE IF EXISTS `cmb_transaction`;
CREATE TABLE IF NOT EXISTS `cmb_transaction` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `chain_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tx_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `org_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `sender` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `block_height` bigint unsigned DEFAULT NULL,
  `block_hash` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tx_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `timestamp` bigint DEFAULT NULL,
  `expiration_time` bigint DEFAULT NULL,
  `tx_index` int DEFAULT NULL,
  `tx_status_code` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tx_hash` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `rw_set_hash` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `contract_result_code` int unsigned DEFAULT NULL,
  `contract_result` mediumblob,
  `contract_message` blob,
  `contract_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `contract_method` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `contract_parameters` longtext COLLATE utf8mb4_unicode_ci,
  `contract_version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `contract_runtime_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `contract_byte_code` mediumblob,
  `endorsement` longtext COLLATE utf8mb4_unicode_ci,
  `sequence` bigint unsigned DEFAULT NULL,
  `user_addr` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `read_set` longtext COLLATE utf8mb4_unicode_ci,
  `write_set` longtext COLLATE utf8mb4_unicode_ci,
  `payer` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `event` longtext COLLATE utf8mb4_unicode_ci,
  `gas_used` bigint unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `tx_id_chain_id_index` (`chain_id`,`tx_id`),
  KEY `tx_chain_id_index` (`chain_id`),
  KEY `tx_id_index` (`tx_id`),
  KEY `block_height_index` (`block_height`),
  KEY `block_hash_index` (`block_hash`),
  KEY `timestamp_index` (`timestamp`),
  KEY `contract_name_index` (`contract_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 正在导出表  chainmaker_explorer_dev.cmb_transaction 的数据：~0 rows (大约)
DELETE FROM `cmb_transaction`;

-- 导出  表 chainmaker_explorer_dev.cmb_transfer 结构
DROP TABLE IF EXISTS `cmb_transfer`;
CREATE TABLE IF NOT EXISTS `cmb_transfer` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `chain_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tx_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `contract_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `block_time` bigint DEFAULT NULL,
  `contract_method` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `from` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `to` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `amount` bigint DEFAULT NULL,
  `token_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tx_status_code` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `contract_result_code` int unsigned DEFAULT NULL,
  `contract_result` mediumblob,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 正在导出表  chainmaker_explorer_dev.cmb_transfer 的数据：~0 rows (大约)
DELETE FROM `cmb_transfer`;

-- 导出  表 chainmaker_explorer_dev.cmb_user 结构
DROP TABLE IF EXISTS `cmb_user`;
CREATE TABLE IF NOT EXISTS `cmb_user` (
  `chain_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `user_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `role` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `org_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `timestamp` bigint DEFAULT NULL,
  `status` int DEFAULT NULL,
  `user_addr` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `id` bigint NOT NULL AUTO_INCREMENT,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `chain_id_user_id_org_id_index` (`chain_id`,`user_id`,`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 正在导出表  chainmaker_explorer_dev.cmb_user 的数据：~0 rows (大约)
DELETE FROM `cmb_user`;

/*!40103 SET TIME_ZONE=IFNULL(@OLD_TIME_ZONE, 'system') */;
/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IFNULL(@OLD_FOREIGN_KEY_CHECKS, 1) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40111 SET SQL_NOTES=IFNULL(@OLD_SQL_NOTES, 1) */;
