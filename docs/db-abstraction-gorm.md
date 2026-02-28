# DB Abstraction Layer — GORM

## The Problem Before

The code talked directly to PostgreSQL using raw SQL strings scattered everywhere:

```go
// minitwit.go
db.QueryRow("SELECT user_id, username, email, pw_hash FROM users WHERE username = $1", username)
db.Exec("INSERT INTO messages (author_id, text, pub_date, flagged) VALUES ($1, $2, $3, 0)", ...)
db.Query("SELECT messages.*, users.* FROM messages, users WHERE ...")
```

This is **tightly coupled to the database** — the Go code knows exactly which table names, column names, and SQL syntax PostgreSQL uses. If we ever switch databases, or rename a column, or change a query, we would have to find and edit raw strings throughout the whole codebase.

---

## What an ORM Does

An ORM (Object-Relational Mapper) lets us work with **Go structs** instead of SQL. We define our data as structs, and the ORM generates the SQL queries. We never write `SELECT` or `INSERT` again.

---

## What Was Changed

### Step 1 — Models (structs) that map to DB tables

```go
type User struct {
    UserID   int    `gorm:"column:user_id;primaryKey;autoIncrement"`
    Username string `gorm:"column:username"`
    Email    string `gorm:"column:email"`
    PWHash   string `gorm:"column:pw_hash"`
}

type Message struct {
    MessageID int    `gorm:"column:message_id;primaryKey;autoIncrement"`
    AuthorID  int    `gorm:"column:author_id"`
    Text      string `gorm:"column:text"`
    PubDate   int64  `gorm:"column:pub_date"`
    Flagged   int    `gorm:"column:flagged"`
    Author    User   `gorm:"foreignKey:AuthorID;references:UserID"` // ← relationship
}

type Follower struct {
    WhoID  int `gorm:"column:who_id"`
    WhomID int `gorm:"column:whom_id"`
}
```

The backtick tags (`gorm:"..."`) tell GORM which column in the database each field corresponds to.
The `Author User` field on `Message` tells GORM there is a relationship — a message belongs to a user.

**No schema changes** — these structs describe the tables that already existed.

---

### Step 2 — Swapped `*sql.DB` for `*gorm.DB`

```go
// Before
var db *sql.DB
db, err = sql.Open("postgres", connStr)

// After
var db *gorm.DB
db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
```

GORM wraps the database connection and adds its query-building layer on top.

---

### Step 3 — Replaced every raw SQL call with GORM methods

| Before (raw SQL) | After (GORM) |
|---|---|
| `db.QueryRow("SELECT ... WHERE username = $1", u)` | `db.Where("username = ?", u).First(&user)` |
| `db.Exec("INSERT INTO messages ...")` | `db.Create(&Message{...})` |
| `db.Exec("INSERT INTO follower ...")` | `db.Create(&Follower{...})` |
| `db.Exec("DELETE FROM follower WHERE ...")` | `db.Where("who_id = ? AND whom_id = ?", ...).Delete(&Follower{})` |
| `db.Query("SELECT messages.*, users.* ...")` | `db.Preload("Author").Where("flagged = 0").Find(&messages)` |

The `Preload("Author")` call replaces the big `JOIN` queries. It tells GORM: _when you fetch messages, also fetch the associated `User` for each message and put it in the `Author` field._ The old code needed a flat `TimelineMessage` struct with duplicated user fields because joins flatten everything into one row. With GORM we get a proper nested struct: `message.Author.Username`, `message.Author.Email`.

---

## Why the `Follower` Type Was New

The old code never needed a Go struct for the `follower` table because it only ever used raw SQL on it. GORM needs a struct to know what table to target — `db.Create(&Follower{})` and `db.Delete(&Follower{})` use the struct's `TableName()` to figure out which table to write to.

