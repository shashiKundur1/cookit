package api

import (
	"net/http"

	"cookit/internal/history"

	"github.com/gin-gonic/gin"
)

type Server struct {
	history *history.Store
	router  *gin.Engine
}

func New(h *history.Store) *Server {
	gin.SetMode(gin.ReleaseMode)

	s := &Server{
		history: h,
		router:  gin.New(),
	}

	s.router.Use(gin.Recovery())
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.GET("/health", s.healthHandler)
	s.router.GET("/history", s.historyHandler)
	s.router.GET("/history/last", s.lastFolderHandler)
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "app": "cookit"})
}

func (s *Server) historyHandler(c *gin.Context) {
	entries := s.history.GetAll()
	c.JSON(http.StatusOK, gin.H{"entries": entries, "count": len(entries)})
}

func (s *Server) lastFolderHandler(c *gin.Context) {
	last := s.history.GetLastFolder()
	if last == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "no history yet"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"lastFolder": last})
}
