package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

var Latest int = -1

type SimMessage struct {
	Content string `json:"content"`
	PubDate int64  `json:"pub_date"`
	User    string `json:"user"`
}

type SimRegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Pwd      string `json:"pwd"`
}

type SimPostMessageRequest struct {
	Content string `json:"content"`
}

type SimFollowAction struct {
	Follow   string `json:"follow,omitempty"`
	Unfollow string `json:"unfollow,omitempty"`
}

func updateLatest(c *gin.Context) {
	latestStr := c.Query("latest")
	if latestStr != "" {
		parsedLatest, err := strconv.Atoi(latestStr)
		if err == nil {
			Latest = parsedLatest
		}
	}
}

// GET /latest
func get_latest_value(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"latest": Latest})
}

// POST /register
func post_register(c *gin.Context) {
	updateLatest(c)
	var req SimRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": 400, "error_msg": err.Error()})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Pwd), bcrypt.DefaultCost)
	_, err := db.Exec("INSERT INTO user (username, email, pw_hash) VALUES (?, ?, ?)",
		req.Username, req.Email, string(hashedPassword))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": 400, "error_msg": "Username already taken"})
		return
	}

	c.Status(http.StatusNoContent)
}

// GET /msgs
func get_messages(c *gin.Context) {
	updateLatest(c)
	numMsgsStr := c.DefaultQuery("no", "20")
	numMsgs, _ := strconv.Atoi(numMsgsStr)

	query := `
		SELECT message.text, message.pub_date, user.username 
		FROM message, user 
		WHERE message.flagged = 0 AND message.author_id = user.user_id 
		ORDER BY message.pub_date DESC LIMIT ?`

	rows, err := db.Query(query, numMsgs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var messages []SimMessage
	for rows.Next() {
		var m SimMessage
		rows.Scan(&m.Content, &m.PubDate, &m.User)
		messages = append(messages, m)
	}
	c.JSON(http.StatusOK, messages)
}

// GET /msgs/:username
func get_messages_per_user(c *gin.Context) {
	updateLatest(c)
	username := c.Param("username")
	numMsgsStr := c.DefaultQuery("no", "20")
	numMsgs, _ := strconv.Atoi(numMsgsStr)

	userID, _ := get_user_id(username)
	if userID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": 404, "error_msg": "User not found"})
		return
	}

	query := `
		SELECT message.text, message.pub_date, user.username 
		FROM message, user 
		WHERE message.flagged = 0 AND user.user_id = message.author_id AND user.user_id = ?
		ORDER BY message.pub_date DESC LIMIT ?`

	rows, _ := db.Query(query, userID, numMsgs)
	defer rows.Close()

	messages := []SimMessage{}
	for rows.Next() {
		var m SimMessage
		rows.Scan(&m.Content, &m.PubDate, &m.User)
		messages = append(messages, m)
	}
	c.JSON(http.StatusOK, messages)
}

// POST /msgs/:username
func post_messages_per_user(c *gin.Context) {
	updateLatest(c)
	username := c.Param("username")
	userID, _ := get_user_id(username)

	var req SimPostMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": 400, "error_msg": err.Error()})
		return
	}

	_, err := db.Exec("INSERT INTO message (author_id, text, pub_date, flagged) VALUES (?, ?, ?, 0)",
		userID, req.Content, time.Now().Unix())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": 500, "error_msg": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// GET /fllws/:username
func get_follow(c *gin.Context) {
	updateLatest(c)
	username := c.Param("username")
	userID, _ := get_user_id(username)

	numMsgsStr := c.DefaultQuery("no", "20")
	numMsgs, _ := strconv.Atoi(numMsgsStr)

	query := `
		SELECT user.username FROM user
		INNER JOIN follower ON follower.whom_id = user.user_id
		WHERE follower.who_id = ? LIMIT ?`

	rows, _ := db.Query(query, userID, numMsgs)
	defer rows.Close()

	follows := []string{}
	for rows.Next() {
		var name string
		rows.Scan(&name)
		follows = append(follows, name)
	}
	c.JSON(http.StatusOK, gin.H{"follows": follows})
}

// POST /fllws/:username
func post_follow(c *gin.Context) {
	updateLatest(c)
	username := c.Param("username")
	userID, _ := get_user_id(username)

	var action SimFollowAction
	if err := c.ShouldBindJSON(&action); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": 400, "error_msg": "Invalid JSON"})
		return
	}

	if action.Follow != "" {
		whomID, _ := get_user_id(action.Follow)
		db.Exec("INSERT INTO follower (who_id, whom_id) VALUES (?, ?)", userID, whomID)
	} else if action.Unfollow != "" {
		whomID, _ := get_user_id(action.Unfollow)
		db.Exec("DELETE FROM follower WHERE who_id = ? AND whom_id = ?", userID, whomID)
	}

	c.Status(http.StatusNoContent)
}
