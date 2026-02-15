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
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

const (
	DATABASE   = "/tmp/minitwit.db"
	PER_PAGE   = 30
	DEBUG      = true
	SECRET_KEY = "development key"
)

func main() {
	if _, err := os.Stat(DATABASE); os.IsNotExist(err) {
		initDB()
	}

	db := connectDB()
	defer db.Close()

	r := gin.Default()

	store := cookie.NewStore([]byte(SECRET_KEY))
	r.Use(sessions.Sessions("session", store))

	r.Use(beforeRequest(db))

	r.SetFuncMap(template.FuncMap{
		"datetimeformat": formatDatetime,
		"gravatar":       gravatarURL,
		"url_for": func(endpoint string, args ...string) string {
			routes := map[string]string{
				"timeline":        "/",
				"public_timeline": "/public",
				"login":           "/login",
				"register":        "/register",
				"logout":          "/logout",
			}
			return routes[endpoint]
		},
	})

	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")

	r.GET("/", timeline)
	r.GET("/public", publicTimeline)
	r.POST("/add_message", addMessage)
	r.GET("/login", login)
	r.POST("/login", login)
	r.GET("/register", register)
	r.POST("/register", register)
	r.GET("/logout", logout)
	r.GET("/:username", userTimeline)
	r.GET("/:username/follow", followUser)
	r.GET("/:username/unfollow", unfollowUser)
	r.Run("0.0.0.0:8080")
}

func connectDB() *sql.DB {
	db, err := sql.Open("sqlite3", DATABASE)
	if err != nil {
		panic(err)
	}
	return db
}

func initDB() {
	db := connectDB()
	defer db.Close()

	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(string(schema))
	if err != nil {
		panic(err)
	}
}

func queryDB(db *sql.DB, query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}
	return results, nil
}

func getUserID(db *sql.DB, username string) (int64, error) {
	var userID int64
	err := db.QueryRow("select user_id from user where username = ?", username).Scan(&userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func formatDatetime(timestamp int64) string {
	t := time.Unix(timestamp, 0).UTC()
	return t.Format("2006-01-02 @ 15:04")
}

func gravatarURL(email string, size int) string {
	hash := md5.Sum([]byte(strings.ToLower(strings.TrimSpace(email))))
	return fmt.Sprintf("http://www.gravatar.com/avatar/%x?d=identicon&s=%d", hash, size)
}

func beforeRequest(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("db", db)
		session := sessions.Default(c)
		userID := session.Get("user_id")
		if userID != nil {
			rows, err := queryDB(db, "select * from user where user_id = ?", userID)
			if err == nil && len(rows) > 0 {
				c.Set("user", rows[0])
			}
		} else {
			c.Set("user", nil)
		}
		c.Next()
	}
}
func timeline(c *gin.Context) {
	fmt.Printf("We got a visitor from: %s\n", c.ClientIP())

	user, exists := c.Get("user")
	if !exists || user == nil {
		c.Redirect(http.StatusFound, "/public")
		return
	}

	session := sessions.Default(c)
	userID := session.Get("user_id")

	db := c.MustGet("db").(*sql.DB)
	messages, err := queryDB(db, `
		select message.*, user.* from message, user
		where message.flagged = 0 and message.author_id = user.user_id and (
			user.user_id = ? or
			user.user_id in (select whom_id from follower
							where who_id = ?))
		order by message.pub_date desc limit ?`,
		userID, userID, PER_PAGE)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.HTML(http.StatusOK, "timeline.html", gin.H{
		"messages": messages,
		"user":     user,
		"endpoint": "timeline",
	})
}

func publicTimeline(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)
	messages, err := queryDB(db, `
		select message.*, user.* from message, user
		where message.flagged = 0 and message.author_id = user.user_id
		order by message.pub_date desc limit ?`, PER_PAGE)
	if err != nil {
		fmt.Println("ERROR:", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.HTML(http.StatusOK, "timeline.html", gin.H{
		"messages": messages,
		"endpoint": "public_timeline",
	})
}

func userTimeline(c *gin.Context) {
	username := c.Param("username")
	db := c.MustGet("db").(*sql.DB)

	profileUsers, err := queryDB(db, "select * from user where username = ?", username)
	if err != nil || len(profileUsers) == 0 {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	profileUser := profileUsers[0]

	followed := false
	user, exists := c.Get("user")
	if exists && user != nil {
		session := sessions.Default(c)
		userID := session.Get("user_id")
		rows, err := queryDB(db, `select 1 from follower where
			follower.who_id = ? and follower.whom_id = ?`,
			userID, profileUser["user_id"])
		if err == nil && len(rows) > 0 {
			followed = true
		}
	}

	messages, err := queryDB(db, `
		select message.*, user.* from message, user where
		user.user_id = message.author_id and user.user_id = ?
		order by message.pub_date desc limit ?`,
		profileUser["user_id"], PER_PAGE)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.HTML(http.StatusOK, "timeline.html", gin.H{
		"messages":    messages,
		"followed":    followed,
		"profileUser": profileUser,
		"user":        user,
		"endpoint":    "user_timeline",
	})
}

func followUser(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists || user == nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	username := c.Param("username")
	db := c.MustGet("db").(*sql.DB)

	whomID, err := getUserID(db, username)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	session := sessions.Default(c)
	userID := session.Get("user_id")

	_, err = db.Exec("insert into follower (who_id, whom_id) values (?, ?)", userID, whomID)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	session.AddFlash("You are now following \"" + username + "\"")
	session.Save()

	c.Redirect(http.StatusFound, "/"+username)
}

func unfollowUser(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists || user == nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	username := c.Param("username")
	db := c.MustGet("db").(*sql.DB)

	whomID, err := getUserID(db, username)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	session := sessions.Default(c)
	userID := session.Get("user_id")

	_, err = db.Exec("delete from follower where who_id=? and whom_id=?", userID, whomID)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	session.AddFlash("You are no longer following \"" + username + "\"")
	session.Save()

	c.Redirect(http.StatusFound, "/"+username)
}

// replaces @app.route('/add_message', methods=['POST']) in Python
func addMessage(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")
	if userID == nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	text := c.PostForm("text")
	if text != "" {
		db := c.MustGet("db").(*sql.DB)
		_, err := db.Exec(`insert into message (author_id, text, pub_date, flagged)
			values (?, ?, ?, 0)`, userID, text, time.Now().Unix())
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		session.AddFlash("Your message was recorded")
		session.Save()
	}

	c.Redirect(http.StatusFound, "/")
}

// replaces @app.route('/login', methods=['GET', 'POST']) in Python
func login(c *gin.Context) {
	session := sessions.Default(c)
	user, _ := c.Get("user")
	if user != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	var errorMsg string

	if c.Request.Method == http.MethodPost {
		db := c.MustGet("db").(*sql.DB)
		username := c.PostForm("username")
		password := c.PostForm("password")

		users, err := queryDB(db, "select * from user where username = ?", username)
		if err != nil || len(users) == 0 {
			errorMsg = "Invalid username"
		} else if !checkPasswordHash(password, users[0]["pw_hash"].(string)) {
			errorMsg = "Invalid password"
		} else {
			session.Set("user_id", users[0]["user_id"])
			session.AddFlash("You were logged in")
			session.Save()
			c.Redirect(http.StatusFound, "/")
			return
		}
	}

	c.HTML(http.StatusOK, "login.html", gin.H{
		"error": errorMsg,
	})
}

func register(c *gin.Context) {
	session := sessions.Default(c)
	user, _ := c.Get("user")
	if user != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	var errorMsg string

	if c.Request.Method == http.MethodPost {
		db := c.MustGet("db").(*sql.DB)
		username := c.PostForm("username")
		email := c.PostForm("email")
		password := c.PostForm("password")
		password2 := c.PostForm("password2")

		if username == "" {
			errorMsg = "You have to enter a username"
		} else if email == "" || !strings.Contains(email, "@") {
			errorMsg = "You have to enter a valid email address"
		} else if password == "" {
			errorMsg = "You have to enter a password"
		} else if password != password2 {
			errorMsg = "The two passwords do not match"
		} else {
			_, err := getUserID(db, username)
			if err == nil {
				errorMsg = "The username is already taken"
			} else {
				hash := generatePasswordHash(password)
				_, err = db.Exec(`insert into user (username, email, pw_hash) values (?, ?, ?)`,
					username, email, hash)
				if err != nil {
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}
				session.AddFlash("You were successfully registered and can login now")
				session.Save()
				c.Redirect(http.StatusFound, "/login")
				return
			}
		}
	}

	c.HTML(http.StatusOK, "register.html", gin.H{
		"error": errorMsg,
	})
}
func generatePasswordHash(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
func logout(c *gin.Context) {
	session := sessions.Default(c)
	session.AddFlash("You were logged out")
	session.Delete("user_id")
	session.Save()
	c.Redirect(http.StatusFound, "/public")
}
