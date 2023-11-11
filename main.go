package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type Task struct {
	ID          int
	Description string
	Completed   bool
	Priority    int
	Deadline    *time.Time
	Category    string
}

// Add a new struct for rendering the update task page
type UpdatePageData struct {
	Task Task
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "tasks.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		description TEXT,
		completed BOOLEAN,
		priority INTEGER,
		deadline TEXT,
		category TEXT
	);
`)

	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	initDB()
	defer db.Close()

	http.HandleFunc("/", listTasks)
	http.HandleFunc("/add", addTask)
	http.HandleFunc("/delete/", deleteTask)
	http.HandleFunc("/update/", renderUpdateTaskPage)
	http.HandleFunc("/update/submit/", updateTask)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	fmt.Println("Server started 🏁")
	fmt.Println("Listening at 👉 http://localhost:8080")
	fmt.Println("Ctrl+c to Close the Server ❌")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	if !(path == "/" || path == "/add" || path == "/delete/" || path == "/update/" || path == "/update/submit/") {
		http.Error(w, "😒😒😒 404 Page Not Found 😒😒😒 ", http.StatusNotFound)
		return true
	}
	return false
}

func listTasks(w http.ResponseWriter, r *http.Request) {
	if NotFoundHandler(w, r) {
		return
	}
	rows, err := db.Query(`SELECT id, description, completed, priority, deadline, category FROM tasks;`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var deadlineStr sql.NullString
		err := rows.Scan(&task.ID, &task.Description, &task.Completed, &task.Priority, &deadlineStr, &task.Category)
		if err != nil {
			log.Fatal(err)
		}

		// Convert the deadline string to time.Time if not NULL
		if deadlineStr.Valid {
			deadlineTime, err := time.Parse("2006-01-02 15:04:05-07:00", deadlineStr.String)
			if err != nil {
				log.Fatal(err)
			}
			task.Deadline = &deadlineTime
		}

		tasks = append(tasks, task)
	}

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatal(err)
	}

	tmpl.Execute(w, struct{ Tasks []Task }{Tasks: tasks})
}

func addTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	description := r.FormValue("task")
	if description == " " {
		http.Error(w, "😒😒 Task description cannot be empty", http.StatusBadRequest)
		return
	}

	priority, err := strconv.Atoi(r.FormValue("priority"))
	if err != nil {
		http.Error(w, "Invalid priority value", http.StatusBadRequest)
		return
	}

	deadlineStr := r.FormValue("deadline")
	var deadline *time.Time
	if deadlineStr != "" {
		deadlineTime, err := time.Parse("2006-01-02T15:04", deadlineStr)
		if err != nil {
			http.Error(w, "Invalid deadline format", http.StatusBadRequest)
			return
		}
		deadline = &deadlineTime
	}

	category := r.FormValue("category")

	_, err = db.Exec(`
		INSERT INTO tasks (description, completed, priority, deadline, category)
		VALUES (?, ?, ?, ?, ?);
	`, description, false, priority, deadline, category)
	if err != nil {
		log.Fatal(err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Handle function for deleting a task
func deleteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	taskID := r.URL.Path[len("/delete/"):]
	_, err := db.Exec(`DELETE FROM tasks WHERE id=?;`, taskID)
	if err != nil {
		log.Fatal(err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Handle function for rendering the update task page
func renderUpdateTaskPage(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Path[len("/update/"):]
	var task Task
	err := db.QueryRow(`SELECT id, description, completed FROM tasks WHERE id=?;`, taskID).Scan(&task.ID, &task.Description, &task.Completed)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	tmpl, err := template.ParseFiles("templates/update.html")
	if err != nil {
		log.Fatal(err)
	}

	tmpl.Execute(w, UpdatePageData{Task: task})
}

// Handle function for updating a task
func updateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	taskID := r.FormValue("id") // Retrieve the task ID from the form
	description := r.FormValue("task")

	_, err := db.Exec(`UPDATE tasks SET description=? WHERE id=?;`, description, taskID)
	if err != nil {
		log.Fatal(err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
