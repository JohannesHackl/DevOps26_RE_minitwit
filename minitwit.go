package main

import (
	"crypto/md5"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	PER_PAGE   int    = 30
	SECRET_KEY string = "development key"
)

var db *gorm.DB

type User struct {
	UserID   int    `gorm:"column:user_id;primaryKey;autoIncrement"`
	Username string `gorm:"column:username;not null"`
	Email    string `gorm:"column:email;not null"`
	PWHash   string `gorm:"column:pw_hash;not null"`
}

type Message struct {
	MessageID int    `gorm:"column:message_id;primaryKey;autoIncrement"`
	AuthorID  int    `gorm:"column:author_id;not null"`
	Text      string `gorm:"column:text;not null"`
	PubDate   int64  `gorm:"column:pub_date"`
	Flagged   int    `gorm:"column:flagged"`
	Author    User   `gorm:"foreignKey:AuthorID;references:UserID"`
}

func (Message) TableName() string  { return "messages" }
func (User) TableName() string     { return "users" }

type Follower struct {
	WhoID  int `gorm:"column:who_id"`
	WhomID int `gorm:"column:whom_id"`
	Whom   User `gorm:"foreignKey:WhomID"`
}

func (Follower) TableName() string { return "follower" }

func main() {
	err := init_db()
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		return
	}

	router := create_app()
	router.Run(":5001")
}

func create_app() *gin.Engine {
	router := gin.Default()
	store := cookie.NewStore([]byte(SECRET_KEY))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 24,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
	router.Use(sessions.Sessions("mysession", store))
	router.Use(before_request)

	funcMap := template.FuncMap{
		"gravatar_url":    gravatar_url,
		"format_datetime": format_datetime,
	}
	router.SetFuncMap(funcMap)
	router.LoadHTMLGlob("./templates/*")
	router.Static("/static", "./static")
	router.StaticFile("/favicon.ico", "./static/favicon.ico")

	simAuth := gin.BasicAuth(gin.Accounts{
		"simulator": "super_safe!",
	})

	api := router.Group("/api")
	api.Use(simAuth)
	{
		api.GET("/latest", get_latest_value)
		api.POST("/register", post_register)
		api.GET("/msgs", get_messages)
		api.GET("/msgs/:username", get_messages_per_user)
		api.POST("/msgs/:username", post_messages_per_user)
		api.GET("/fllws/:username", get_follow)
		api.POST("/fllws/:username", post_follow)
	}

	router.GET("/register", registerGet)
	router.POST("/register", registerPost)

	router.GET("/", timeline)
	router.GET("/public", public_timeline)
	router.GET("/logout", logout)
	router.GET("/:username", user_timeline)
	router.GET("/:username/follow", follow_user)
	router.GET("/:username/unfollow", unfollow_user)

	router.POST("/add_message", add_message)
	router.GET("/login", loginGet)
	router.POST("/login", loginPost)

	return router
}

func connect_db() (*gorm.DB, error) {
	host := os.Getenv("DB_ADDR")
	if host == "" {
		fmt.Println("WARNING: DB_ADDR environment variable is empty!")
		host = "localhost"
	}
	dsn := fmt.Sprintf("host=%s user=minitwit password=minitwit dbname=minitwit sslmode=disable", host)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

func init_db() error {
	var err error
	db, err = connect_db()
	return err
}

func get_user_id(username string) (int, error) {
	var user User
	result := db.Where("username = ?", username).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, result.Error
	}
	return user.UserID, nil
}

func format_datetime(timestamp int64) string {
	return time.Unix(timestamp, 0).Format("2006-01-02 @ 15:04")
}

func gravatar_url(email string, size int) string {
	email = strings.ToLower(strings.TrimSpace(email))
	hash := md5.Sum([]byte(email))
	return fmt.Sprintf("https://www.gravatar.com/avatar/%x?d=identicon&s=%d", hash, size)
}

func before_request(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")

	if userID != nil {
		var user User
		if err := db.First(&user, userID).Error; err == nil {
			c.Set("user", user)
		}
	}
	c.Next()
}

// Router functions

func timeline(c *gin.Context) {
	val, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusFound, "/public")
		return
	}
	user := val.(User)

	var messages []Message
	db.Preload("Author").Where("flagged = 0 AND (author_id = ? OR author_id IN (SELECT whom_id FROM follower WHERE who_id = ?))", user.UserID, user.UserID).
		Order("pub_date DESC").Limit(PER_PAGE).Find(&messages)

	render(c, http.StatusOK, "timeline.html", gin.H{
		"messages": messages,
		"endpoint": "timeline",
	})
}

func public_timeline(c *gin.Context) {
	var messages []Message
	db.Preload("Author").Where("flagged = 0").
		Order("pub_date DESC").Limit(PER_PAGE).Find(&messages)

	render(c, http.StatusOK, "timeline.html", gin.H{
		"messages": messages,
		"endpoint": "public_timeline",
	})
}

func user_timeline(c *gin.Context) {
	username := c.Param("username")
	var profileUser User
	if err := db.Where("username = ?", username).First(&profileUser).Error; err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	followed := false
	if val, exists := c.Get("user"); exists {
		currUser := val.(User)
		var count int64
		db.Model(&Follower{}).Where("who_id = ? AND whom_id = ?", currUser.UserID, profileUser.UserID).Count(&count)
		followed = count > 0
	}

	var messages []Message
	db.Preload("Author").Where("author_id = ?", profileUser.UserID).
		Order("pub_date DESC").Limit(PER_PAGE).Find(&messages)

	render(c, http.StatusOK, "timeline.html", gin.H{
		"messages":     messages,
		"profile_user": profileUser,
		"followed":     followed,
		"endpoint":     "user_timeline",
	})
}

func follow_user(c *gin.Context) {
	val, exists := c.Get("user")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	currUser := val.(User)
	username := c.Param("username")

	whomID, err := get_user_id(username)
	if err != nil || whomID == 0 {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	db.Create(&Follower{WhoID: currUser.UserID, WhomID: whomID})
	c.Redirect(http.StatusFound, "/"+username)
}

func unfollow_user(c *gin.Context) {
	val, exists := c.Get("user")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	currUser := val.(User)
	username := c.Param("username")

	whomID, err := get_user_id(username)
	if err != nil || whomID == 0 {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	db.Where("who_id = ? AND whom_id = ?", currUser.UserID, whomID).Delete(&Follower{})
	c.Redirect(http.StatusFound, "/"+username)
}

func add_message(c *gin.Context) {
	val, exists := c.Get("user")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	user := val.(User)
	text := c.PostForm("text")

	if text != "" {
		msg := Message{AuthorID: user.UserID, Text: text, PubDate: time.Now().Unix(), Flagged: 0}
		if err := db.Create(&msg).Error; err == nil {
			session := sessions.Default(c)
			session.AddFlash("Your message was recorded")
			session.Save()
		}
	}

	c.Redirect(http.StatusFound, "/")
}

// --- Auth Handlers ---

func loginGet(c *gin.Context) {
	if _, exists := c.Get("user"); exists {
		c.Redirect(http.StatusFound, "/")
		return
	}
	render(c, http.StatusOK, "login.html", nil)
}

func loginPost(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	var user User
	err := db.Where("username = ?", username).First(&user).Error

	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PWHash), []byte(password)) != nil {
		render(c, http.StatusOK, "login.html", gin.H{"error": "Invalid username or password"})
		return
	}

	session := sessions.Default(c)
	session.AddFlash("You were logged in")
	session.Set("user_id", user.UserID)
	session.Save()
	c.Redirect(http.StatusFound, "/")
}

func registerGet(c *gin.Context) {
	if _, exists := c.Get("user"); exists {
		c.Redirect(http.StatusFound, "/")
		return
	}
	render(c, http.StatusOK, "register.html", nil)
}

func registerPost(c *gin.Context) {
	username := c.PostForm("username")
	email := c.PostForm("email")
	password := c.PostForm("password")
	passwordConf := c.PostForm("password2")

	var errorStr string
	if username == "" {
		errorStr = "You have to enter a username"
	} else if email == "" || !strings.Contains(email, "@") {
		errorStr = "You have to enter a valid email address"
	} else if password == "" {
		errorStr = "You have to enter a password"
	} else if password != passwordConf {
		errorStr = "The two passwords do not match"
	} else {
		var existing User
		if err := db.Where("username = ?", username).First(&existing).Error; err == nil {
			errorStr = "The username is already taken"
		}
	}

	if errorStr != "" {
		render(c, http.StatusOK, "register.html", gin.H{"error": errorStr})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	newUser := User{Username: username, Email: email, PWHash: string(hashedPassword)}
	if err := db.Create(&newUser).Error; err != nil {
		render(c, http.StatusOK, "register.html", gin.H{"error 404": err})
		return
	}

	c.Redirect(http.StatusFound, "/login")
}

func logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("user_id")
	session.Save()
	c.Redirect(http.StatusFound, "/public")
}

func render(c *gin.Context, code int, name string, data gin.H) {
	if data == nil {
		data = gin.H{}
	}

	if user, exists := c.Get("user"); exists {
		data["user"] = user
	}

	session := sessions.Default(c)
	flashes := session.Flashes()

	if len(flashes) > 0 {
		data["flashes"] = flashes
		_ = session.Save()
	}
	c.HTML(code, name, data)
}


