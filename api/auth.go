package api

func Register(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
	})
}


func Login(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
	})
}