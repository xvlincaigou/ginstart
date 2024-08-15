package main

import (
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Todo struct {
	gorm.Model
	Title       string `json:"title"`
	Description string `json:"description"`
}

// JWT secret
var jwtSecret = []byte("ewidmqwxuicewhfnewuixhrmrWEE2rwde")

// JWT claims
type Claims struct {
	UserID uint `json:"user_id"`
	jwt.StandardClaims
}

// GenerateToken generates a JWT token for the given user ID
func GenerateToken(userID uint) (string, error) {
	claims := Claims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// Middleware to check JWT token
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Authorization header is missing"})
			c.Abort()
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(authHeader, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}
}

// 这是缓存，表示缓存的有效期为5分钟，每10分钟更新一次缓存
var memCache = cache.New(5*time.Minute, 10*time.Minute)

func main() {
	router := gin.Default()

	db, err := gorm.Open(sqlite.Open("todo.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&Todo{})

	// Public routes
	router.POST("/login", loginHandler)

	// Protected routes
	protected := router.Group("/")
	protected.Use(JWTMiddleware())
	{
		protected.POST("/todos", createTodoHandler(db))
		protected.GET("/todos", getTodosHandler(db))
		protected.GET("/todos/:id", getTodoHandler(db))
		protected.PUT("/todos/:id", updateTodoHandler(db))
		protected.DELETE("/todos/:id", deleteTodoHandler(db))
		protected.GET("/manytodos", getManyTodosHandler(db))
		protected.POST("/manytodos", createManyTodosHandler(db))
		protected.GET("/cached-todos", cachedTodosHandler(db))
	}

	router.Run(":8080")
}

func loginHandler(c *gin.Context) {
	userID, err := strconv.Atoi(c.Query("userid"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid userid parameter"})
		return
	}

	token, err := GenerateToken(uint(userID))
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(200, gin.H{"token": token})
}

func createTodoHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var todo Todo
		if err := c.ShouldBindJSON(&todo); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		db.Create(&todo)
		c.JSON(200, todo)
	}
}

func getTodosHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var todos []Todo
		db.Find(&todos)
		c.JSON(200, todos)
	}
}

func getTodoHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var todo Todo
		todoID := c.Param("id")

		result := db.First(&todo, todoID)
		if result.Error != nil {
			c.JSON(404, gin.H{"error": "Todo not found"})
			return
		}

		c.JSON(200, todo)
	}
}

func updateTodoHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func deleteTodoHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var todo Todo
		todoID := c.Param("id")

		result := db.First(&todo, todoID)
		if result.Error != nil {
			c.JSON(404, gin.H{"error": "Todo not found"})
			return
		}

		db.Delete(&todo)
		c.JSON(200, gin.H{"message": "Todo deleted successfully"})
	}
}

func getManyTodosHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		count, err := strconv.Atoi(c.DefaultQuery("count", "5"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid count parameter"})
			return
		}
		offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid offset parameter"})
			return
		}

		var todos []Todo
		db.Limit(count).Offset(offset).Find(&todos)
		c.JSON(200, todos)
	}
}

func createManyTodosHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var todos []Todo
		if err := c.ShouldBindJSON(&todos); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		err := db.Transaction(func(tx *gorm.DB) error {
			for _, todo := range todos {
				if err := tx.Create(&todo).Error; err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, todos)
	}
}

func cachedTodosHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cached, found := memCache.Get("todos"); found {
			c.JSON(200, cached)
			return
		}

		var todos []Todo
		db.Find(&todos)

		memCache.Set("todos", todos, cache.DefaultExpiration)
		c.JSON(200, todos)
	}
}
