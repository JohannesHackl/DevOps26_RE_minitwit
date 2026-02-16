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
	router.Use(sessions.Sessions("mysession", store))

	router.Use(before_request)

	funcMap := template.FuncMap{
		"gravatar_url":    gravatar_url,
		"format_datetime": format_datetime,
	}
	router.SetFuncMap(funcMap)

	router.LoadHTMLGlob("./templates/*")
	router.Static("/static", "./static")

	// Routes
	router.GET("/", timeline)
	router.GET("/public", public_timeline)
	router.GET("/logout", logout)
	router.GET("/:username", user_timeline)
	router.GET("/:username/follow", follow_user)
	router.GET("/:username/unfollow", unfollow_user)

	router.POST("/add_message", add_message)
	router.GET("/login", loginGet)
	router.POST("/login", loginPost)
	router.GET("/register", registerGet)
	router.POST("/register", registerPost)

	return router
}

func connect_db() (*sql.DB, error) {
	return sql.Open("postgres", "host=192.168.56.10 user=minitwit password=minitwit dbname=minitwit-db sslmode=disable")
}

func init_db() error {
	var err error
	db, err = connect_db()
	if err != nil {
		return err
	}

	schema, err := os.ReadFile(SCHEMA)
	if err != nil {
		return fmt.Errorf("could not read schema file: %w", err)
	}
	_, err = db.Exec(string(schema))
	return err
}

func get_user_id(username string) (int, error) {
	var id int
	query := "SELECT user_id FROM users WHERE username = ?"
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
		err := db.QueryRow("SELECT user_id, username, email, pw_hash from users WHERE user_id = ?", userID).
			Scan(&user.UserID, &user.Username, &user.Email, &user.PWHash)
		if err == nil {
			c.Set("user", user)
		}
	}
	c.Next()
}

// Rrouter functions

func timeline(c *gin.Context) {
	val, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusFound, "/public")
		return
	}
	user := val.(User)

	query := `
        SELECT message.*, user.* FROM message, user
        WHERE message.flagged = 0 AND message.author_id = user.user_id AND (
            user.user_id = ? OR
            user.user_id IN (SELECT whom_id FROM follower WHERE who_id = ?))
        ORDER BY message.pub_date DESC LIMIT ?`

	messages, _ := queryTimeline(query, user.UserID, user.UserID, PER_PAGE)

	render(c, http.StatusOK, "timeline.html", gin.H{
		"messages": messages,
		"endpoint": "timeline",
	})
}

func public_timeline(c *gin.Context) {
	query := `
        SELECT message.*, user.* FROM message, user
        WHERE message.flagged = 0 AND message.author_id = user.user_id
        ORDER BY message.pub_date DESC LIMIT ?`

	messages, _ := queryTimeline(query, PER_PAGE)

	render(c, http.StatusOK, "timeline.html", gin.H{
		"messages": messages,
		"endpoint": "public_timeline",
	})
}

func user_timeline(c *gin.Context) {
	username := c.Param("username")
	var profileUser User
	err := db.QueryRow("SELECT user_id, username, email, pw_hash from users WHERE username = ?", username).
		Scan(&profileUser.UserID, &profileUser.Username, &profileUser.Email, &profileUser.PWHash)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	followed := false
	if val, exists := c.Get("user"); exists {
		currUser := val.(User)
		var count int
		db.QueryRow("SELECT 1 FROM follower WHERE who_id = ? AND whom_id = ?", currUser.UserID, profileUser.UserID).Scan(&count)
		followed = count > 0
	}

	query := `
        SELECT message.*, user.* FROM message, user 
        WHERE user.user_id = message.author_id AND user.user_id = ?
        ORDER BY message.pub_date DESC LIMIT ?`

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

	db.Exec("INSERT INTO follower (who_id, whom_id) VALUES (?, ?)", currUser.UserID, whomID)
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

	db.Exec("DELETE FROM follower WHERE who_id = ? AND whom_id = ?", currUser.UserID, whomID)
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
		_, err := db.Exec("INSERT INTO message (author_id, text, pub_date, flagged) VALUES (?, ?, ?, 0)",
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
	err := db.QueryRow("SELECT user_id, username, email, pw_hash from users WHERE username = ?", username).
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
		err := db.QueryRow("SELECT user_id from users WHERE username = ?", username).Scan(&existingID)
		if err == nil {
			errorStr = "The username is already taken"
		}
	}

	if errorStr != "" {
		render(c, http.StatusOK, "register.html", gin.H{"error": errorStr})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	db.Exec("INSERT INTO user (username, email, pw_hash) VALUES (?, ?, ?)",
		username, email, string(hashedPassword))

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
