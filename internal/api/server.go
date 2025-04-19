package api

import (
	"net/http"
	"strconv"

	"github.com/euacreations/tvheadend/internal/services"
	"github.com/gin-gonic/gin"
)

type Server struct {
	router         *gin.Engine
	channelService *services.ChannelService
}

func NewServer(channelService *services.ChannelService) *Server {
	router := gin.Default()
	s := &Server{
		router:         router,
		channelService: channelService,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	api := s.router.Group("/api/v1")
	{
		api.GET("/channels", s.listChannels)
		api.POST("/channels/:id/start", s.startChannel)
		api.POST("/channels/:id/stop", s.stopChannel)
		api.GET("/channels/:id/status", s.channelStatus)
	}
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
