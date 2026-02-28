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
	newUser := User{Username: req.Username, Email: req.Email, PWHash: string(hashedPassword)}
	if err := db.Create(&newUser).Error; err != nil {
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

	var msgs []Message
	db.Preload("Author").Where("flagged = 0").
		Order("pub_date DESC").Limit(numMsgs).Find(&msgs)

	var messages []SimMessage
	for _, m := range msgs {
		messages = append(messages, SimMessage{
			Content: m.Text,
			PubDate: m.PubDate,
			User:    m.Author.Username,
		})
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

	var msgs []Message
	db.Preload("Author").Where("flagged = 0 AND author_id = ?", userID).
		Order("pub_date DESC").Limit(numMsgs).Find(&msgs)

	messages := []SimMessage{}
	for _, m := range msgs {
		messages = append(messages, SimMessage{
			Content: m.Text,
			PubDate: m.PubDate,
			User:    m.Author.Username,
		})
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

	msg := Message{AuthorID: userID, Text: req.Content, PubDate: time.Now().Unix(), Flagged: 0}
	if err := db.Create(&msg).Error; err != nil {
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

	var followers []Follower
	db.Where("who_id = ?", userID).Limit(numMsgs).Find(&followers)

	follows := []string{}
	for _, f := range followers {
		var u User
		if db.First(&u, f.WhomID).Error == nil {
			follows = append(follows, u.Username)
		}
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
		db.Create(&Follower{WhoID: userID, WhomID: whomID})
	} else if action.Unfollow != "" {
		whomID, _ := get_user_id(action.Unfollow)
		db.Where("who_id = ? AND whom_id = ?", userID, whomID).Delete(&Follower{})
	}

	c.Status(http.StatusNoContent)
}
