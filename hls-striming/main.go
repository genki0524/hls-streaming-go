package main

import (
	"time"

	"github.com/gin-gonic/gin"
)

type Program struct {
	start_time    time.Time
	duration_sec  int32
	program_type  string
	path_template string
}

func main() {
	router := gin.Default()

	router.Run("localhost:8000")
}
