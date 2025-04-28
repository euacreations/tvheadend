package api

import (
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
		api.GET("/channels/:id/", s.getChannel)
		api.POST("/channels/:id/start", s.startChannel)
		api.POST("/channels/:id/stop", s.stopChannel)
		api.GET("/channels/:id/status", s.channelStatus)
		api.POST("/channels/:id/scan", s.scanMedia)
		api.GET("/playlists/:id", s.getPlaylist)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"channels": channels})
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

	running, err := s.channelService.CheckChannelStatus(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"running": running})
}

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

func (s *Server) getPlaylist(c *gin.Context) {
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
