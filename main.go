package main

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"database/sql"
	"encoding/hex"
	"fmt"
	"hash"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

// configuration
const (
	DATABASE  = "/tmp/minitwit.db"
	PER_PAGE  = 30
	SecretKey = "development key"
)

var db *sql.DB

// User represents a user record
type User struct {
	UserID   int
	Username string
	Email    string
	PwHash   string
}

// Message represents a message record joined with user info
type Message struct {
	MessageID int
	AuthorID  int
	Text      string
	PubDate   int64
	Flagged   int
	Username  string
	Email     string
}

// connectDb returns a database connection
func connectDb() (*sql.DB, error) {
	return sql.Open("sqlite3", DATABASE)
}

// initDb creates the database tables
func initDb() {
	database, err := connectDb()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	schemaSQL, err := os.ReadFile("schema.sql")
	if err != nil {
		log.Fatal(err)
	}

	_, err = database.Exec(string(schemaSQL))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Database initialized.")
}

// getUserID looks up the id for a username
func getUserID(username string) (int, error) {
	var userID int
	err := db.QueryRow("SELECT user_id FROM user WHERE username = ?", username).Scan(&userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

// formatDatetime formats a unix timestamp for display
func formatDatetime(timestamp int64) string {
	t := time.Unix(timestamp, 0).UTC()
	return t.Format("2006-01-02 @ 15:04")
}

// gravatarURL returns the gravatar image URL for the given email
func gravatarURL(email string, size int) string {
	h := md5.Sum([]byte(strings.TrimSpace(strings.ToLower(email))))
	return fmt.Sprintf("http://www.gravatar.com/avatar/%x?d=identicon&s=%d", h, size)
}

// getUserFromSession retrieves the current user from session
func getUserFromSession(c *gin.Context) *User {
	session := sessions.Default(c)
	userID := session.Get("user_id")
	if userID == nil {
		return nil
	}

	var user User
	err := db.QueryRow("SELECT user_id, username, email, pw_hash FROM user WHERE user_id = ?", userID).
		Scan(&user.UserID, &user.Username, &user.Email, &user.PwHash)
	if err != nil {
		return nil
	}
	return &user
}

// scanMessages scans database rows into Message structs
// Expects queries like: SELECT message.*, user.* FROM message, user ...
// which produces columns: message_id, author_id, text, pub_date, flagged, user_id, username, email, pw_hash
func scanMessages(rows *sql.Rows) []Message {
	defer rows.Close()
	var messages []Message

	for rows.Next() {
		var m Message
		var userID int
		var pwHash string
		err := rows.Scan(&m.MessageID, &m.AuthorID, &m.Text, &m.PubDate, &m.Flagged,
			&userID, &m.Username, &m.Email, &pwHash)
		if err != nil {
			log.Printf("scanMessages error: %v", err)
			continue
		}
		messages = append(messages, m)
	}
	return messages
}

// hashPassword generates a bcrypt hash for the password
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// checkPasswordHash compares a password against a hash.
// Supports both bcrypt and werkzeug-style pbkdf2 hashes for backward compatibility
// with existing users created by the Python version.
func checkPasswordHash(password, storedHash string) bool {
	// Try bcrypt first (new Go-created hashes)
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err == nil {
		return true
	}

	// Fallback: check werkzeug-style hash (pbkdf2:sha256:N$salt$hash)
	if strings.HasPrefix(storedHash, "pbkdf2:") {
		return checkWerkzeugHash(password, storedHash)
	}

	return false
}

// checkWerkzeugHash verifies a werkzeug-style password hash
// Format: pbkdf2:sha256:260000$salt$hexhash
func checkWerkzeugHash(password, storedHash string) bool {
	parts := strings.SplitN(storedHash, "$", 3)
	if len(parts) != 3 {
		return false
	}

	method := parts[0]   // e.g. "pbkdf2:sha256:260000"
	salt := parts[1]     // random salt
	expected := parts[2] // hex-encoded derived key

	methodParts := strings.SplitN(method, ":", 3)
	if len(methodParts) < 2 || methodParts[0] != "pbkdf2" {
		return false
	}

	hashName := methodParts[1]
	iterations := 260000
	if len(methodParts) == 3 {
		fmt.Sscanf(methodParts[2], "%d", &iterations)
	}

	var hashFunc func() hash.Hash
	var keyLen int
	switch hashName {
	case "sha256":
		hashFunc = sha256.New
		keyLen = 32
	case "sha512":
		hashFunc = sha512.New
		keyLen = 64
	case "sha1":
		hashFunc = sha1.New
		keyLen = 20
	default:
		return false
	}

	dk := pbkdf2.Key([]byte(password), []byte(salt), iterations, keyLen, hashFunc)
	return hmac.Equal([]byte(hex.EncodeToString(dk)), []byte(expected))
}

func main() {
	// Check for init command
	if len(os.Args) > 1 && os.Args[1] == "init" {
		initDb()
		return
	}

	// Open the global database connection
	var err error
	db, err = connectDb()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	router := gin.Default()

	// Session store
	store := cookie.NewStore([]byte(SecretKey))
	router.Use(sessions.Sessions("session", store))

	// Custom template functions
	funcMap := template.FuncMap{
		"datetimeformat": func(timestamp int64) string {
			return formatDatetime(timestamp)
		},
		"gravatar": func(email string) string {
			return gravatarURL(email, 48)
		},
	}

	// Load templates with custom functions
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))
	router.SetHTMLTemplate(tmpl)

	// Static files
	router.Static("/static", "./static")
	router.StaticFile("/favicon.ico", "./static/favicon.ico")

	// Middleware: set user on every request
	router.Use(func(c *gin.Context) {
		user := getUserFromSession(c)
		if user != nil {
			c.Set("user", user)
		}
		c.Next()
	})

	// Routes
	router.GET("/", timelineHandler)
	router.GET("/public", publicTimelineHandler)
	router.GET("/login", loginGetHandler)
	router.POST("/login", loginPostHandler)
	router.GET("/register", registerGetHandler)
	router.POST("/register", registerPostHandler)
	router.GET("/logout", logoutHandler)
	router.POST("/add_message", addMessageHandler)
	router.GET("/u/:username", userTimelineHandler)
	router.GET("/u/:username/follow", followUserHandler)
	router.GET("/u/:username/unfollow", unfollowUserHandler)

	fmt.Println("Starting MiniTwit on :5001")
	port := os.Getenv("PORT")
	if port == "" {
		port = "5001"
	}
	router.Run(":" + port)
}

// timelineHandler shows the user's timeline or redirects to public
func timelineHandler(c *gin.Context) {
	fmt.Printf("We got a visitor from: %s\n", c.ClientIP())
	user, _ := c.Get("user")
	if user == nil {
		c.Redirect(http.StatusFound, "/public")
		return
	}
	u := user.(*User)
	session := sessions.Default(c)
	flashes := session.Flashes()
	session.Save()

	rows, err := db.Query(`
		SELECT message.*, user.* FROM message, user
		WHERE message.flagged = 0 AND message.author_id = user.user_id AND (
			user.user_id = ? OR
			user.user_id IN (SELECT whom_id FROM follower WHERE who_id = ?))
		ORDER BY message.pub_date DESC LIMIT ?`,
		u.UserID, u.UserID, PER_PAGE)
	if err != nil {
		c.String(http.StatusInternalServerError, "Database error")
		return
	}
	messages := scanMessages(rows)

	c.HTML(http.StatusOK, "timeline.html", gin.H{
		"user":     u,
		"messages": messages,
		"flashes":  flashes,
		"endpoint": "timeline",
		"title":    "My Timeline",
	})
}

// publicTimelineHandler shows all public messages
func publicTimelineHandler(c *gin.Context) {
	user, _ := c.Get("user")
	session := sessions.Default(c)
	flashes := session.Flashes()
	session.Save()

	rows, err := db.Query(`
		SELECT message.*, user.* FROM message, user
		WHERE message.flagged = 0 AND message.author_id = user.user_id
		ORDER BY message.pub_date DESC LIMIT ?`, PER_PAGE)
	if err != nil {
		c.String(http.StatusInternalServerError, "Database error")
		return
	}
	messages := scanMessages(rows)

	var u *User
	if user != nil {
		u = user.(*User)
	}

	c.HTML(http.StatusOK, "timeline.html", gin.H{
		"user":     u,
		"messages": messages,
		"flashes":  flashes,
		"endpoint": "public_timeline",
		"title":    "Public Timeline",
	})
}

// userTimelineHandler shows a specific user's messages
func userTimelineHandler(c *gin.Context) {
	username := c.Param("username")

	var profileUser User
	err := db.QueryRow("SELECT user_id, username, email, pw_hash FROM user WHERE username = ?", username).
		Scan(&profileUser.UserID, &profileUser.Username, &profileUser.Email, &profileUser.PwHash)
	if err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	user, _ := c.Get("user")
	session := sessions.Default(c)
	flashes := session.Flashes()
	session.Save()

	followed := false
	if user != nil {
		u := user.(*User)
		var count int
		err := db.QueryRow("SELECT 1 FROM follower WHERE who_id = ? AND whom_id = ?",
			u.UserID, profileUser.UserID).Scan(&count)
		if err == nil {
			followed = true
		}
	}

	rows, err := db.Query(`
		SELECT message.*, user.* FROM message, user
		WHERE user.user_id = message.author_id AND user.user_id = ?
		ORDER BY message.pub_date DESC LIMIT ?`,
		profileUser.UserID, PER_PAGE)
	if err != nil {
		c.String(http.StatusInternalServerError, "Database error")
		return
	}
	messages := scanMessages(rows)

	var u *User
	if user != nil {
		u = user.(*User)
	}

	c.HTML(http.StatusOK, "timeline.html", gin.H{
		"user":         u,
		"messages":     messages,
		"flashes":      flashes,
		"followed":     followed,
		"profile_user": profileUser,
		"endpoint":     "user_timeline",
		"title":        profileUser.Username + "'s Timeline",
	})
}

// followUserHandler adds the current user as follower
func followUserHandler(c *gin.Context) {
	user, _ := c.Get("user")
	if user == nil {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}
	u := user.(*User)
	username := c.Param("username")

	whomID, err := getUserID(username)
	if err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	_, err = db.Exec("INSERT INTO follower (who_id, whom_id) VALUES (?, ?)", u.UserID, whomID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Database error")
		return
	}

	session := sessions.Default(c)
	session.AddFlash(fmt.Sprintf("You are now following \"%s\"", username))
	session.Save()

	c.Redirect(http.StatusFound, "/u/"+username)
}

// unfollowUserHandler removes the current user as follower
func unfollowUserHandler(c *gin.Context) {
	user, _ := c.Get("user")
	if user == nil {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}
	u := user.(*User)
	username := c.Param("username")

	whomID, err := getUserID(username)
	if err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	_, err = db.Exec("DELETE FROM follower WHERE who_id = ? AND whom_id = ?", u.UserID, whomID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Database error")
		return
	}

	session := sessions.Default(c)
	session.AddFlash(fmt.Sprintf("You are no longer following \"%s\"", username))
	session.Save()

	c.Redirect(http.StatusFound, "/u/"+username)
}

// addMessageHandler records a new message
func addMessageHandler(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")
	if userID == nil {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	text := c.PostForm("text")
	if text != "" {
		_, err := db.Exec("INSERT INTO message (author_id, text, pub_date, flagged) VALUES (?, ?, ?, 0)",
			userID, text, time.Now().Unix())
		if err != nil {
			c.String(http.StatusInternalServerError, "Database error")
			return
		}
		session.AddFlash("Your message was recorded")
		session.Save()
	}

	c.Redirect(http.StatusFound, "/")
}

// loginGetHandler shows the login form
func loginGetHandler(c *gin.Context) {
	user, _ := c.Get("user")
	if user != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	session := sessions.Default(c)
	flashes := session.Flashes()
	session.Save()

	c.HTML(http.StatusOK, "login.html", gin.H{
		"user":    nil,
		"error":   "",
		"flashes": flashes,
		"title":   "Sign In",
	})
}

// loginPostHandler processes the login form
func loginPostHandler(c *gin.Context) {
	user, _ := c.Get("user")
	if user != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	username := c.PostForm("username")
	password := c.PostForm("password")

	var u User
	err := db.QueryRow("SELECT user_id, username, email, pw_hash FROM user WHERE username = ?", username).
		Scan(&u.UserID, &u.Username, &u.Email, &u.PwHash)

	var errorMsg string
	if err != nil {
		errorMsg = "Invalid username"
	} else if !checkPasswordHash(password, u.PwHash) {
		errorMsg = "Invalid password"
	} else {
		session := sessions.Default(c)
		session.AddFlash("You were logged in")
		session.Set("user_id", u.UserID)
		session.Save()
		c.Redirect(http.StatusFound, "/")
		return
	}

	session := sessions.Default(c)
	flashes := session.Flashes()
	session.Save()

	c.HTML(http.StatusOK, "login.html", gin.H{
		"user":     nil,
		"error":    errorMsg,
		"flashes":  flashes,
		"username": username,
		"title":    "Sign In",
	})
}

// registerGetHandler shows the registration form
func registerGetHandler(c *gin.Context) {
	user, _ := c.Get("user")
	if user != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	session := sessions.Default(c)
	flashes := session.Flashes()
	session.Save()

	c.HTML(http.StatusOK, "register.html", gin.H{
		"user":    nil,
		"error":   "",
		"flashes": flashes,
		"title":   "Sign Up",
	})
}

// registerPostHandler processes the registration form
func registerPostHandler(c *gin.Context) {
	user, _ := c.Get("user")
	if user != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	username := c.PostForm("username")
	email := c.PostForm("email")
	password := c.PostForm("password")
	password2 := c.PostForm("password2")

	var errorMsg string
	if username == "" {
		errorMsg = "You have to enter a username"
	} else if email == "" || !strings.Contains(email, "@") {
		errorMsg = "You have to enter a valid email address"
	} else if password == "" {
		errorMsg = "You have to enter a password"
	} else if password != password2 {
		errorMsg = "The two passwords do not match"
	} else {
		_, err := getUserID(username)
		if err == nil {
			errorMsg = "The username is already taken"
		}
	}

	if errorMsg == "" {
		pwHash, err := hashPassword(password)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error hashing password")
			return
		}
		_, err = db.Exec("INSERT INTO user (username, email, pw_hash) VALUES (?, ?, ?)",
			username, email, pwHash)
		if err != nil {
			c.String(http.StatusInternalServerError, "Database error")
			return
		}
		session := sessions.Default(c)
		session.AddFlash("You were successfully registered and can login now")
		session.Save()
		c.Redirect(http.StatusFound, "/login")
		return
	}

	session := sessions.Default(c)
	flashes := session.Flashes()
	session.Save()

	c.HTML(http.StatusOK, "register.html", gin.H{
		"user":     nil,
		"error":    errorMsg,
		"flashes":  flashes,
		"username": username,
		"email":    email,
		"title":    "Sign Up",
	})
}

// logoutHandler logs the user out
func logoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.AddFlash("You were logged out")
	session.Delete("user_id")
	session.Save()
	c.Redirect(http.StatusFound, "/public")
}
