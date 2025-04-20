-- TV Headend Playlist Server Database Schema
-- Complete MySQL script with all improvements and media scanning support

-- Create database if it doesn't exist
CREATE DATABASE IF NOT EXISTS tvheadend CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE tvheadend;

-- Table for storing channels configuration
CREATE TABLE channels (
    channel_id INT AUTO_INCREMENT PRIMARY KEY,
    channel_name VARCHAR(100) NOT NULL,
    storage_root VARCHAR(255) NOT NULL,
    output_udp VARCHAR(255) NOT NULL,
    playlist_type ENUM('infinite_all_media', 'daily_playlist', 'infinite_length_playlist') NOT NULL,
    start_time TIME DEFAULT '00:00:00',
    enabled BOOLEAN DEFAULT FALSE,
    use_previous_day_fallback BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Encoding parameters with default values
    video_codec VARCHAR(50) DEFAULT 'hevc_nvenc',
    video_bitrate VARCHAR(20) DEFAULT '800k',
    min_bitrate VARCHAR(20) DEFAULT '800k',
    max_bitrate VARCHAR(20) DEFAULT '800k',
    buffer_size VARCHAR(20) DEFAULT '1600k',
    packet_size INT DEFAULT 1316,
    output_resolution VARCHAR(20) DEFAULT '1920x1080',
    mpegts_original_network_id INT DEFAULT 1,
    mpegts_transport_stream_id INT DEFAULT 101,
    mpegts_service_id INT DEFAULT 1,
    mpegts_start_pid INT DEFAULT 481,
    mpegts_pmt_start_pid INT DEFAULT 480,
    metadata_service_provider VARCHAR(100) DEFAULT 'TV Lanka',
    
    -- Add unique constraint for channel name
    UNIQUE INDEX idx_channel_name (channel_name),
    
    -- Add check constraints
    CONSTRAINT chk_valid_udp CHECK (output_udp LIKE 'udp://%:%'),
    CONSTRAINT chk_valid_storage_root CHECK (storage_root LIKE '/storage/channels/ch-%')
);

-- Table for storing overlay configurations
CREATE TABLE overlay_configs (
    overlay_id INT AUTO_INCREMENT PRIMARY KEY,
    channel_id INT NOT NULL,
    overlay_type ENUM('image', 'delogo') NOT NULL,
    file_path VARCHAR(255),
    position_x INT NOT NULL DEFAULT 0,
    position_y INT NOT NULL DEFAULT 0,
    width INT,
    height INT,
    enabled BOOLEAN DEFAULT TRUE,
    overlay_index INT NOT NULL COMMENT 'Index for ordering multiple overlays (1-4)',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Add foreign key constraint to link to channels table
    CONSTRAINT fk_overlay_channel FOREIGN KEY (channel_id) REFERENCES channels (channel_id) ON DELETE CASCADE,
    
    -- Add constraint to ensure only 4 overlays per channel
    CONSTRAINT chk_overlay_index CHECK (overlay_index BETWEEN 1 AND 4),
    
    -- Add unique constraint for channel_id and overlay_index
    UNIQUE INDEX idx_channel_overlay (channel_id, overlay_type, overlay_index)
);

-- Table for storing text overlay configuration
CREATE TABLE text_overlay_configs (
    text_overlay_id INT AUTO_INCREMENT PRIMARY KEY,
    channel_id INT NOT NULL,
    position_x INT NOT NULL DEFAULT 10,
    position_y INT NOT NULL DEFAULT 10,
    font_size INT DEFAULT 24,
    font_color VARCHAR(20) DEFAULT 'white',
    bg_color VARCHAR(20) DEFAULT 'black@0.5',
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Add foreign key constraint to link to channels table
    CONSTRAINT fk_text_overlay_channel FOREIGN KEY (channel_id) REFERENCES channels (channel_id) ON DELETE CASCADE,
    
    -- Add unique constraint for channel_id (one text overlay config per channel)
    UNIQUE INDEX idx_channel_text_overlay (channel_id)
);

-- Table for storing language-specific text labels
CREATE TABLE languages (
    language_id INT AUTO_INCREMENT PRIMARY KEY,
    language_code VARCHAR(10) NOT NULL,
    language_name VARCHAR(50) NOT NULL,
    now_prefix VARCHAR(20) NOT NULL DEFAULT 'Now: ',
    next_prefix VARCHAR(20) NOT NULL DEFAULT 'Next: ',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Add unique constraint for language_code
    UNIQUE INDEX idx_language_code (language_code)
);

-- Insert default languages
INSERT INTO languages (language_code, language_name, now_prefix, next_prefix) VALUES 
('en', 'English', 'Now: ', 'Next: '),
('si', 'Sinhala', 'දැන්: ', 'ඊළඟට: '),
('ta', 'Tamil', 'இப்போது: ', 'அடுத்து: ');

-- Table for storing media files information
CREATE TABLE media_files (
    media_id INT AUTO_INCREMENT PRIMARY KEY,
    channel_id INT NOT NULL,
    file_path VARCHAR(255) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    duration_seconds INT NOT NULL,
    program_name VARCHAR(255),
    language_id INT,
    file_size BIGINT,
    mime_type VARCHAR(100),
    last_modified TIMESTAMP NULL DEFAULT NULL,
    file_hash VARCHAR(64) COMMENT 'SHA-256 hash of file for change detection',
    scanned_at TIMESTAMP NULL DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Add foreign key constraints
    CONSTRAINT fk_media_channel FOREIGN KEY (channel_id) REFERENCES channels (channel_id) ON DELETE CASCADE,
    CONSTRAINT fk_media_language FOREIGN KEY (language_id) REFERENCES languages (language_id) ON DELETE SET NULL,
    CONSTRAINT chk_positive_duration CHECK (duration_seconds > 0),
    
    -- Add indexes for faster queries
    INDEX idx_file_path (file_path),
    INDEX idx_channel_media (channel_id),
    INDEX idx_file_name (file_name),
    INDEX idx_scanned_at (scanned_at),
    FULLTEXT INDEX idx_ft_program_name (program_name),
    UNIQUE INDEX idx_channel_file_path (channel_id, file_path)
);

-- Table for storing playlists
CREATE TABLE playlists (
    playlist_id INT AUTO_INCREMENT PRIMARY KEY,
    channel_id INT NOT NULL,
    playlist_date DATE COMMENT 'For daily playlists, NULL for infinite playlists',
    status ENUM('scheduled', 'active', 'completed') DEFAULT 'scheduled',
    total_duration_seconds INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Add foreign key constraint
    CONSTRAINT fk_playlist_channel FOREIGN KEY (channel_id) REFERENCES channels (channel_id) ON DELETE CASCADE,
    
    -- Add unique constraint for channel_id and playlist_date
    UNIQUE INDEX idx_channel_date (channel_id, playlist_date)
);

-- Table for storing playlist items
CREATE TABLE playlist_items (
    item_id INT AUTO_INCREMENT PRIMARY KEY,
    playlist_id INT NOT NULL,
    media_id INT NOT NULL,
    position INT NOT NULL COMMENT 'Order in playlist',
    scheduled_start_time DATETIME,
    scheduled_end_time DATETIME,
    actual_start_time DATETIME,
    actual_end_time DATETIME,
    locked BOOLEAN DEFAULT FALSE COMMENT 'Lock item to prevent modifications',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Add foreign key constraints
    CONSTRAINT fk_item_playlist FOREIGN KEY (playlist_id) REFERENCES playlists (playlist_id) ON DELETE CASCADE,
    CONSTRAINT fk_item_media FOREIGN KEY (media_id) REFERENCES media_files (media_id) ON DELETE CASCADE,
    
    -- Add indexes for faster lookups
    INDEX idx_playlist_position (playlist_id, position),
    INDEX idx_scheduled_times (scheduled_start_time, scheduled_end_time),
    INDEX idx_actual_times (actual_start_time, actual_end_time)
);

-- Table for storing channel status and current state
CREATE TABLE channel_states (
    state_id INT AUTO_INCREMENT PRIMARY KEY,
    channel_id INT NOT NULL,
    current_playlist_id INT,
    current_item_id INT,
    current_position_seconds FLOAT DEFAULT 0,
    last_update_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    ffmpeg_pid INT COMMENT 'Process ID of FFmpeg instance',
    running BOOLEAN DEFAULT FALSE,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Add foreign key constraints
    CONSTRAINT fk_state_channel FOREIGN KEY (channel_id) REFERENCES channels (channel_id) ON DELETE CASCADE,
    CONSTRAINT fk_state_playlist FOREIGN KEY (current_playlist_id) REFERENCES playlists (playlist_id) ON DELETE SET NULL,
    CONSTRAINT fk_state_item FOREIGN KEY (current_item_id) REFERENCES playlist_items (item_id) ON DELETE SET NULL,
    
    -- Add unique constraint for channel_id (one state per channel)
    UNIQUE INDEX idx_channel_state (channel_id)
);

-- Table for logging events
CREATE TABLE event_logs (
    event_id INT AUTO_INCREMENT PRIMARY KEY,
    channel_id INT,
    event_type ENUM('info', 'warning', 'error', 'system') NOT NULL,
    event_category VARCHAR(50) NOT NULL COMMENT 'E.g., playlist, media, channel, system',
    message TEXT NOT NULL,
    details JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Add foreign key constraint
    CONSTRAINT fk_event_channel FOREIGN KEY (channel_id) REFERENCES channels (channel_id) ON DELETE SET NULL,
    
    -- Add index for efficient filtering
    INDEX idx_event_type_category (event_type, event_category),
    INDEX idx_event_channel (channel_id),
    INDEX idx_event_time (created_at)
) PARTITION BY RANGE (TO_DAYS(created_at)) (
    PARTITION p_prev VALUES LESS THAN (TO_DAYS('2023-01-01')),
    PARTITION p_2023_01 VALUES LESS THAN (TO_DAYS('2023-02-01')),
    PARTITION p_2023_02 VALUES LESS THAN (TO_DAYS('2023-03-01')),
    PARTITION p_future VALUES LESS THAN MAXVALUE
);

-- Table for storing API users
CREATE TABLE users (
    user_id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(100),
    role ENUM('admin', 'operator', 'viewer') NOT NULL DEFAULT 'viewer',
    api_key VARCHAR(64),
    is_active BOOLEAN DEFAULT TRUE,
    last_login DATETIME,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Add unique constraints
    UNIQUE INDEX idx_username (username),
    UNIQUE INDEX idx_api_key (api_key)
);

-- Table for storing system settings
CREATE TABLE system_settings (
    setting_id INT AUTO_INCREMENT PRIMARY KEY,
    setting_key VARCHAR(50) NOT NULL,
    setting_value TEXT,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Add unique constraint
    UNIQUE INDEX idx_setting_key (setting_key)
);

-- Insert default system settings
INSERT INTO system_settings (setting_key, setting_value, description) VALUES
('channel_check_interval', '60', 'Interval in seconds to check channel status'),
('playlist_check_interval', '300', 'Interval in seconds to check for new playlists'),
('log_retention_days', '30', 'Number of days to retain event logs'),
('gpu_memory_limit', '4096', 'Maximum GPU memory to use in MB'),
('default_language', 'en', 'Default language code'),
('max_ffmpeg_instances', '8', 'Maximum number of concurrent FFmpeg processes'),
('gpu_memory_per_channel', '512', 'GPU memory allocation per channel in MB'),
('udp_timeout_seconds', '30', 'UDP stream timeout in seconds'),
('health_check_interval', '10', 'Interval in seconds for health checks'),
('max_log_entries_per_channel', '1000', 'Maximum log entries to keep per channel'),
('playlist_preload_seconds', '30', 'Seconds before scheduled time to preload playlist'),
('overlay_cache_size', '256', 'Overlay cache size in MB'),
('media_scan_interval', '3600', 'Interval in seconds for media directory scans'),
('media_file_extensions', 'mp4,mov,mkv,ts,mpeg,mpg,m2ts', 'Supported media file extensions'),
('media_min_duration', '10', 'Minimum duration in seconds for media files'),
('media_max_duration', '86400', 'Maximum duration in seconds for media files');

-- Table for storing channel schedules (for EPG)
CREATE TABLE channel_schedules (
    schedule_id INT AUTO_INCREMENT PRIMARY KEY,
    channel_id INT NOT NULL,
    program_name VARCHAR(255) NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    description TEXT,
    category VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_schedule_channel FOREIGN KEY (channel_id) REFERENCES channels (channel_id) ON DELETE CASCADE,
    
    INDEX idx_channel_time (channel_id, start_time)
);

-- Table for storing backup sources
CREATE TABLE backup_sources (
    backup_id INT AUTO_INCREMENT PRIMARY KEY,
    channel_id INT NOT NULL,
    backup_type ENUM('udp', 'file', 'playlist') NOT NULL,
    source_path VARCHAR(255) NOT NULL,
    priority INT NOT NULL DEFAULT 1,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_backup_channel FOREIGN KEY (channel_id) REFERENCES channels (channel_id) ON DELETE CASCADE,
    
    INDEX idx_channel_backup (channel_id, priority)
);

-- Table for storing system jobs
CREATE TABLE system_jobs (
    job_id INT AUTO_INCREMENT PRIMARY KEY,
    job_type VARCHAR(50) NOT NULL,
    job_data JSON,
    status ENUM('pending', 'running', 'completed', 'failed') DEFAULT 'pending',
    attempts INT DEFAULT 0,
    max_attempts INT DEFAULT 3,
    run_at TIMESTAMP NULL DEFAULT NULL,
    started_at TIMESTAMP NULL DEFAULT NULL,
    completed_at TIMESTAMP NULL DEFAULT NULL,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_job_status (status, run_at),
    INDEX idx_job_type (job_type)
);

-- Table for API access logs
CREATE TABLE api_access_logs (
    log_id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT,
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    status_code INT,
    ip_address VARCHAR(45),
    user_agent VARCHAR(255),
    request_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    duration_ms INT,
    request_params JSON,
    
    INDEX idx_access_time (request_time),
    INDEX idx_user_access (user_id, request_time),
    INDEX idx_endpoint (endpoint, method)
);

-- Table for sensitive operations audit
CREATE TABLE audit_logs (
    audit_id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT,
    action_type VARCHAR(50) NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    target_id INT,
    old_value JSON,
    new_value JSON,
    ip_address VARCHAR(45),
    action_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_audit_time (action_time),
    INDEX idx_audit_action (action_type, target_type)
);

-- Table for tracking schema migrations
CREATE TABLE schema_migrations (
    migration_id INT AUTO_INCREMENT PRIMARY KEY,
    migration_name VARCHAR(255) NOT NULL,
    batch INT NOT NULL,
    run_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE INDEX idx_migration_name (migration_name)
);

-- Create views for easier reporting

-- View for active channels with their current status
CREATE VIEW view_active_channels AS
SELECT 
    c.channel_id,
    c.channel_name,
    c.playlist_type,
    c.enabled,
    cs.running,
    cs.error_message,
    p.playlist_id,
    p.playlist_date,
    p.status AS playlist_status,
    cs.current_position_seconds,
    cs.last_update_time,
    mf.program_name AS current_program_name
FROM 
    channels c
LEFT JOIN 
    channel_states cs ON c.channel_id = cs.channel_id
LEFT JOIN 
    playlists p ON cs.current_playlist_id = p.playlist_id
LEFT JOIN 
    playlist_items pi ON cs.current_item_id = pi.item_id
LEFT JOIN 
    media_files mf ON pi.media_id = mf.media_id
WHERE 
    c.enabled = TRUE;

-- View for today's playlists
CREATE VIEW view_today_playlists AS
SELECT 
    p.playlist_id,
    c.channel_id,
    c.channel_name,
    p.playlist_date,
    p.status,
    p.total_duration_seconds,
    COUNT(pi.item_id) AS total_items,
    MIN(pi.scheduled_start_time) AS start_time,
    MAX(pi.scheduled_end_time) AS end_time
FROM 
    playlists p
JOIN 
    channels c ON p.channel_id = c.channel_id
LEFT JOIN 
    playlist_items pi ON p.playlist_id = pi.playlist_id
WHERE 
    p.playlist_date = CURDATE()
GROUP BY 
    p.playlist_id, c.channel_id, c.channel_name, p.playlist_date, p.status, p.total_duration_seconds;

-- View for recent errors
CREATE VIEW view_recent_errors AS
SELECT 
    e.event_id,
    e.channel_id,
    c.channel_name,
    e.event_type,
    e.event_category,
    e.message,
    e.created_at
FROM 
    event_logs e
LEFT JOIN 
    channels c ON e.channel_id = c.channel_id
WHERE 
    e.event_type = 'error'
    AND e.created_at > DATE_SUB(NOW(), INTERVAL 24 HOUR)
ORDER BY 
    e.created_at DESC;

-- View for channel resource usage
CREATE VIEW view_channel_resources AS
SELECT 
    c.channel_id,
    c.channel_name,
    cs.running,
    COUNT(mf.media_id) AS media_files_count,
    SUM(mf.file_size) AS total_media_size,
    COUNT(pi.item_id) AS scheduled_items,
    SUM(mf.duration_seconds) AS total_duration
FROM 
    channels c
LEFT JOIN 
    channel_states cs ON c.channel_id = cs.channel_id
LEFT JOIN 
    media_files mf ON c.channel_id = mf.channel_id
LEFT JOIN 
    playlists p ON c.channel_id = p.channel_id AND p.status = 'active'
LEFT JOIN 
    playlist_items pi ON p.playlist_id = pi.playlist_id
GROUP BY 
    c.channel_id, c.channel_name, cs.running;

-- View for upcoming programs
CREATE VIEW view_upcoming_programs AS
SELECT 
    c.channel_id,
    c.channel_name,
    pi.item_id,
    mf.program_name,
    pi.scheduled_start_time,
    pi.scheduled_end_time,
    TIMESTAMPDIFF(MINUTE, NOW(), pi.scheduled_start_time) AS minutes_until_start
FROM 
    playlist_items pi
JOIN 
    playlists p ON pi.playlist_id = p.playlist_id
JOIN 
    channels c ON p.channel_id = c.channel_id
JOIN 
    media_files mf ON pi.media_id = mf.media_id
WHERE 
    pi.scheduled_start_time > NOW()
    AND pi.scheduled_start_time < DATE_ADD(NOW(), INTERVAL 4 HOUR)
ORDER BY 
    c.channel_id, pi.scheduled_start_time;

-- View for media files needing scanning
CREATE VIEW view_media_needing_scan AS
SELECT 
    mf.media_id,
    mf.channel_id,
    c.channel_name,
    mf.file_path,
    mf.file_name,
    mf.scanned_at,
    mf.last_modified,
    mf.duration_seconds
FROM 
    media_files mf
JOIN 
    channels c ON mf.channel_id = c.channel_id
WHERE 
    mf.scanned_at IS NULL 
    OR mf.scanned_at < mf.last_modified
    OR mf.duration_seconds = 0;

-- Stored procedures

DELIMITER //

-- Procedure to add a new channel
CREATE PROCEDURE sp_add_channel(
    IN p_channel_name VARCHAR(100),
    IN p_storage_root VARCHAR(255),
    IN p_output_udp VARCHAR(255),
    IN p_playlist_type VARCHAR(50),
    IN p_start_time TIME,
    IN p_enabled BOOLEAN
)
BEGIN
    INSERT INTO channels (
        channel_name,
        storage_root,
        output_udp,
        playlist_type,
        start_time,
        enabled
    ) VALUES (
        p_channel_name,
        p_storage_root,
        p_output_udp,
        p_playlist_type,
        p_start_time,
        p_enabled
    );
    
    -- Initialize channel state
    INSERT INTO channel_states (channel_id, running)
    VALUES (LAST_INSERT_ID(), FALSE);
    
    -- Log the event
    INSERT INTO event_logs (
        channel_id,
        event_type,
        event_category,
        message
    ) VALUES (
        LAST_INSERT_ID(),
        'info',
        'channel',
        CONCAT('Channel created: ', p_channel_name)
    );
END //

-- Procedure to update channel state
CREATE PROCEDURE sp_update_channel_state(
    IN p_channel_id INT,
    IN p_running BOOLEAN,
    IN p_current_playlist_id INT,
    IN p_current_item_id INT,
    IN p_current_position FLOAT,
    IN p_error_message TEXT
)
BEGIN
    DECLARE v_prev_state BOOLEAN;
    
    -- Get previous running state
    SELECT running INTO v_prev_state FROM channel_states WHERE channel_id = p_channel_id;
    
    -- Update channel state
    INSERT INTO channel_states (
        channel_id,
        running,
        current_playlist_id,
        current_item_id,
        current_position_seconds,
        error_message,
        last_update_time
    ) VALUES (
        p_channel_id,
        p_running,
        p_current_playlist_id,
        p_current_item_id,
        p_current_position,
        p_error_message,
        NOW()
    ) ON DUPLICATE KEY UPDATE
        running = VALUES(running),
        current_playlist_id = VALUES(current_playlist_id),
        current_item_id = VALUES(current_item_id),
        current_position_seconds = VALUES(current_position_seconds),
        error_message = VALUES(error_message),
        last_update_time = VALUES(last_update_time);
    
    -- Log state change if different
    IF v_prev_state IS NOT NULL AND v_prev_state != p_running THEN
        INSERT INTO event_logs (
            channel_id,
            event_type,
            event_category,
            message
        ) VALUES (
            p_channel_id,
            'info',
            'channel',
            CONCAT('Channel state changed to: ', IF(p_running, 'running', 'stopped'))
        );
    END IF;
    
    -- Log error if present
    IF p_error_message IS NOT NULL THEN
        INSERT INTO event_logs (
            channel_id,
            event_type,
            event_category,
            message,
            details
        ) VALUES (
            p_channel_id,
            'error',
            'channel',
            'Channel error occurred',
            JSON_OBJECT('error', p_error_message)
        );
    END IF;
END //

-- Procedure to handle playlist fallback
CREATE PROCEDURE sp_handle_playlist_fallback(IN p_channel_id INT)
BEGIN
    DECLARE v_playlist_id INT;
    DECLARE v_playlist_date DATE;
    DECLARE v_use_fallback BOOLEAN;
    
    -- Check if channel uses fallback
    SELECT use_previous_day_fallback INTO v_use_fallback 
    FROM channels WHERE channel_id = p_channel_id;
    
    IF v_use_fallback THEN
        -- Find most recent playlist before today
        SELECT playlist_id, playlist_date INTO v_playlist_id, v_playlist_date
        FROM playlists
        WHERE channel_id = p_channel_id
        AND playlist_date < CURDATE()
        ORDER BY playlist_date DESC
        LIMIT 1;
        
        IF v_playlist_id IS NOT NULL THEN
            -- Create a copy for today
            INSERT INTO playlists (
                channel_id,
                playlist_date,
                status,
                total_duration_seconds
            )
            SELECT 
                channel_id,
                CURDATE(),
                'active',
                total_duration_seconds
            FROM playlists
            WHERE playlist_id = v_playlist_id;
            
            SET @new_playlist_id = LAST_INSERT_ID();
            
            -- Copy playlist items
            INSERT INTO playlist_items (
                playlist_id,
                media_id,
                position,
                scheduled_start_time,
                scheduled_end_time,
                locked
            )
            SELECT 
                @new_playlist_id,
                media_id,
                position,
                NULL, -- Will be set by trigger
                NULL, -- Will be set by trigger
                locked
            FROM playlist_items
            WHERE playlist_id = v_playlist_id;
            
            -- Log the fallback activation
            INSERT INTO event_logs (
                channel_id,
                event_type,
                event_category,
                message,
                details
            ) VALUES (
                p_channel_id,
                'warning',
                'playlist',
                'Using fallback playlist from previous day',
                JSON_OBJECT(
                    'original_date', v_playlist_date,
                    'new_date', CURDATE(),
                    'original_playlist_id', v_playlist_id,
                    'new_playlist_id', @new_playlist_id
                )
            );
            
            -- Return the new playlist ID
            SELECT @new_playlist_id AS fallback_playlist_id;
        ELSE
            -- No fallback playlist available
            INSERT INTO event_logs (
                channel_id,
                event_type,
                event_category,
                message
            ) VALUES (
                p_channel_id,
                'error',
                'playlist',
                'No fallback playlist available'
            );
            
            SIGNAL SQLSTATE '45000' 
            SET MESSAGE_TEXT = 'No fallback playlist available for this channel';
        END IF;
    ELSE
        SIGNAL SQLSTATE '45000' 
        SET MESSAGE_TEXT = 'Channel is not configured to use playlist fallback';
    END IF;
END //

-- Procedure to update playlist timing
CREATE PROCEDURE sp_update_playlist_timing(IN p_playlist_id INT)
BEGIN
    DECLARE v_start_time DATETIME;
    DECLARE v_channel_id INT;
    DECLARE v_playlist_date DATE;
    DECLARE v_playlist_type VARCHAR(50);
    DECLARE v_start_hour TIME;
    
    -- Get playlist information
    SELECT 
        p.channel_id, 
        p.playlist_date,
        c.playlist_type,
        c.start_time
    INTO 
        v_channel_id,
        v_playlist_date,
        v_playlist_type,
        v_start_hour
    FROM 
        playlists p
    JOIN 
        channels c ON p.channel_id = c.channel_id
    WHERE 
        p.playlist_id = p_playlist_id;
    
    -- Determine start time based on playlist type
    IF v_playlist_type = 'daily_playlist' AND v_playlist_date IS NOT NULL THEN
        -- For daily playlists, use the configured start time
        SET v_start_time = TIMESTAMP(v_playlist_date, v_start_hour);
    ELSE
        -- For other playlist types, use the current time as start
        SET v_start_time = NOW();
    END IF;
    
    -- Update playlist items with calculated timing
    UPDATE playlist_items pi
    JOIN (
        SELECT 
            item_id,
            @running_time := IF(@prev_playlist_id = playlist_id, 
                               @running_time + IFNULL(
                                   (SELECT duration_seconds FROM media_files WHERE media_id = prev_media_id), 
                                   0
                               ), 
                               0) AS start_offset,
            @prev_playlist_id := playlist_id,
            @prev_media_id := media_id
        FROM (
            SELECT 
                pi.item_id, 
                pi.playlist_id, 
                pi.media_id,
                pi.position,
                mf.media_id AS prev_media_id
            FROM 
                playlist_items pi
            JOIN 
                media_files mf ON pi.media_id = mf.media_id
            WHERE 
                pi.playlist_id = p_playlist_id
            ORDER BY 
                pi.position
        ) AS sorted_items
        JOIN (SELECT @running_time := 0, @prev_playlist_id := NULL, @prev_media_id := NULL) AS vars
    ) AS calculated ON pi.item_id = calculated.item_id
    JOIN media_files mf ON pi.media_id = mf.media_id
    SET 
        pi.scheduled_start_time = DATE_ADD(v_start_time, INTERVAL calculated.start_offset SECOND),
        pi.scheduled_end_time = DATE_ADD(v_start_time, INTERVAL calculated.start_offset + mf.duration_seconds SECOND)
    WHERE 
        pi.playlist_id = p_playlist_id;
    
    -- Update playlist total duration
    UPDATE playlists
    SET total_duration_seconds = (
        SELECT SUM(mf.duration_seconds)
        FROM playlist_items pi
        JOIN media_files mf ON pi.media_id = mf.media_id
        WHERE pi.playlist_id = p_playlist_id
    )
    WHERE playlist_id = p_playlist_id;
    
    -- Log the event
    INSERT INTO event_logs (
        channel_id,
        event_type,
        event_category,
        message
    ) VALUES (
        v_channel_id,
        'info',
        'playlist',
        CONCAT('Playlist timing updated for playlist ID: ', p_playlist_id)
    );
END //

-- Procedure to scan media files for a channel
CREATE PROCEDURE sp_scan_channel_media(IN p_channel_id INT)
BEGIN
    DECLARE v_storage_root VARCHAR(255);
    DECLARE v_media_extensions VARCHAR(255);
    DECLARE v_min_duration INT;
    DECLARE v_max_duration INT;
    
    -- Get channel storage root
    SELECT storage_root INTO v_storage_root FROM channels WHERE channel_id = p_channel_id;
    
    -- Get system settings for media scanning
    SELECT 
        setting_value INTO v_media_extensions 
    FROM system_settings 
    WHERE setting_key = 'media_file_extensions';
    
    SELECT 
        setting_value INTO v_min_duration 
    FROM system_settings 
    WHERE setting_key = 'media_min_duration';
    
    SELECT 
        setting_value INTO v_max_duration 
    FROM system_settings 
    WHERE setting_key = 'media_max_duration';
    
    -- Log start of scanning
    INSERT INTO event_logs (
        channel_id,
        event_type,
        event_category,
        message,
        details
    ) VALUES (
        p_channel_id,
        'info',
        'media',
        'Starting media file scan',
        JSON_OBJECT('storage_root', v_storage_root)
    );
    
    -- Create temporary table for files found in directory
    CREATE TEMPORARY TABLE IF NOT EXISTS temp_media_files (
        file_path VARCHAR(255) NOT NULL PRIMARY KEY,
        file_name VARCHAR(255) NOT NULL,
        file_size BIGINT,
        last_modified TIMESTAMP,
        file_hash VARCHAR(64)
    );
    
    -- Clear temporary table
    TRUNCATE TABLE temp_media_files;
    
    -- Note: In a real implementation, you would need to use a system that can interface with the filesystem
    -- This is a placeholder for the concept - actual implementation would depend on your environment
    
    -- For MySQL, you would typically use a UDF (User Defined Function) or external program
    -- Here we assume the files are already inserted into the temp table by an external process
    
    -- Mark existing files as not found initially
    UPDATE media_files 
    SET scanned_at = NULL 
    WHERE channel_id = p_channel_id;
    
    -- Update existing files that were found
    UPDATE media_files mf
    JOIN temp_media_files tmp ON mf.file_path = tmp.file_path
    SET 
        mf.file_size = tmp.file_size,
        mf.last_modified = tmp.last_modified,
        mf.file_hash = tmp.file_hash,
        mf.scanned_at = NOW(),
        mf.updated_at = NOW()
    WHERE 
        mf.channel_id = p_channel_id;
    
    -- Insert new files that weren't in the database
    INSERT INTO media_files (
        channel_id,
        file_path,
        file_name,
        file_size,
        last_modified,
        file_hash,
        scanned_at,
        created_at,
        updated_at
    )
    SELECT 
        p_channel_id,
        tmp.file_path,
        tmp.file_name,
        tmp.file_size,
        tmp.last_modified,
        tmp.file_hash,
        NOW(),
        NOW(),
        NOW()
    FROM 
        temp_media_files tmp
    LEFT JOIN 
        media_files mf ON tmp.file_path = mf.file_path AND mf.channel_id = p_channel_id
    WHERE 
        mf.media_id IS NULL;
    
    -- Log completion of scanning
    INSERT INTO event_logs (
        channel_id,
        event_type,
        event_category,
        message,
        details
    ) VALUES (
        p_channel_id,
        'info',
        'media',
        'Completed media file scan',
        JSON_OBJECT(
            'files_scanned', (SELECT COUNT(*) FROM temp_media_files),
            'new_files', (SELECT COUNT(*) FROM media_files WHERE channel_id = p_channel_id AND scanned_at = NOW()),
            'updated_files', (SELECT COUNT(*) FROM media_files WHERE channel_id = p_channel_id AND scanned_at = NOW() AND created_at < scanned_at)
        )
    );
    
    -- Clean up
    DROP TEMPORARY TABLE IF EXISTS temp_media_files;
END //

-- Procedure to scan all channels media
CREATE PROCEDURE sp_scan_all_channels_media()
BEGIN
    DECLARE done INT DEFAULT FALSE;
    DECLARE v_channel_id INT;
    DECLARE v_channel_name VARCHAR(100);
    DECLARE channel_cursor CURSOR FOR SELECT channel_id, channel_name FROM channels WHERE enabled = TRUE;
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = TRUE;
    
    -- Log start of full scan
    INSERT INTO event_logs (
        event_type,
        event_category,
        message
    ) VALUES (
        'info',
        'system',
        'Starting full media scan for all channels'
    );
    
    OPEN channel_cursor;
    
    read_loop: LOOP
        FETCH channel_cursor INTO v_channel_id, v_channel_name;
        IF done THEN
            LEAVE read_loop;
        END IF;
        
        -- Scan media for this channel
        CALL sp_scan_channel_media(v_channel_id);
    END LOOP;
    
    CLOSE channel_cursor;
    
    -- Log completion of full scan
    INSERT INTO event_logs (
        event_type,
        event_category,
        message
    ) VALUES (
        'info',
        'system',
        'Completed full media scan for all channels'
    );
END //

-- Procedure to update media file duration
CREATE PROCEDURE sp_update_media_duration(
    IN p_media_id INT,
    IN p_duration_seconds INT
)
BEGIN
    DECLARE v_channel_id INT;
    
    -- Get channel ID for logging
    SELECT channel_id INTO v_channel_id FROM media_files WHERE media_id = p_media_id;
    
    -- Update duration
    UPDATE media_files
    SET 
        duration_seconds = p_duration_seconds,
        scanned_at = NOW(),
        updated_at = NOW()
    WHERE 
        media_id = p_media_id;
    
    -- Log the update
    INSERT INTO event_logs (
        channel_id,
        event_type,
        event_category,
        message,
        details
    ) VALUES (
        v_channel_id,
        'info',
        'media',
        'Updated media file duration',
        JSON_OBJECT(
            'media_id', p_media_id,
            'duration_seconds', p_duration_seconds
        )
    );
    
    -- Update any playlists that include this media
    UPDATE playlists p
    JOIN playlist_items pi ON p.playlist_id = pi.playlist_id
    SET p.total_duration_seconds = (
        SELECT SUM(mf.duration_seconds)
        FROM playlist_items pi2
        JOIN media_files mf ON pi2.media_id = mf.media_id
        WHERE pi2.playlist_id = p.playlist_id
    )
    WHERE pi.media_id = p_media_id;
END //

DELIMITER ;

-- Triggers

DELIMITER //

-- Trigger to maintain playlist item ordering
CREATE TRIGGER trg_maintain_playlist_order
BEFORE INSERT ON playlist_items
FOR EACH ROW
BEGIN
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
END //

-- Trigger to update playlist timing when items are added
CREATE TRIGGER trg_update_playlist_on_item_add
AFTER INSERT ON playlist_items
FOR EACH ROW
BEGIN
    -- Call the stored procedure to update playlist timing
    CALL sp_update_playlist_timing(NEW.playlist_id);
END //

-- Trigger to update playlist timing when items are updated
CREATE TRIGGER trg_update_playlist_on_item_update
AFTER UPDATE ON playlist_items
FOR EACH ROW
BEGIN
    IF OLD.media_id != NEW.media_id OR OLD.position != NEW.position THEN
        -- Call the stored procedure to update playlist timing
        CALL sp_update_playlist_timing(NEW.playlist_id);
    END IF;
END //

-- Trigger to update playlist timing when items are deleted
CREATE TRIGGER trg_update_playlist_on_item_delete
AFTER DELETE ON playlist_items
FOR EACH ROW
BEGIN
    -- Call the stored procedure to update playlist timing
    CALL sp_update_playlist_timing(OLD.playlist_id);
    
    -- Also reorder the remaining items to maintain sequence
    SET @pos := 0;
    UPDATE playlist_items
    SET position = (@pos := @pos + 1)
    WHERE playlist_id = OLD.playlist_id
    ORDER BY position;
END //

-- Trigger to log media file changes
CREATE TRIGGER trg_log_media_changes
AFTER UPDATE ON media_files
FOR EACH ROW
BEGIN
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
END //

DELIMITER ;

-- Grant appropriate permissions
GRANT SELECT, INSERT, UPDATE, DELETE, EXECUTE ON tvheadend.* TO 'tvheadend'@'localhost';