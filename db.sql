-- MySQL dump 10.13  Distrib 8.0.41, for Linux (x86_64)
--
-- Host: localhost    Database: tvheadend
-- ------------------------------------------------------
-- Server version	8.0.41-0ubuntu0.24.04.1

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `api_access_logs`
--

DROP TABLE IF EXISTS `api_access_logs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `api_access_logs` (
  `log_id` int NOT NULL AUTO_INCREMENT,
  `user_id` int DEFAULT NULL,
  `endpoint` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `method` varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status_code` int DEFAULT NULL,
  `ip_address` varchar(45) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `user_agent` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `request_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `duration_ms` int DEFAULT NULL,
  `request_params` json DEFAULT NULL,
  PRIMARY KEY (`log_id`),
  KEY `idx_access_time` (`request_time`),
  KEY `idx_user_access` (`user_id`,`request_time`),
  KEY `idx_endpoint` (`endpoint`,`method`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `audit_logs`
--

DROP TABLE IF EXISTS `audit_logs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `audit_logs` (
  `audit_id` int NOT NULL AUTO_INCREMENT,
  `user_id` int DEFAULT NULL,
  `action_type` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `target_type` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `target_id` int DEFAULT NULL,
  `old_value` json DEFAULT NULL,
  `new_value` json DEFAULT NULL,
  `ip_address` varchar(45) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `action_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`audit_id`),
  KEY `idx_audit_time` (`action_time`),
  KEY `idx_audit_action` (`action_type`,`target_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `backup_sources`
--

DROP TABLE IF EXISTS `backup_sources`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `backup_sources` (
  `backup_id` int NOT NULL AUTO_INCREMENT,
  `channel_id` int NOT NULL,
  `backup_type` enum('udp','file','playlist') COLLATE utf8mb4_unicode_ci NOT NULL,
  `source_path` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `priority` int NOT NULL DEFAULT '1',
  `enabled` tinyint(1) DEFAULT '1',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`backup_id`),
  KEY `idx_channel_backup` (`channel_id`,`priority`),
  CONSTRAINT `fk_backup_channel` FOREIGN KEY (`channel_id`) REFERENCES `channels` (`channel_id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `channel_schedules`
--

DROP TABLE IF EXISTS `channel_schedules`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `channel_schedules` (
  `schedule_id` int NOT NULL AUTO_INCREMENT,
  `channel_id` int NOT NULL,
  `program_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `start_time` datetime NOT NULL,
  `end_time` datetime NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci,
  `category` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`schedule_id`),
  KEY `idx_channel_time` (`channel_id`,`start_time`),
  CONSTRAINT `fk_schedule_channel` FOREIGN KEY (`channel_id`) REFERENCES `channels` (`channel_id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `channel_states`
--

DROP TABLE IF EXISTS `channel_states`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `channel_states` (
  `state_id` int NOT NULL AUTO_INCREMENT,
  `channel_id` int NOT NULL,
  `current_playlist_id` int DEFAULT NULL,
  `current_item_id` int DEFAULT NULL,
  `current_position_seconds` float DEFAULT '0',
  `last_update_time` datetime DEFAULT CURRENT_TIMESTAMP,
  `ffmpeg_pid` int DEFAULT NULL COMMENT 'Process ID of FFmpeg instance',
  `running` tinyint(1) DEFAULT '0',
  `error_message` text COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`state_id`),
  UNIQUE KEY `idx_channel_state` (`channel_id`),
  KEY `fk_state_playlist` (`current_playlist_id`),
  KEY `fk_state_item` (`current_item_id`),
  CONSTRAINT `fk_state_channel` FOREIGN KEY (`channel_id`) REFERENCES `channels` (`channel_id`) ON DELETE CASCADE,
  CONSTRAINT `fk_state_item` FOREIGN KEY (`current_item_id`) REFERENCES `playlist_items` (`item_id`) ON DELETE SET NULL,
  CONSTRAINT `fk_state_playlist` FOREIGN KEY (`current_playlist_id`) REFERENCES `playlists` (`playlist_id`) ON DELETE SET NULL
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `channels`
--

DROP TABLE IF EXISTS `channels`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `channels` (
  `channel_id` int NOT NULL AUTO_INCREMENT,
  `channel_name` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `storage_root` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `output_udp` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `playlist_type` enum('infinite_all_media','daily_playlist','infinite_length_playlist') COLLATE utf8mb4_unicode_ci NOT NULL,
  `start_time` time DEFAULT '00:00:00',
  `enabled` tinyint(1) DEFAULT '0',
  `use_previous_day_fallback` tinyint(1) DEFAULT '0',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `video_codec` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT 'hevc_nvenc',
  `video_bitrate` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT '800k',
  `min_bitrate` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT '800k',
  `max_bitrate` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT '800k',
  `buffer_size` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT '1600k',
  `packet_size` int DEFAULT '1316',
  `output_resolution` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT '1920x1080',
  `mpegts_original_network_id` int DEFAULT '1',
  `mpegts_transport_stream_id` int DEFAULT '101',
  `mpegts_service_id` int DEFAULT '1',
  `mpegts_start_pid` int DEFAULT '481',
  `mpegts_pmt_start_pid` int DEFAULT '480',
  `metadata_service_provider` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT 'TV Lanka',
  PRIMARY KEY (`channel_id`),
  UNIQUE KEY `idx_channel_name` (`channel_name`),
  CONSTRAINT `chk_valid_storage_root` CHECK ((`storage_root` like _utf8mb4'/storage/channels/ch-%')),
  CONSTRAINT `chk_valid_udp` CHECK ((`output_udp` like _utf8mb4'udp://%:%'))
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `event_logs`
--

DROP TABLE IF EXISTS `event_logs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `event_logs` (
  `event_id` int NOT NULL AUTO_INCREMENT,
  `channel_id` int DEFAULT NULL,
  `event_type` enum('info','warning','error','system') COLLATE utf8mb4_unicode_ci NOT NULL,
  `event_category` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'E.g., playlist, media, channel, system',
  `message` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `details` json DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`event_id`),
  KEY `idx_event_type_category` (`event_type`,`event_category`),
  KEY `idx_event_channel` (`channel_id`),
  KEY `idx_event_time` (`created_at`),
  CONSTRAINT `fk_event_channel` FOREIGN KEY (`channel_id`) REFERENCES `channels` (`channel_id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `languages`
--

DROP TABLE IF EXISTS `languages`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `languages` (
  `language_id` int NOT NULL AUTO_INCREMENT,
  `language_code` varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL,
  `language_name` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `now_prefix` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'Now: ',
  `next_prefix` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'Next: ',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`language_id`),
  UNIQUE KEY `idx_language_code` (`language_code`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `media_files`
--

DROP TABLE IF EXISTS `media_files`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `media_files` (
  `media_id` int NOT NULL AUTO_INCREMENT,
  `channel_id` int NOT NULL,
  `file_path` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `file_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `duration_seconds` int NOT NULL,
  `program_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `language_id` int DEFAULT NULL,
  `file_size` bigint DEFAULT NULL,
  `mime_type` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `last_modified` timestamp NULL DEFAULT NULL,
  `file_hash` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'SHA-256 hash of file for change detection',
  `scanned_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`media_id`),
  UNIQUE KEY `idx_channel_file_path` (`channel_id`,`file_path`),
  KEY `fk_media_language` (`language_id`),
  KEY `idx_file_path` (`file_path`),
  KEY `idx_channel_media` (`channel_id`),
  KEY `idx_file_name` (`file_name`),
  KEY `idx_scanned_at` (`scanned_at`),
  FULLTEXT KEY `idx_ft_program_name` (`program_name`),
  CONSTRAINT `fk_media_channel` FOREIGN KEY (`channel_id`) REFERENCES `channels` (`channel_id`) ON DELETE CASCADE,
  CONSTRAINT `fk_media_language` FOREIGN KEY (`language_id`) REFERENCES `languages` (`language_id`) ON DELETE SET NULL,
  CONSTRAINT `chk_positive_duration` CHECK ((`duration_seconds` > 0))
) ENGINE=InnoDB AUTO_INCREMENT=1005 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb4 */ ;
/*!50003 SET character_set_results = utf8mb4 */ ;
/*!50003 SET collation_connection  = utf8mb4_0900_ai_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE*/ /*!50017 DEFINER=`dbadmin`@`%`*/ /*!50003 TRIGGER `trg_log_media_changes` AFTER UPDATE ON `media_files` FOR EACH ROW BEGIN
    IF OLD.duration_seconds != NEW.duration_seconds OR OLD.file_size != NEW.file_size THEN
        INSERT INTO event_logs (
            channel_id,
            event_type,
            event_category,
            message,
            details
        ) VALUES (
            NEW.channel_id,
            'info',
            'media',
            'Media file metadata updated',
            JSON_OBJECT(
                'media_id', NEW.media_id,
                'file_path', NEW.file_path,
                'old_duration', OLD.duration_seconds,
                'new_duration', NEW.duration_seconds,
                'old_size', OLD.file_size,
                'new_size', NEW.file_size
            )
        );
    END IF;
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `overlay_configs`
--

DROP TABLE IF EXISTS `overlay_configs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `overlay_configs` (
  `overlay_id` int NOT NULL AUTO_INCREMENT,
  `channel_id` int NOT NULL,
  `overlay_type` enum('image','delogo') COLLATE utf8mb4_unicode_ci NOT NULL,
  `file_path` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `position_x` int NOT NULL DEFAULT '0',
  `position_y` int NOT NULL DEFAULT '0',
  `width` int DEFAULT NULL,
  `height` int DEFAULT NULL,
  `enabled` tinyint(1) DEFAULT '1',
  `overlay_index` int NOT NULL COMMENT 'Index for ordering multiple overlays (1-4)',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`overlay_id`),
  UNIQUE KEY `idx_channel_overlay` (`channel_id`,`overlay_type`,`overlay_index`),
  CONSTRAINT `fk_overlay_channel` FOREIGN KEY (`channel_id`) REFERENCES `channels` (`channel_id`) ON DELETE CASCADE,
  CONSTRAINT `chk_overlay_index` CHECK ((`overlay_index` between 1 and 4))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `overlays`
--

DROP TABLE IF EXISTS `overlays`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `overlays` (
  `id` int NOT NULL AUTO_INCREMENT,
  `channel_id` int NOT NULL,
  `type` enum('image','text') COLLATE utf8mb4_unicode_ci NOT NULL,
  `file_path` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `position_x` int NOT NULL DEFAULT '0',
  `position_y` int NOT NULL DEFAULT '0',
  `enabled` tinyint(1) DEFAULT '1',
  `font_size` int DEFAULT '24',
  `font_color` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT 'white',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_overlay_channel` (`channel_id`),
  CONSTRAINT `overlays_ibfk_1` FOREIGN KEY (`channel_id`) REFERENCES `channels` (`channel_id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `playlist_items`
--

DROP TABLE IF EXISTS `playlist_items`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `playlist_items` (
  `item_id` int NOT NULL AUTO_INCREMENT,
  `playlist_id` int NOT NULL,
  `media_id` int NOT NULL,
  `position` int NOT NULL COMMENT 'Order in playlist',
  `scheduled_start_time` datetime DEFAULT NULL,
  `scheduled_end_time` datetime DEFAULT NULL,
  `actual_start_time` datetime DEFAULT NULL,
  `actual_end_time` datetime DEFAULT NULL,
  `locked` tinyint(1) DEFAULT '0' COMMENT 'Lock item to prevent modifications',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`item_id`),
  KEY `fk_item_media` (`media_id`),
  KEY `idx_playlist_position` (`playlist_id`,`position`),
  KEY `idx_scheduled_times` (`scheduled_start_time`,`scheduled_end_time`),
  KEY `idx_actual_times` (`actual_start_time`,`actual_end_time`),
  CONSTRAINT `fk_item_media` FOREIGN KEY (`media_id`) REFERENCES `media_files` (`media_id`) ON DELETE CASCADE,
  CONSTRAINT `fk_item_playlist` FOREIGN KEY (`playlist_id`) REFERENCES `playlists` (`playlist_id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb4 */ ;
/*!50003 SET character_set_results = utf8mb4 */ ;
/*!50003 SET collation_connection  = utf8mb4_0900_ai_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE*/ /*!50017 DEFINER=`dbadmin`@`%`*/ /*!50003 TRIGGER `trg_maintain_playlist_order` BEFORE INSERT ON `playlist_items` FOR EACH ROW BEGIN
    DECLARE max_position INT;
    
    -- Get the highest current position for this playlist
    SELECT IFNULL(MAX(position), 0) INTO max_position
    FROM playlist_items
    WHERE playlist_id = NEW.playlist_id;
    
    -- If no position specified or position is greater than max+1, set to max+1
    IF NEW.position IS NULL OR NEW.position > max_position + 1 THEN
        SET NEW.position = max_position + 1;
    ELSE
        -- If inserting in the middle, shift existing items
        UPDATE playlist_items
        SET position = position + 1
        WHERE playlist_id = NEW.playlist_id AND position >= NEW.position;
    END IF;
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb4 */ ;
/*!50003 SET character_set_results = utf8mb4 */ ;
/*!50003 SET collation_connection  = utf8mb4_0900_ai_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE*/ /*!50017 DEFINER=`dbadmin`@`%`*/ /*!50003 TRIGGER `trg_update_playlist_on_item_add` AFTER INSERT ON `playlist_items` FOR EACH ROW BEGIN
    -- Call the stored procedure to update playlist timing
    CALL sp_update_playlist_timing(NEW.playlist_id);
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb4 */ ;
/*!50003 SET character_set_results = utf8mb4 */ ;
/*!50003 SET collation_connection  = utf8mb4_0900_ai_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE*/ /*!50017 DEFINER=`dbadmin`@`%`*/ /*!50003 TRIGGER `trg_update_playlist_on_item_update` AFTER UPDATE ON `playlist_items` FOR EACH ROW BEGIN
    IF OLD.media_id != NEW.media_id OR OLD.position != NEW.position THEN
        -- Call the stored procedure to update playlist timing
        CALL sp_update_playlist_timing(NEW.playlist_id);
    END IF;
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb4 */ ;
/*!50003 SET character_set_results = utf8mb4 */ ;
/*!50003 SET collation_connection  = utf8mb4_0900_ai_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE*/ /*!50017 DEFINER=`dbadmin`@`%`*/ /*!50003 TRIGGER `trg_update_playlist_on_item_delete` AFTER DELETE ON `playlist_items` FOR EACH ROW BEGIN
    -- Call the stored procedure to update playlist timing
    CALL sp_update_playlist_timing(OLD.playlist_id);
    
    -- Also reorder the remaining items to maintain sequence
    SET @pos := 0;
    UPDATE playlist_items
    SET position = (@pos := @pos + 1)
    WHERE playlist_id = OLD.playlist_id
    ORDER BY position;
END */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `playlists`
--

DROP TABLE IF EXISTS `playlists`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `playlists` (
  `playlist_id` int NOT NULL AUTO_INCREMENT,
  `channel_id` int NOT NULL,
  `playlist_date` date DEFAULT NULL COMMENT 'For daily playlists, NULL for infinite playlists',
  `status` enum('scheduled','active','completed') COLLATE utf8mb4_unicode_ci DEFAULT 'scheduled',
  `total_duration_seconds` int DEFAULT '0',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`playlist_id`),
  UNIQUE KEY `idx_channel_date` (`channel_id`,`playlist_date`),
  CONSTRAINT `fk_playlist_channel` FOREIGN KEY (`channel_id`) REFERENCES `channels` (`channel_id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `schema_migrations`
--

DROP TABLE IF EXISTS `schema_migrations`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `schema_migrations` (
  `migration_id` int NOT NULL AUTO_INCREMENT,
  `migration_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `batch` int NOT NULL,
  `run_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`migration_id`),
  UNIQUE KEY `idx_migration_name` (`migration_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `system_jobs`
--

DROP TABLE IF EXISTS `system_jobs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `system_jobs` (
  `job_id` int NOT NULL AUTO_INCREMENT,
  `job_type` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `job_data` json DEFAULT NULL,
  `status` enum('pending','running','completed','failed') COLLATE utf8mb4_unicode_ci DEFAULT 'pending',
  `attempts` int DEFAULT '0',
  `max_attempts` int DEFAULT '3',
  `run_at` timestamp NULL DEFAULT NULL,
  `started_at` timestamp NULL DEFAULT NULL,
  `completed_at` timestamp NULL DEFAULT NULL,
  `error_message` text COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`job_id`),
  KEY `idx_job_status` (`status`,`run_at`),
  KEY `idx_job_type` (`job_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `system_settings`
--

DROP TABLE IF EXISTS `system_settings`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `system_settings` (
  `setting_id` int NOT NULL AUTO_INCREMENT,
  `setting_key` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `setting_value` text COLLATE utf8mb4_unicode_ci,
  `description` text COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`setting_id`),
  UNIQUE KEY `idx_setting_key` (`setting_key`)
) ENGINE=InnoDB AUTO_INCREMENT=17 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `text_overlay_configs`
--

DROP TABLE IF EXISTS `text_overlay_configs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `text_overlay_configs` (
  `text_overlay_id` int NOT NULL AUTO_INCREMENT,
  `channel_id` int NOT NULL,
  `position_x` int NOT NULL DEFAULT '10',
  `position_y` int NOT NULL DEFAULT '10',
  `font_size` int DEFAULT '24',
  `font_color` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT 'white',
  `bg_color` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT 'black@0.5',
  `enabled` tinyint(1) DEFAULT '1',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`text_overlay_id`),
  UNIQUE KEY `idx_channel_text_overlay` (`channel_id`),
  CONSTRAINT `fk_text_overlay_channel` FOREIGN KEY (`channel_id`) REFERENCES `channels` (`channel_id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `users`
--

DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `users` (
  `user_id` int NOT NULL AUTO_INCREMENT,
  `username` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `password_hash` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `email` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `role` enum('admin','operator','viewer') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'viewer',
  `api_key` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `is_active` tinyint(1) DEFAULT '1',
  `last_login` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`user_id`),
  UNIQUE KEY `idx_username` (`username`),
  UNIQUE KEY `idx_api_key` (`api_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Temporary view structure for view `view_active_channels`
--

DROP TABLE IF EXISTS `view_active_channels`;
/*!50001 DROP VIEW IF EXISTS `view_active_channels`*/;
SET @saved_cs_client     = @@character_set_client;
/*!50503 SET character_set_client = utf8mb4 */;
/*!50001 CREATE VIEW `view_active_channels` AS SELECT 
 1 AS `channel_id`,
 1 AS `channel_name`,
 1 AS `playlist_type`,
 1 AS `enabled`,
 1 AS `running`,
 1 AS `error_message`,
 1 AS `playlist_id`,
 1 AS `playlist_date`,
 1 AS `playlist_status`,
 1 AS `current_position_seconds`,
 1 AS `last_update_time`,
 1 AS `current_program_name`*/;
SET character_set_client = @saved_cs_client;

--
-- Temporary view structure for view `view_channel_resources`
--

DROP TABLE IF EXISTS `view_channel_resources`;
/*!50001 DROP VIEW IF EXISTS `view_channel_resources`*/;
SET @saved_cs_client     = @@character_set_client;
/*!50503 SET character_set_client = utf8mb4 */;
/*!50001 CREATE VIEW `view_channel_resources` AS SELECT 
 1 AS `channel_id`,
 1 AS `channel_name`,
 1 AS `running`,
 1 AS `media_files_count`,
 1 AS `total_media_size`,
 1 AS `scheduled_items`,
 1 AS `total_duration`*/;
SET character_set_client = @saved_cs_client;

--
-- Temporary view structure for view `view_media_needing_scan`
--

DROP TABLE IF EXISTS `view_media_needing_scan`;
/*!50001 DROP VIEW IF EXISTS `view_media_needing_scan`*/;
SET @saved_cs_client     = @@character_set_client;
/*!50503 SET character_set_client = utf8mb4 */;
/*!50001 CREATE VIEW `view_media_needing_scan` AS SELECT 
 1 AS `media_id`,
 1 AS `channel_id`,
 1 AS `channel_name`,
 1 AS `file_path`,
 1 AS `file_name`,
 1 AS `scanned_at`,
 1 AS `last_modified`,
 1 AS `duration_seconds`*/;
SET character_set_client = @saved_cs_client;

--
-- Temporary view structure for view `view_recent_errors`
--

DROP TABLE IF EXISTS `view_recent_errors`;
/*!50001 DROP VIEW IF EXISTS `view_recent_errors`*/;
SET @saved_cs_client     = @@character_set_client;
/*!50503 SET character_set_client = utf8mb4 */;
/*!50001 CREATE VIEW `view_recent_errors` AS SELECT 
 1 AS `event_id`,
 1 AS `channel_id`,
 1 AS `channel_name`,
 1 AS `event_type`,
 1 AS `event_category`,
 1 AS `message`,
 1 AS `created_at`*/;
SET character_set_client = @saved_cs_client;

--
-- Temporary view structure for view `view_today_playlists`
--

DROP TABLE IF EXISTS `view_today_playlists`;
/*!50001 DROP VIEW IF EXISTS `view_today_playlists`*/;
SET @saved_cs_client     = @@character_set_client;
/*!50503 SET character_set_client = utf8mb4 */;
/*!50001 CREATE VIEW `view_today_playlists` AS SELECT 
 1 AS `playlist_id`,
 1 AS `channel_id`,
 1 AS `channel_name`,
 1 AS `playlist_date`,
 1 AS `status`,
 1 AS `total_duration_seconds`,
 1 AS `total_items`,
 1 AS `start_time`,
 1 AS `end_time`*/;
SET character_set_client = @saved_cs_client;

--
-- Temporary view structure for view `view_upcoming_programs`
--

DROP TABLE IF EXISTS `view_upcoming_programs`;
/*!50001 DROP VIEW IF EXISTS `view_upcoming_programs`*/;
SET @saved_cs_client     = @@character_set_client;
/*!50503 SET character_set_client = utf8mb4 */;
/*!50001 CREATE VIEW `view_upcoming_programs` AS SELECT 
 1 AS `channel_id`,
 1 AS `channel_name`,
 1 AS `item_id`,
 1 AS `program_name`,
 1 AS `scheduled_start_time`,
 1 AS `scheduled_end_time`,
 1 AS `minutes_until_start`*/;
SET character_set_client = @saved_cs_client;

--
-- Final view structure for view `view_active_channels`
--

/*!50001 DROP VIEW IF EXISTS `view_active_channels`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb4 */;
/*!50001 SET character_set_results     = utf8mb4 */;
/*!50001 SET collation_connection      = utf8mb4_0900_ai_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50013 DEFINER=`dbadmin`@`%` SQL SECURITY DEFINER */
/*!50001 VIEW `view_active_channels` AS select `c`.`channel_id` AS `channel_id`,`c`.`channel_name` AS `channel_name`,`c`.`playlist_type` AS `playlist_type`,`c`.`enabled` AS `enabled`,`cs`.`running` AS `running`,`cs`.`error_message` AS `error_message`,`p`.`playlist_id` AS `playlist_id`,`p`.`playlist_date` AS `playlist_date`,`p`.`status` AS `playlist_status`,`cs`.`current_position_seconds` AS `current_position_seconds`,`cs`.`last_update_time` AS `last_update_time`,`mf`.`program_name` AS `current_program_name` from ((((`channels` `c` left join `channel_states` `cs` on((`c`.`channel_id` = `cs`.`channel_id`))) left join `playlists` `p` on((`cs`.`current_playlist_id` = `p`.`playlist_id`))) left join `playlist_items` `pi` on((`cs`.`current_item_id` = `pi`.`item_id`))) left join `media_files` `mf` on((`pi`.`media_id` = `mf`.`media_id`))) where (`c`.`enabled` = true) */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;

--
-- Final view structure for view `view_channel_resources`
--

/*!50001 DROP VIEW IF EXISTS `view_channel_resources`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb4 */;
/*!50001 SET character_set_results     = utf8mb4 */;
/*!50001 SET collation_connection      = utf8mb4_0900_ai_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50013 DEFINER=`dbadmin`@`%` SQL SECURITY DEFINER */
/*!50001 VIEW `view_channel_resources` AS select `c`.`channel_id` AS `channel_id`,`c`.`channel_name` AS `channel_name`,`cs`.`running` AS `running`,count(`mf`.`media_id`) AS `media_files_count`,sum(`mf`.`file_size`) AS `total_media_size`,count(`pi`.`item_id`) AS `scheduled_items`,sum(`mf`.`duration_seconds`) AS `total_duration` from ((((`channels` `c` left join `channel_states` `cs` on((`c`.`channel_id` = `cs`.`channel_id`))) left join `media_files` `mf` on((`c`.`channel_id` = `mf`.`channel_id`))) left join `playlists` `p` on(((`c`.`channel_id` = `p`.`channel_id`) and (`p`.`status` = 'active')))) left join `playlist_items` `pi` on((`p`.`playlist_id` = `pi`.`playlist_id`))) group by `c`.`channel_id`,`c`.`channel_name`,`cs`.`running` */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;

--
-- Final view structure for view `view_media_needing_scan`
--

/*!50001 DROP VIEW IF EXISTS `view_media_needing_scan`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb4 */;
/*!50001 SET character_set_results     = utf8mb4 */;
/*!50001 SET collation_connection      = utf8mb4_0900_ai_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50013 DEFINER=`dbadmin`@`%` SQL SECURITY DEFINER */
/*!50001 VIEW `view_media_needing_scan` AS select `mf`.`media_id` AS `media_id`,`mf`.`channel_id` AS `channel_id`,`c`.`channel_name` AS `channel_name`,`mf`.`file_path` AS `file_path`,`mf`.`file_name` AS `file_name`,`mf`.`scanned_at` AS `scanned_at`,`mf`.`last_modified` AS `last_modified`,`mf`.`duration_seconds` AS `duration_seconds` from (`media_files` `mf` join `channels` `c` on((`mf`.`channel_id` = `c`.`channel_id`))) where ((`mf`.`scanned_at` is null) or (`mf`.`scanned_at` < `mf`.`last_modified`) or (`mf`.`duration_seconds` = 0)) */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;

--
-- Final view structure for view `view_recent_errors`
--

/*!50001 DROP VIEW IF EXISTS `view_recent_errors`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb4 */;
/*!50001 SET character_set_results     = utf8mb4 */;
/*!50001 SET collation_connection      = utf8mb4_0900_ai_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50013 DEFINER=`dbadmin`@`%` SQL SECURITY DEFINER */
/*!50001 VIEW `view_recent_errors` AS select `e`.`event_id` AS `event_id`,`e`.`channel_id` AS `channel_id`,`c`.`channel_name` AS `channel_name`,`e`.`event_type` AS `event_type`,`e`.`event_category` AS `event_category`,`e`.`message` AS `message`,`e`.`created_at` AS `created_at` from (`event_logs` `e` left join `channels` `c` on((`e`.`channel_id` = `c`.`channel_id`))) where ((`e`.`event_type` = 'error') and (`e`.`created_at` > (now() - interval 24 hour))) order by `e`.`created_at` desc */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;

--
-- Final view structure for view `view_today_playlists`
--

/*!50001 DROP VIEW IF EXISTS `view_today_playlists`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb4 */;
/*!50001 SET character_set_results     = utf8mb4 */;
/*!50001 SET collation_connection      = utf8mb4_0900_ai_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50013 DEFINER=`dbadmin`@`%` SQL SECURITY DEFINER */
/*!50001 VIEW `view_today_playlists` AS select `p`.`playlist_id` AS `playlist_id`,`c`.`channel_id` AS `channel_id`,`c`.`channel_name` AS `channel_name`,`p`.`playlist_date` AS `playlist_date`,`p`.`status` AS `status`,`p`.`total_duration_seconds` AS `total_duration_seconds`,count(`pi`.`item_id`) AS `total_items`,min(`pi`.`scheduled_start_time`) AS `start_time`,max(`pi`.`scheduled_end_time`) AS `end_time` from ((`playlists` `p` join `channels` `c` on((`p`.`channel_id` = `c`.`channel_id`))) left join `playlist_items` `pi` on((`p`.`playlist_id` = `pi`.`playlist_id`))) where (`p`.`playlist_date` = curdate()) group by `p`.`playlist_id`,`c`.`channel_id`,`c`.`channel_name`,`p`.`playlist_date`,`p`.`status`,`p`.`total_duration_seconds` */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;

--
-- Final view structure for view `view_upcoming_programs`
--

/*!50001 DROP VIEW IF EXISTS `view_upcoming_programs`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb4 */;
/*!50001 SET character_set_results     = utf8mb4 */;
/*!50001 SET collation_connection      = utf8mb4_0900_ai_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50013 DEFINER=`dbadmin`@`%` SQL SECURITY DEFINER */
/*!50001 VIEW `view_upcoming_programs` AS select `c`.`channel_id` AS `channel_id`,`c`.`channel_name` AS `channel_name`,`pi`.`item_id` AS `item_id`,`mf`.`program_name` AS `program_name`,`pi`.`scheduled_start_time` AS `scheduled_start_time`,`pi`.`scheduled_end_time` AS `scheduled_end_time`,timestampdiff(MINUTE,now(),`pi`.`scheduled_start_time`) AS `minutes_until_start` from (((`playlist_items` `pi` join `playlists` `p` on((`pi`.`playlist_id` = `p`.`playlist_id`))) join `channels` `c` on((`p`.`channel_id` = `c`.`channel_id`))) join `media_files` `mf` on((`pi`.`media_id` = `mf`.`media_id`))) where ((`pi`.`scheduled_start_time` > now()) and (`pi`.`scheduled_start_time` < (now() + interval 4 hour))) order by `c`.`channel_id`,`pi`.`scheduled_start_time` */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2025-04-20 12:27:51
