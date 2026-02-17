package main

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const (
	DATABASE   string = "./tmp/minitwit.db"
	SCHEMA     string = "schema.sql"
	PER_PAGE   int    = 30
	DEBUG      bool   = true
	SECRET_KEY string = "development key"
)

var db *sql.DB

type User struct {
	UserID   int
	Username string
	Email    string
	PWHash   string
}

type TimelineMessage struct {
	MessageID int
	AuthorID  int
	Text      string
	PubDate   int64
	Flagged   int
	UserID    int
	Username  string
	Email     string
	PWHash    string
}

func main() {
	err := init_db()
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		return
	}
	defer db.Close()

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

func connect_db() (*sql.DB, error) {
	host := os.Getenv("DB_ADDR")
	if host == "" {
		fmt.Println("WARNING: DB_ADDR environment variable is empty!")
		host = "localhost"
	}
	connStr := fmt.Sprintf("host=%s user=minitwit password=minitwit dbname=minitwit sslmode=disable", host)
	return sql.Open("postgres", connStr)
}

func init_db() error {
	var err error
	db, err = connect_db()
	return err
}

func get_user_id(username string) (int, error) {
	var id int
	query := "SELECT user_id FROM users WHERE username = $1"
	err := db.QueryRow(query, username).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return id, nil
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
		err := db.QueryRow("SELECT user_id, username, email, pw_hash FROM users WHERE user_id = $1", userID).
			Scan(&user.UserID, &user.Username, &user.Email, &user.PWHash)
		if err == nil {
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

	query := `
        SELECT messages.*, users.* FROM messages, users
        WHERE messages.flagged = 0 AND messages.author_id = users.user_id AND (
            users.user_id = $1 OR
            users.user_id IN (SELECT whom_id FROM follower WHERE who_id = $2))
        ORDER BY messages.pub_date DESC LIMIT $3`

	messages, err := queryTimeline(query, user.UserID, user.UserID, PER_PAGE)
	if err != nil {
		fmt.Println(err)
	}

	render(c, http.StatusOK, "timeline.html", gin.H{
		"messages": messages,
		"endpoint": "timeline",
	})
}

func public_timeline(c *gin.Context) {
	query := `
        SELECT messages.*, users.* FROM messages, users
        WHERE messages.flagged = 0 AND messages.author_id = users.user_id
        ORDER BY messages.pub_date DESC LIMIT $1`

	messages, _ := queryTimeline(query, PER_PAGE)

	render(c, http.StatusOK, "timeline.html", gin.H{
		"messages": messages,
		"endpoint": "public_timeline",
	})
}

func user_timeline(c *gin.Context) {
	username := c.Param("username")
	var profileUser User
	err := db.QueryRow("SELECT user_id, username, email, pw_hash FROM users WHERE username = $1", username).
		Scan(&profileUser.UserID, &profileUser.Username, &profileUser.Email, &profileUser.PWHash)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	followed := false
	if val, exists := c.Get("user"); exists {
		currUser := val.(User)
		var count int
		db.QueryRow("SELECT 1 FROM follower WHERE who_id = $1 AND whom_id = $2", currUser.UserID, profileUser.UserID).Scan(&count)
		followed = count > 0
	}

	query := `
        SELECT messages.*, users.* FROM messages, users 
        WHERE users.user_id = messages.author_id AND users.user_id = $1
        ORDER BY messages.pub_date DESC LIMIT $2`

	messages, _ := queryTimeline(query, profileUser.UserID, PER_PAGE)

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

	db.Exec("INSERT INTO follower (who_id, whom_id) VALUES ($1, $2)", currUser.UserID, whomID)
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

	db.Exec("DELETE FROM follower WHERE who_id = $1 AND whom_id = $2", currUser.UserID, whomID)
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
		_, err := db.Exec("INSERT INTO messages (author_id, text, pub_date, flagged) VALUES ($1, $2, $3, 0)",
			user.UserID, text, time.Now().Unix())

		if err == nil {
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
	err := db.QueryRow("SELECT user_id, username, email, pw_hash FROM users WHERE username = $1", username).
		Scan(&user.UserID, &user.Username, &user.Email, &user.PWHash)

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
		var existingID int
		err := db.QueryRow("SELECT user_id FROM users WHERE username = $1", username).Scan(&existingID)
		if err == nil {
			errorStr = "The username is already taken"
		}
	}

	if errorStr != "" {
		render(c, http.StatusOK, "register.html", gin.H{"error": errorStr})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	_, err := db.Exec("INSERT INTO users (username, email, pw_hash) VALUES ($1, $2, $3)",
		username, email, string(hashedPassword))
	if err != nil {
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

// Database Helper

func queryTimeline(query string, args ...interface{}) ([]TimelineMessage, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []TimelineMessage
	for rows.Next() {
		var tm TimelineMessage
		err := rows.Scan(&tm.MessageID, &tm.AuthorID, &tm.Text, &tm.PubDate, &tm.Flagged,
			&tm.UserID, &tm.Username, &tm.Email, &tm.PWHash)
		if err == nil {
			messages = append(messages, tm)
		}
	}
	return messages, nil
}
