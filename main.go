package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	_ "github.com/mattn/go-sqlite3"
)

type Task struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type AIResponse struct {
	Content string `json:"content"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

var db *sql.DB

func main() {
	// Инициализация SQLite
	var err error
	db, err = sql.Open("sqlite3", "./tasks.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Create table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tasks (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT,
        description TEXT
    )`)
	if err != nil {
		panic(err)
	}

	// Routes
	r := gin.Default()

	// Add CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/tasks", getTasks)
	r.GET("/tasks/:id", getTask)
	r.POST("/tasks", createTask)              // Anthropic API, if not provided API key task created with empty description
	r.POST("/tasks/ollama", createTaskOllama) // Ollama
	r.POST("/tasks/gemini", createTaskGemini) // Gemini
	r.PUT("/tasks/:id", updateTask)
	r.DELETE("/tasks/:id", deleteTask)
	r.Run(":8080")
}

func getTasks(c *gin.Context) {
	rows, err := db.Query("SELECT id, title, description FROM tasks")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	tasks := []Task{}
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		tasks = append(tasks, t)
	}
	c.JSON(http.StatusOK, tasks)
}

func getTask(c *gin.Context) {
	id := c.Param("id")
	var t Task
	err := db.QueryRow("SELECT id, title, description FROM tasks WHERE id = ?", id).Scan(&t.ID, &t.Title, &t.Description)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}
	c.JSON(http.StatusOK, t)
}

func createTask(c *gin.Context) {
	var t Task
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Calling AI to generate a description
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer YOUR_ANTHROPIC_API_KEY").
		SetBody(map[string]string{
			"prompt": fmt.Sprintf("Generate a short description for a task titled '%s'", t.Title),
			"model":  "claude-3-7-sonnet",
		}).
		SetResult(&AIResponse{}).
		Post("https://api.anthropic.com/v1/complete")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to call AI"})
		return
	}

	t.Description = resp.Result().(*AIResponse).Content

	// Save task to database
	result, err := db.Exec("INSERT INTO tasks (title, description) VALUES (?, ?)", t.Title, t.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	t.ID = int(id)
	c.JSON(http.StatusCreated, t)
}

func updateTask(c *gin.Context) {
	id := c.Param("id")
	var t Task
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec("UPDATE tasks SET title = ?, description = ? WHERE id = ?", t.Title, t.Description, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Task updated"})
}

func deleteTask(c *gin.Context) {
	id := c.Param("id")
	_, err := db.Exec("DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Task deleted"})
}

func createTaskOllama(c *gin.Context) {
	var t Task
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Call Ollama for generating description
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"model":  "gemma3:1b", // or whatever model you have available
			"prompt": fmt.Sprintf("Generate a short description for a task titled '%s'. Keep it under 100 characters.", t.Title),
		}).
		Post("http://localhost:11434/api/generate")

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to call Ollama: %s", err.Error())})
		return
	}

	// Handle streaming response - each line is a separate JSON object
	responseBody := string(resp.Body())

	// For debugging
	fmt.Println("Response from Ollama:", responseBody)

	// Process the streaming response by collecting all pieces
	var description strings.Builder

	// Split the response by newlines
	lines := strings.Split(responseBody, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse each line as a JSON object
		var partialResponse struct {
			Response string `json:"response"`
			Done     bool   `json:"done"`
		}

		if err := json.Unmarshal([]byte(line), &partialResponse); err != nil {
			continue // Skip lines that can't be parsed
		}

		// Add this part to our accumulated description
		description.WriteString(partialResponse.Response)
	}

	// Use the accumulated description
	t.Description = description.String()

	// If the description is too long, trim it
	if len(t.Description) > 500 {
		t.Description = t.Description[:500] + "..."
	}

	result, err := db.Exec("INSERT INTO tasks (title, description) VALUES (?, ?)", t.Title, t.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	t.ID = int(id)
	c.JSON(http.StatusCreated, t)
}

func createTaskGemini(c *gin.Context) {
	var t Task
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get API-key from env
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GEMINI_API_KEY not set"})
		return
	}

	// call Gemini for genrate description
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetQueryParam("key", apiKey).
		SetBody(map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]string{
						{"text": fmt.Sprintf("Generate a short description for a task titled '%s'. Keep it concise and clear.", t.Title)},
					},
				},
			},
			"generationConfig": map[string]interface{}{
				"temperature":     0.7,
				"maxOutputTokens": 100,
			},
		}).
		SetResult(&GeminiResponse{}).
		Post("https://generativelanguage.googleapis.com/v1/models/gemini-2.0-flash:generateContent")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to call Gemini: %v"})
		return
	}

	// Print response for debugging
	// fmt.Printf("Gemini API response: %+v\n", resp)

	geminiResp := resp.Result().(*GeminiResponse)
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Empty response from Gemini"})
		return
	}
	t.Description = geminiResp.Candidates[0].Content.Parts[0].Text

	// Save to db
	result, err := db.Exec("INSERT INTO tasks (title, description) VALUES (?, ?)", t.Title, t.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	t.ID = int(id)
	c.JSON(http.StatusCreated, t)
}
