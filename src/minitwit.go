package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	UserID   int    `json:"user_id"`
	UserName string `json:"username"`
	Email    string `json:"email"`
	PW_Hash  string `json:"pw_hash"`
}

type TimelineMessage struct {
	MessageID int    `json:"message_id"`
	AutherID  int    `json:"auther_id"`
	Text      string `json:"text"`
	PubDate   int64  `json:"pub_date"`
	Flagged   int    `json:"flagged"`
	UserID    int    `json:"user_id"`
	UserName  string `json:"username"`
	Email     string `json:"email"`
	PW_Hash   string `json:"pw_hash"`
}

type Gravatar struct {
	Scheme  string
	Host    string
	Hash    string
	Default string
	Rating  string
	Size    int
}

const (
	DATABASE   = "./tmp/minitwit.db"
	PER_PAGE   = 30
	DEBUG      = true
	SECRET_KEY = "development key"

	// Gravatar scheme and host
	gravatarScheme = "https"
	gravatarHost   = "Gravatar.com"
)

var db *sql.DB

func main() {
	r := gin.Default()
	setupApp(r)
	initDB()
	db = connectDB()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.Run(":8080") // http://localhost:8080
}

func setupApp(c *gin.Engine) {
	store := cookie.NewStore([]byte(SECRET_KEY))
	c.Use(sessions.Sessions("mysession", store))

	c.Use(beforeRequest)

	funcMap := template.FuncMap{
		"gravatar_url":    gravatar_url,
		"format_datetime": format_datetime,
	}

	c.SetFuncMap(funcMap)
	c.LoadHTMLGlob("./templates/*")
	c.Static("/static", "./static")

	// Routes
	c.GET("/", timeline)
	c.GET("/public", publicTimeline)
	c.GET("/:username", userTimeline)
	/*
		c.GET("/logout", logout)
		c.GET("/:username/follow", follow_user)
		c.GET("/:username/unfollow", unfollow_user)

		c.POST("/add_message", add_message)
		c.GET("/login", loginGet)
		c.POST("/login", loginPost)
		c.GET("/register", registerGet)
		c.POST("/register", registerPost)
	*/
}

func connectDB() *sql.DB {
	//"""Creates the database tables."""
	var err error
	db, err = sql.Open("sqlite3", "./tmp/minitwit.db")
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func initDB() {
	schema, err := os.ReadFile("./tmp/schema.sql")
	if err != nil {
		log.Fatal(err)
	}
	db, err = sql.Open("sqlite3", DATABASE)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(string(schema))
	if err != nil {
		log.Fatalf("Failed to execute schema: %v", err)
	}
	fmt.Println("Database created and schema applied successfully.")
}

func getUserID(username string) (int, error) {
	IDs, err := queryUsers("select user_id from user where username = ?", username)
	if err != nil {
		return 0, err
	}
	return IDs[0].UserID, nil
}

func format_datetime(timestamp int64) string {
	return time.Unix(timestamp, 0).Format("2006-01-02 @ 12006-01-02 @ 15:04")
}

func gravatar_url(email string, size int) string {
	return createGravatarfromEmail(email, size).gravatarURL()
}

func createGravatarfromEmail(email string, size int) Gravatar {
	// why sha256 and not md5? look here: https://docs.gravatar.com/rest/api-data-specifications/
	email = strings.ToLower(strings.TrimSpace(email))
	hasher := sha256.Sum256([]byte(email))
	hash := hex.EncodeToString(hasher[:])

	g := NewGravatar()
	g.Hash = hash
	g.Size = size
	return g
}

func NewGravatar() Gravatar {
	return Gravatar{
		Scheme: gravatarScheme,
		Host:   gravatarHost,
	}
}

func (g Gravatar) gravatarURL() string {
	path := "/avatar/" + g.Hash

	v := url.Values{}
	if g.Size > 0 {
		v.Add("s", strconv.Itoa(g.Size))
	}

	if g.Rating != "" {
		v.Add("r", g.Rating)
	}

	if g.Default != "" {
		v.Add("d", g.Default)
	}

	url := url.URL{
		Scheme:   g.Scheme,
		Host:     g.Host,
		Path:     path,
		RawQuery: v.Encode(),
	}

	return url.String()
}

func beforeRequest(c *gin.Context) {
	userID := sessions.Default(c).Get("user_id")
	if userID != nil {
		users, err := queryUsers("select * from user where user_id = ?", userID)
		if err != nil {
			log.Println("Error querying user:", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if len(users) > 0 {
			c.Set("user", users[0])
		}
	}
	c.Next()
}

func timeline(c *gin.Context) {
	print("We got a visitor from: %s", c.Request.RemoteAddr)
	val, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusFound, "/public")
		return
	}
	user := val.(User)

	query := `select message.*, user.* from message, user
        where message.flagged = 0 and message.author_id = user.user_id and (
            user.user_id = ? or
            user.user_id in (select whom_id from follower where who_id = ?))
        order by message.pub_date desc limit ?`

	messages, err := queryTimeline(query, user.UserID, user.UserID, PER_PAGE)
	if err != nil {
		log.Println("Error querying timeline:", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	render(c, http.StatusOK, "timeline.html", gin.H{
		"messages": messages,
		"endpoint": "timeline",
	})
}

func publicTimeline(c *gin.Context) {
	query := `select message.*, user.* from message, user
        where message.flagged = 0 and message.author_id = user.user_id
        order by message.pub_date desc limit ?`
	messages, err := queryTimeline(query, PER_PAGE)
	if err != nil {
		log.Println("Error querying timeline:", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	render(c, http.StatusOK, "timeline.html", gin.H{
		"messages": messages,
		"endpoint": "public_timeline",
	})
}

func userTimeline(c *gin.Context) {
	username := c.Param("username")
	query := `select * from user where username = ?`

	users, err := queryUsers(query, username)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if len(users) == 0 {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	profileUser := users[0]

	followed := false
	if val, exists := c.Get("user"); exists {
		currentUser := val.(User)

		var count int
		query = `select 1 from follower where
				follower.who_id = ? and follower.whom_id = ?`
		db.QueryRow(query, currentUser.UserID, profileUser.UserID).Scan(&count)
		followed = count > 0
	}

	timelineQuery := `select message.*, user.* from message, user where
            user.user_id = message.author_id and user.user_id = ?
            order by message.pub_date desc limit ?`
	messages, err := queryTimeline(timelineQuery, profileUser.UserID, PER_PAGE)
	if err != nil {
		log.Println("Error querying timeline:", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	render(c, http.StatusOK, "timeline.html", gin.H{
		"messages":     messages,
		"profile_user": profileUser,
		"followed":     followed,
		"endpoint":     "user_timeline",
	})
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

// =============== Database query helper functions START ================
// NOTE: These could be implemented as a more generic db query function

func queryUsers(query string, args ...any) ([]User, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.UserID, &user.UserName, &user.Email, &user.PW_Hash)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func queryTimeline(query string, args ...any) ([]TimelineMessage, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var timeline []TimelineMessage
	for rows.Next() {
		var msg TimelineMessage
		err := rows.Scan(&msg.MessageID, &msg.AutherID, &msg.Text, &msg.PubDate, &msg.Flagged, &msg.UserID, &msg.UserName, &msg.Email, &msg.PW_Hash)
		if err != nil {
			return nil, err
		}
		timeline = append(timeline, msg)
	}
	return timeline, nil
}

// ==================== Databse helper functions END ===============
