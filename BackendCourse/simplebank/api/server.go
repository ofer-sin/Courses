package api

import (
	db "github.com/ofer-sin/Courses/BackendCourse/simplebank/db/sqlc"

	"github.com/gin-gonic/gin"
)

// Server serves HTTP requests for our banking service.
// It contains the router and the store
type Server struct {
	store  db.Store
	router *gin.Engine
}

// NewServer creates a new HTTP server and sets up routing
// with the provided store.
func NewServer(store db.Store) *Server {
	router := gin.Default()
	server := &Server{store: store, router: router}

	// Set up routes
	// When a request is made to /accounts, with the indicated handler method of the server is called
	router.POST("/accounts", server.createAccount)
	router.GET("/accounts/:id", server.getAccount) // the ':' indicates a uri (path) parameter
	router.GET("/accounts", server.listAccounts)

	return server
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
