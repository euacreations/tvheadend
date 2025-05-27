package api

import (
	"database/sql"
	"math"
	"net/http"
	"strconv"

	"github.com/euacreations/tvheadend/internal/models"
	"github.com/euacreations/tvheadend/internal/services"
	"github.com/gin-gonic/gin"
)

type Server struct {
	router           *gin.Engine
	channelService   *services.ChannelService
	mediaScanner     *services.MediaScanner
	PlaylistExecutor *services.PlaylistExecutor
	overlayService   *services.OverlayService
}

func NewServer(
	channelService *services.ChannelService,
	mediaScanner *services.MediaScanner,
	PlaylistExecutor *services.PlaylistExecutor,
	overlayService *services.OverlayService,
) *Server {
	router := gin.Default()
	s := &Server{
		router:           router,
		channelService:   channelService,
		mediaScanner:     mediaScanner,
		PlaylistExecutor: PlaylistExecutor,
		overlayService:   overlayService,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	api := s.router.Group("/api/v1")
	{
		api.GET("/channels", s.listChannels)
		api.GET("/channels/:id", s.getChannel)
		api.GET("/channels/:id/start", s.startChannel)
		api.POST("/channels/:id/stop", s.stopChannel)
		api.GET("/channels/:id/status", s.channelStatus)
		api.POST("/channels/:id/scan", s.scanMedia)
		api.GET("/channels/:id/playlists", s.getPlaylists)
		api.GET("/channels/:id/playlists/:playlistId", s.getPlaylist)
		api.GET("/channels/:id/media", s.getMediaFiles)
		api.POST("/overlays", s.createOverlay)
	}
}

func (s *Server) scanMedia(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	if err := s.mediaScanner.ScanChannelMedia(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "media scan initiated"})
}

func (s *Server) getChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	channel, err := s.channelService.GetChannel(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, channel)
}

func (s *Server) listChannels(c *gin.Context) {
	channels, err := s.channelService.GetAllChannels(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"channels": []*models.Channel{},
			"error":    err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"channels": channels,
		"error":    nil,
	})
}

func (s *Server) startChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	if err := s.channelService.StartChannel(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "channel started"})
}

func (s *Server) stopChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	if err := s.channelService.StopChannel(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "channel stopped"})
}

func (s *Server) channelStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	// Get the channel to ensure it exists
	channel, err := s.channelService.GetChannel(c.Request.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Get comprehensive status information
	state, isStreamerRunning, err := s.channelService.GetChannelStatus(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Prepare response with detailed status information
	statusResponse := gin.H{
		"channel_id":       channel.ChannelID,
		"name":             channel.ChannelName,
		"running":          state.Running && isStreamerRunning, // Only true if both DB state and actual streamer are running
		"current_position": state.CurrentPosition,
		"last_update_time": state.LastUpdateTime,
		"ffmpeg_pid":       state.FFmpegPID,
	}

	// If we have playlist information and a currently playing item, add it
	if state.CurrentPlaylistID > 0 {
		statusResponse["current_playlist_id"] = state.CurrentPlaylistID

		// Try to get playlist name if available
		playlist, _ := s.PlaylistExecutor.GetPlaylist(c.Request.Context(), state.CurrentPlaylistID)
		if playlist != nil {
			statusResponse["playlist_name"] = playlist.PlaylistName
		}
	}

	if state.CurrentItemID > 0 {
		statusResponse["current_item_id"] = state.CurrentItemID

		// Could add logic here to get the current item's details if needed
	}

	c.JSON(http.StatusOK, statusResponse)
}

// func (s *Server) channelStatus(c *gin.Context) {
// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
// 		return
// 	}

// 	running, err := s.channelService.CheckChan
// nelStatus(c.Request.Context(), id)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"running": running})
// }

func (s *Server) Start(addr string) error {
	return s.router.Run(addr)
}

func (s *Server) createOverlay(c *gin.Context) {
	var overlay models.Overlay
	if err := c.ShouldBindJSON(&overlay); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := s.overlayService.CreateOverlay(c.Request.Context(), &overlay)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (s *Server) getPlaylist1(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid playlist ID"})
		return
	}

	playlist, err := s.PlaylistExecutor.GetPlaylist(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	items, err := s.PlaylistExecutor.GetPlaylistItems(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"playlist": playlist,
		"items":    items,
	})
}

func (s *Server) getPlaylists(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid channel ID"})
		return
	}

	// Check if the channel exists
	_, err = s.channelService.GetChannel(c.Request.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Channel not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database error: " + err.Error(),
		})
		return
	}

	// Fetch the playlists
	playlists, err := s.PlaylistExecutor.GetPlaylists(c.Request.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Playlist not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database error: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, playlists)

}

func (s *Server) getPlaylist(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid channel ID"})
		return
	}

	playlistID, err := strconv.Atoi(c.Param("playlistId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid playlist ID"})
		return
	}

	// Check if the channel exists
	channel, err := s.channelService.GetChannel(c.Request.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Channel not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database error: " + err.Error(),
		})
		return
	}

	// Fetch the playlist
	playlist, err := s.PlaylistExecutor.GetPlaylist(c.Request.Context(), playlistID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Playlist not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database error: " + err.Error(),
		})
		return
	}

	// Verify that the playlist belongs to the specified channel
	if playlist.ChannelID != id {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Playlist does not belong to the specified channel",
		})
		return
	}

	// Fetch playlist items
	items, err := s.PlaylistExecutor.GetPlaylistItems(c.Request.Context(), playlistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch playlist items: " + err.Error(),
		})
		return
	}

	// Add items to the playlist
	//playlist.Items = items

	// Return successful response with the playlist data
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"playlist": playlist,
			"channel":  channel,
			"items":    items,
		},
	})
}

/*
func (s *Server) getMediaFiles(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid channel ID"})
		return
	}

	// Check if the channel exists
	_, err = s.channelService.GetChannel(c.Request.Context(), id)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Channel not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database error: " + err.Error(),
		})
		return
	}

	// Fetch the Media Files

	mediafiles, err := s.PlaylistExecutor.GetMediaFiles(c.Request.Context(), id)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Media Files not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database error: " + err.Error(),
		})
		return
	}
	// Return successful response with the playlist data
	c.JSON(http.StatusOK, mediafiles)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"channel":    channel,
			"mediafiles": mediafiles,
		},
	})
}

*/

func (s *Server) getMediaFiles(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid channel ID"})
		return
	}

	// Get pagination parameters from query string
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	// Check if the channel exists
	_, err = s.channelService.GetChannel(c.Request.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Channel not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database error: " + err.Error(),
		})
		return
	}

	// Get paginated media files
	mediafiles, err := s.PlaylistExecutor.GetMediaFiles(c.Request.Context(), id, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database error: " + err.Error(),
		})
		return
	}

	// Get total count for pagination metadata
	total, err := s.PlaylistExecutor.CountMediaFiles(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database error: " + err.Error(),
		})
		return
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	// Return response with pagination metadata
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    mediafiles,
		"pagination": gin.H{
			"total":        total,
			"total_pages":  totalPages,
			"current_page": page,
			"page_size":    pageSize,
		},
	})
}
