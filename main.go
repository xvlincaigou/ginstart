package main

import (
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 这个结构体使用gorm.Model作为它的基类，这个基类包含了创建时间、更新时间、删除时间等字段。
// 同时，增加了Title和Description两个字段。这两个标签制定了在序列化的时候，需要命名为title和description。
type Todo struct {
	gorm.Model
	Title       string `json:"title"`
	Description string `json:"description"`
}

/*
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		log.Printf("%s %s %d %s", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), duration)
	}
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-KEY")
		if apiKey == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "API key is missing"})
			return
		}
		c.Next()
	}
}

// UserController represents a user-related controller
type UserController struct{}

func (uc *UserController) GetUserInfo(c *gin.Context) {
	name := c.Param("name")
	c.String(200, "Hello, %s!", name)
}
*/

func main() {
	/**
	* Routing or router in web development is a mechanism where HTTP requests are routed to the code that handles them.
	* To put simply, in the Router you determine what should happen when a user visits a certain page.
	* Here, we are using the Gin framework to create a router.
	 */
	router := gin.Default()
	// Use attaches a global middleware to the router. i.e. the middleware attached through Use() will be
	// included in the handlers chain for every single request. Even 404, 405, static files...
	// router.Use(LoggerMiddleware())

	// Connect to the SQLite database
	db, err := gorm.Open(sqlite.Open("todo.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Auto-migrate the Todo model to create the table
	// 根据传入的结构体指针，自动创建或者更新表结构
	db.AutoMigrate(&Todo{})

	// Route to create a new Todo
	// ShouldBindJson 把请求的body中的json数据绑定到结构体中。
	router.POST("/todos", func(c *gin.Context) {
		var todo Todo
		if err := c.ShouldBindJSON(&todo); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		db.Create(&todo)
		c.JSON(200, todo)
	})

	router.GET("/todos", func(c *gin.Context) {
		var todos []Todo
		db.Find(&todos)
		c.JSON(200, todos)
	})

	// Route to get a specific Todo by ID
	router.GET("/todos/:id", func(c *gin.Context) {
		var todo Todo
		todoID := c.Param("id")

		// Retrieve the Todo from the database
		result := db.First(&todo, todoID)
		if result.Error != nil {
			c.JSON(404, gin.H{"error": "Todo not found"})
			return
		}

		c.JSON(200, todo)
	})

	router.PUT("/todos/:id", func(c *gin.Context) {
		var todo Todo
		todoID := c.Param("id")

		result := db.First(&todo, todoID)
		if result.Error != nil {
			c.JSON(404, gin.H{"error": "Todo not found"})
			return
		}

		var updatedTodo Todo
		if err := c.ShouldBindJSON(&updatedTodo); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		todo.Title = updatedTodo.Title
		todo.Description = updatedTodo.Description
		db.Save(&todo)
		c.JSON(200, todo)
	})

	router.DELETE("/todos/:id", func(c *gin.Context) {
		var todo Todo
		todoID := c.Param("id")

		result := db.First(&todo, todoID)
		if result.Error != nil {
			c.JSON(404, gin.H{"error": "Todo not found"})
			return
		}

		db.Delete(&todo)
		c.JSON(200, gin.H{"message": "Todo deleted successfully"})
	})

	/*
		// Define a route for the root URL
		// Context is a container for request data.
		router.GET("/", func(c *gin.Context) {
			// c.String returns a string with a status code of 200
			c.String(200, "Hello, World!")
		})

		// Define a route for goodbye
		router.GET("/goodbye", func(c *gin.Context) {
			c.String(200, "Goodbye!")
		})

		// Define a user controller
		userController := UserController{}
		// Route with a URL parameter
		router.GET("/hello/:name", userController.GetUserInfo)

		// Route with query parameters
		router.GET("/search", func(c *gin.Context) {
			query := c.DefaultQuery("q", "golang")
			c.String(200, "Search query: %s", query)
		})

		// Public routes (no authentication required)
		public := router.Group("/public")
		{
			public.GET("/info", func(c *gin.Context) {
				c.String(200, "Public information")
			})
			public.GET("/products", func(c *gin.Context) {
				c.String(200, "Public product list")
			})
		}

		// Private routes (require authentication)
		private := router.Group("/private")
		private.Use(AuthMiddleware())
		{
			private.GET("/data", func(c *gin.Context) {
				c.String(200, "Private data accessible after authentication")
			})
			private.POST("/create", func(c *gin.Context) {
				c.String(200, "Create a new resource")
			})
		}
	*/

	// Run the server on port 8080
	router.Run(":8080")
}
