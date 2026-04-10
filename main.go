package main

import (
	"NoobOJ/database"
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

var store = sessions.NewCookieStore([]byte("super-secret-key"))

func home(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/home.html",
		"templates/footer.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session, _ := store.Get(r, "session")
	username, ok := session.Values["username"].(string)
	var user_type string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&user_type)
	logout := "Logout"
	if !ok || session.Values["username"] == nil {
		username = "Login"
		logout = "Register"
	}
	var admin bool = false
	if user_type == "admin" {
		admin = true
	}
	type PageData struct {
		Title     string
		Pusername string
		Logout    string
		Admin     bool
	}
	data := PageData{
		Title:     "NoobOJ",
		Pusername: username,
		Logout:    logout,
		Admin:     admin,
	}
	t.ExecuteTemplate(w, "index.html", data)
}

func problemsetHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var user_type string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&user_type)
	var admin bool = false
	if user_type == "admin" {
		admin = true
	}
	rows, err := database.DB.Query("SELECT id, title FROM problems where visibility=1")
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	defer rows.Close()

	type Problem struct {
		ID     int
		Title  string
		Solved int
		Tags   []string
		Rating string
		Tried  int
	}
	var list []Problem
	for rows.Next() {
		var p Problem
		rows.Scan(&p.ID, &p.Title)
		database.DB.QueryRow("SELECT COUNT(DISTINCT username) FROM submissions WHERE problem_id=? AND verdict='Accepted'", p.ID).Scan(&p.Solved)
		database.DB.QueryRow("SELECT rating FROM ratings WHERE problem_id=? LIMIT 1", p.ID).Scan(&p.Rating)
		tags, err := database.DB.Query("SELECT name FROM tags,problem_tags WHERE problem_id=? AND tags.id=problem_tags.tag_id", p.ID)
		if err != nil {
			fmt.Fprint(w, "Database error: "+err.Error())
			return
		}

		for tags.Next() {
			temp := ""
			tags.Scan(&temp)
			p.Tags = append(p.Tags, temp)
		}
		tags.Close()
		p.Tried = 0
		var cnt int
		err = database.DB.QueryRow("SELECT COUNT(*) FROM submissions WHERE problem_id=? AND username=?", p.ID, username).Scan(&cnt)
		if err == nil && cnt > 0 {
			p.Tried = 1
		}
		err = database.DB.QueryRow("SELECT COUNT(*) FROM submissions WHERE problem_id=? AND username=? AND verdict='Accepted'", p.ID, username).Scan(&cnt)
		if err == nil && cnt > 0 {
			p.Tried = 2
		}
		list = append(list, p)
	}
	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/problemset.html",
		"templates/footer.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type PageData struct {
		Title     string
		Problems  []Problem
		Logout    string
		Pusername string
		Admin     bool
	}
	data := PageData{
		Title:     "Problemset - NoobOJ",
		Problems:  list,
		Logout:    "Logout",
		Pusername: username,
		Admin:     admin,
	}
	t.ExecuteTemplate(w, "index.html", data)
}
func leaderboardHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var user_type string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&user_type)
	var admin bool = false
	if user_type == "admin" {
		admin = true
	}
	rows, err := database.DB.Query(`SELECT users.username,name,count(DISTINCT submissions.problem_id) AS cnt 
									FROM users,submissions,problems 
									WHERE users.username=submissions.username AND submissions.problem_id = problems.id AND verdict='Accepted' and visibility=1 
									GROUP BY users.username,name 
									ORDER BY cnt DESC`)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	defer rows.Close()
	type Leaderboard struct {
		Username string
		Name     string
		Count    int
	}
	var list []Leaderboard
	for rows.Next() {
		var p Leaderboard
		if err := rows.Scan(&p.Username, &p.Name, &p.Count); err != nil {
			continue
		}
		list = append(list, p)
	}
	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/leaderboard.html",
		"templates/footer.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type PageData struct {
		Title     string
		Items     []Leaderboard
		Pusername string
		Logout    string
		Admin     bool
	}
	data := PageData{
		Title:     "Leaderboard - NoobOJ",
		Items:     list,
		Pusername: username,
		Logout:    "Logout",
		Admin:     admin,
	}
	t.ExecuteTemplate(w, "index.html", data)
}

func newProblemHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	switch r.Method {
	case "GET":
		t, err := template.ParseFiles(
			"templates/index.html",
			"templates/new_problem.html",
			"templates/footer.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		type PageData struct {
			Title     string
			Pusername string
			Logout    string
			Admin     bool
		}
		data := PageData{
			Title:     "Create Problem - NoobOJ",
			Pusername: username,
			Logout:    "Logout",
			Admin:     true,
		}
		t.ExecuteTemplate(w, "index.html", data)
		return
	case "POST":
		name := r.FormValue("title")
		res, err := database.DB.Exec("INSERT INTO problems(title,author) values(?,?)", name, username)
		if err != nil {
			fmt.Fprint(w, "Error inserting Problem : "+err.Error())
			return
		}
		problemID64, _ := res.LastInsertId()
		problemID := int(problemID64)

		http.Redirect(w, r, "/edit/problem?id="+strconv.Itoa(int(problemID)), http.StatusSeeOther)

	default:
		http.Redirect(w, r, "/administration?tab=problem", http.StatusSeeOther)
	}
}

func newContestHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		t, err := template.ParseFiles(
			"templates/index.html",
			"templates/new_contest.html",
			"templates/footer.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		session, _ := store.Get(r, "session")
		username, _ := session.Values["username"].(string)
		type PageData struct {
			Title     string
			Pusername string
			Logout    string
			Admin     bool
		}
		data := PageData{
			Title:     "Create Contest - NoobOJ",
			Pusername: username,
			Logout:    "Logout",
			Admin:     true,
		}
		t.ExecuteTemplate(w, "index.html", data)
		return
	case "POST":
		name := r.FormValue("title")
		startStr := r.FormValue("start")
		endStr := r.FormValue("end")
		tzStr := r.FormValue("timezone")

		loc, err := time.LoadLocation(tzStr)
		if err != nil {
			loc = time.UTC
		}

		layout := "2006-01-02T15:04"

		startTime, err := time.ParseInLocation(layout, startStr, loc)
		if err != nil {
			fmt.Fprint(w, "Invalid start time")
			return
		}

		endTime, err := time.ParseInLocation(layout, endStr, loc)
		if err != nil {
			fmt.Fprint(w, "Invalid end time")
			return
		}

		startTime = startTime.UTC()
		endTime = endTime.UTC()
		// Start a Transaction
		tx, err := database.DB.Begin()
		if err != nil {
			http.Error(w, "Transaction error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()
		session, _ := store.Get(r, "session")
		username, _ := session.Values["username"].(string)
		res, err := tx.Exec(
			"INSERT INTO contests(title, start_time, end_time) VALUES (?, ?, ?)",
			name,
			startTime,
			endTime,
		)
		if err != nil {
			fmt.Fprint(w, "Error inserting contest: "+err.Error())
			return
		}
		contestID, err := res.LastInsertId()
		if err != nil {
			fmt.Fprint(w, "Error getting last ID: "+err.Error())
			return
		}
		res, err = tx.Exec("INSERT INTO authors(contest_id,author,role) VALUES(?,?,?)", contestID, username, "owner")
		if err != nil {
			fmt.Fprint(w, "Error inserting owner: "+err.Error())
			return
		}

		err = tx.Commit()
		if err != nil {
			fmt.Fprint(w, "Commit error: "+err.Error())
			return
		}

		http.Redirect(w, r, "/edit/contest?id="+strconv.Itoa(int(contestID)), http.StatusSeeOther)

	default:
		http.Redirect(w, r, "/administration?tab=contest", http.StatusSeeOther)
	}
}

func viewProblemHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var admin bool = false
	var user_type string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&user_type)
	if user_type == "admin" {
		admin = true
	}
	id := r.URL.Query().Get("id")
	row := database.DB.QueryRow("SELECT title, statement,input,output,constraints,author,time_limit,memory_limit FROM problems WHERE id = ? and visibility=1", id)
	var title, statement, input, output, constraints, author, time, memory string
	err := row.Scan(&title, &statement, &input, &output, &constraints, &author, &time, &memory)
	if err != nil {
		fmt.Fprint(w, "Problem not found")
		return
	}
	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/problem.html",
		"templates/footer.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type Testcases struct {
		Input  string
		Output string
	}
	var list []Testcases
	rows, err := database.DB.Query("SELECT input,output FROM test_cases WHERE problem_id=? AND type='sample'", id)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		var p Testcases
		if err := rows.Scan(&p.Input, &p.Output); err != nil {
			continue
		}
		list = append(list, p)
	}
	rows, err = database.DB.Query("SELECT tag FROM tags WHERE problem_id=?", id)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	defer rows.Close()
	var tags []string
	var rating string
	database.DB.QueryRow("SELECT rating FROM ratings WHERE problem_id=? LIMIT 1", id).Scan(&rating)
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			continue
		}
		tags = append(tags, p)
	}
	tags = append(tags, "*"+rating)

	rows, err = database.DB.Query("SELECT id,submitted_at,verdict FROM submissions WHERE problem_id=? AND username=? ORDER BY id DESC", id, username)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	defer rows.Close()

	type Submissions struct {
		Submission string
		Time       string
		Verdict    string
		Accepted   string
	}
	var subs []Submissions
	for rows.Next() {
		var p Submissions
		if err := rows.Scan(&p.Submission, &p.Time, &p.Verdict); err != nil {
			continue
		}
		p.Accepted = "Accepted"
		subs = append(subs, p)
	}

	type PageData struct {
		Title       string
		Ptitle      string
		Statement   string
		Input       string
		Output      string
		Constraints string
		Time        string
		Memory      string
		Pusername   string
		Logout      string
		Items       []Testcases
		Tags        []string
		Subs        []Submissions
		Author      string
		ID          string
		Admin       bool
		One         string
	}
	data := PageData{
		Title:       "Problem - " + id + " - NoobOJ",
		Ptitle:      title,
		Statement:   statement,
		Input:       input,
		Output:      output,
		Constraints: constraints,
		Time:        time,
		Memory:      memory,
		Pusername:   username,
		Logout:      "Logout",
		Items:       list,
		Tags:        tags,
		Subs:        subs,
		Author:      author,
		ID:          id,
		Admin:       admin,
		One:         "1",
	}
	t.ExecuteTemplate(w, "index.html", data)
}
func viewSubmissionHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var admin bool = false
	var user_type string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&user_type)
	if user_type == "admin" {
		admin = true
	}
	id := r.URL.Query().Get("id")
	row := database.DB.QueryRow("SELECT problem_id, username, code, submitted_at, verdict FROM submissions WHERE id = ?", id)
	var problem_id, user, code, submitted_at, verdict string
	err := row.Scan(&problem_id, &user, &code, &submitted_at, &verdict)
	if err != nil {
		fmt.Fprint(w, "Submission not found")
		return
	}
	var ptitle string
	err = database.DB.QueryRow("SELECT title FROM problems WHERE id=?", problem_id).Scan(&ptitle)
	if err != nil {
		fmt.Fprint(w, "Problem title not found")
		return
	}
	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/submission.html",
		"templates/footer.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type PageData struct {
		Title     string // page title
		Ptitle    string
		Pusername string // session
		Logout    string
		ID        string // submission id
		Username  string // submitted by
		PID       string // problem id
		Verdict   string
		Time      string
		Code      string
		Accepted  string
		Admin     bool
	}

	data := PageData{
		Title:     "Submission #" + id + " - Nooboj",
		Ptitle:    ptitle,
		Pusername: username,
		Logout:    "Logout",
		ID:        id,
		Username:  user,
		PID:       problem_id,
		Verdict:   verdict,
		Time:      submitted_at,
		Code:      code,
		Accepted:  "Accepted",
		Admin:     admin,
	}
	t.ExecuteTemplate(w, "index.html", data)
}
func problemSubmissionsHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get(("id"))
	rows, err := database.DB.Query("SELECT  id,  username,  submitted_at,  verdict  FROM submissions  WHERE submissions.problem_id = ? ORDER BY submissions.id DESC", id)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	defer rows.Close()

	type Submissions struct {
		SubID        string
		Title        string
		Username     string
		Submitted_at string
		Verdict      string
		Accepted     string
		ID           string
	}

	var subs []Submissions
	var title string
	var visibility bool
	database.DB.QueryRow("SELECT title,visibility FROM problems WHERE id=?", id).Scan(&title, &visibility)
	if !visibility {
		fmt.Fprint(w, "No problem found")
		return
	}
	for rows.Next() {
		var p Submissions
		if err := rows.Scan(&p.SubID, &p.Username, &p.Submitted_at, &p.Verdict); err != nil {
			continue
		}
		p.Title = title
		p.Accepted = "Accepted"
		title = p.Title
		p.ID = id
		subs = append(subs, p)
	}
	type PageData struct {
		Title        string
		Pusername    string
		Logout       string
		Items        []Submissions
		ProblemTitle string
		Admin        bool
	}
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var admin bool = false
	var user_type string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&user_type)
	if user_type == "admin" {
		admin = true
	}
	Data := PageData{
		Title:        "submissions - problem - " + id + " - NoobOJ",
		Pusername:    username,
		Logout:       "Logout",
		Items:        subs,
		ProblemTitle: title,
		Admin:        admin,
	}

	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/problem_submissions.html",
		"templates/footer.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.ExecuteTemplate(w, "index.html", Data)

}
func userSubmissionsHandler(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get(("username"))
	rows, err := database.DB.Query("SELECT id,problem_id,submitted_at,verdict FROM submissions WHERE username=? ORDER BY id DESC", user)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	defer rows.Close()

	type Submissions struct {
		SubID        string
		Title        string
		Username     string
		Submitted_at string
		Verdict      string
		Accepted     string
		ID           string
	}

	var subs []Submissions
	for rows.Next() {
		var p Submissions
		if err := rows.Scan(&p.SubID, &p.ID, &p.Submitted_at, &p.Verdict); err != nil {
			continue
		}
		database.DB.QueryRow("SELECT title FROM problems WHERE id=?", p.ID).Scan(&p.Title)
		p.Accepted = "Accepted"
		p.Username = user
		subs = append(subs, p)
	}

	type PageData struct {
		Title     string
		Pusername string
		Username  string
		Logout    string
		Items     []Submissions
		Admin     bool
	}
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var admin bool = false
	var user_type string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&user_type)
	if user_type == "admin" {
		admin = true
	}
	Data := PageData{
		Title:     "submissions -" + user + " - NoobOJ",
		Pusername: username,
		Username:  user,
		Logout:    "Logout",
		Items:     subs,
		Admin:     admin,
	}

	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/user_submissions.html",
		"templates/footer.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.ExecuteTemplate(w, "index.html", Data)

}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		t := template.Must(template.ParseFiles(
			"templates/index.html",
			"templates/register.html",
			"templates/footer.html",
		))
		type PageData struct {
			Title     string
			Pusername string
			Logout    string
			Admin     bool
		}
		data := PageData{
			Title:     "Register - NoobOJ",
			Pusername: "Login",
			Logout:    "Register",
			Admin:     false,
		}
		t.ExecuteTemplate(w, "index.html", data)
		return
	}

	name := r.FormValue("name")
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")
	confirm := r.FormValue("confirm_password")

	if password != confirm {
		fmt.Fprint(w, "Passwords do not match")
		return
	}

	var exists int
	database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username=? OR email=?", username, email).Scan(&exists)
	if exists > 0 {
		fmt.Fprint(w, "Username or Email already exists")
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	_, err := database.DB.Exec("INSERT INTO users(name, username, email, password, user_type) VALUES (?, ?, ?, ?, 'user')",
		name, username, email, hash)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}

	// http.Redirect(w, r, "/login", http.StatusSeeOther)
	session, _ := store.Get(r, "session")
	session.Values["username"] = username
	session.Values["user_type"] = "user"
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		t := template.Must(template.ParseFiles(
			"templates/index.html",
			"templates/login.html",
			"templates/footer.html",
		))
		type PageData struct {
			Title     string
			Pusername string
			Logout    string
			Admin     bool
		}
		data := PageData{
			Title:     "Login - NoobOJ",
			Pusername: "Login",
			Logout:    "Register",
			Admin:     false,
		}
		t.ExecuteTemplate(w, "index.html", data)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	var username string
	var hash string
	var userType string

	err := database.DB.QueryRow("SELECT  username, password, user_type FROM users WHERE email=?", email).
		Scan(&username, &hash, &userType)
	if err == sql.ErrNoRows {
		fmt.Fprint(w, "Email not registered")
		return
	} else if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		fmt.Fprint(w, "Incorrect password")
		return
	}

	session, _ := store.Get(r, "session")
	session.Values["username"] = username
	session.Values["user_type"] = userType
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func requiredLogin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		if session.Values["username"] == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func requiredAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		if session.Values["username"] == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		var usr string
		database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", session.Values["username"]).Scan(&usr)
		if usr != "admin" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	if session.Values["username"] == nil {
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("username")
	row := database.DB.QueryRow("SELECT UPPER(name),DATE_FORMAT(created_on, '%b %Y'),user_type FROM users WHERE username=?", user)
	var name, created_on, is_admin string
	err := row.Scan(&name, &created_on, &is_admin)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/profile.html",
		"templates/footer.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows, err := database.DB.Query("SELECT problem_id,verdict FROM submissions WHERE username=? LIMIT 3", user)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	defer rows.Close()
	type Recent struct {
		ID       string
		Problem  string
		Verdict  string
		Accepted string
	}

	var recent []Recent
	for rows.Next() {
		var p Recent
		if err := rows.Scan(&p.ID, &p.Verdict); err != nil {
			continue
		}
		database.DB.QueryRow("SELECT title FROM problems WHERE id=?", p.ID).Scan(&p.Problem)
		p.Accepted = "Accepted"
		recent = append(recent, p)
	}

	rows, err = database.DB.Query("SELECT id,problem_id,username,submitted_at,verdict FROM submissions WHERE username=? ORDER BY id DESC", user)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	defer rows.Close()

	type Submissions struct {
		SubID        string
		Title        string
		Username     string
		Submitted_at string
		Verdict      string
		Accepted     string
		ID           string
	}

	var subs []Submissions
	for rows.Next() {
		var p Submissions
		if err := rows.Scan(&p.SubID, &p.ID, &p.Username, &p.Submitted_at, &p.Verdict); err != nil {
			continue
		}
		database.DB.QueryRow("SELECT title FROM problems WHERE id=?", p.ID).Scan(&p.Title)
		p.Accepted = "Accepted"
		subs = append(subs, p)
	}

	rows, err = database.DB.Query("SELECT DISTINCT CURRENT_DATE-date FROM streaks WHERE username=? ORDER BY CURRENT_DATE-date", user)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	defer rows.Close()
	streak := 0
	var today bool = false
	var prev int = 0
	for rows.Next() {
		var p int
		rows.Scan(&p)
		if p == 0 || p == 1 || p == prev+1 {
			if p == 0 {
				today = true
			}
			streak = streak + 1
			prev = p
		} else {
			break
		}
	}
	type PageData struct {
		Title      string
		Pusername  string
		Logout     string
		Username   string
		Name       string
		Time       string
		Rank       int
		GlobalRank int
		Recents    []Recent
		Solved     int
		Accuracy   int
		Subs       []Submissions
		Streak     int
		Today      bool
		Unsolved   []string
		Admin      bool
		User       bool
	}
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var admin bool = false
	var user_type string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&user_type)
	if user_type == "admin" {
		admin = true
	}
	var solved float32
	var accuracy float32
	database.DB.QueryRow("SELECT  COUNT(DISTINCT problem_id) FROM submissions WHERE username=? AND verdict='Accepted'", user).Scan(&solved)
	database.DB.QueryRow("SELECT  COUNT(DISTINCT problem_id) FROM submissions WHERE username=?", user).Scan(&accuracy)
	rows, err = database.DB.Query("select users.username from users,submissions where users.username=submissions.username and verdict='Accepted' group by users.username,name order by count(distinct submissions.problem_id) desc")
	if err != nil {
		fmt.Fprint(w, "Database error "+err.Error())
		return
	}
	defer rows.Close()

	var rank int
	if int(solved) >= 50 {
		rank = 2
	} else if int(solved) >= 10 {
		rank = 1
	} else {
		rank = 0
	}
	if is_admin == "admin" {
		rank = 3
	}
	global_rank := 0
	found := false
	for rows.Next() {
		var p string
		rows.Scan(&p)
		global_rank = global_rank + 1
		if p == user {
			found = true
			break
		}
	}
	if !found {
		global_rank = 0
	}

	rows, err = database.DB.Query("SELECT problem_id FROM submissions WHERE username =? GROUP BY problem_id HAVING SUM(verdict = 'Accepted') = 0;", user)
	if err != nil {
		fmt.Fprint(w, "Database error "+err.Error())
	}
	defer rows.Close()
	var unsolved []string
	for rows.Next() {
		var p string
		err := rows.Scan(&p)
		if err != nil {
			continue
		}
		unsolved = append(unsolved, p)
	}

	if accuracy != 0 {
		accuracy = (solved / accuracy) * 100
	}
	data := PageData{
		Title:      user + " - NoobOJ",
		Pusername:  username,
		Logout:     "Logout",
		Username:   user,
		Name:       name,
		Time:       created_on,
		Subs:       subs,
		Rank:       rank,
		GlobalRank: global_rank,
		Recents:    recent,
		Solved:     int(solved),
		Accuracy:   int(accuracy),
		Streak:     streak,
		Today:      today,
		Unsolved:   unsolved,
		Admin:      admin,
		User:       is_admin == "admin",
	}
	t.ExecuteTemplate(w, "index.html", data)
}
func administrationHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/administration.html",
		"templates/footer.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows, err := database.DB.Query("SELECT id,title, visibility FROM problems WHERE author=? ORDER BY id DESC", username)
	if err != nil {
		fmt.Fprint(w, "Databae error: "+err.Error())
		return
	}
	defer rows.Close()

	type Problem struct {
		ID         string
		Title      string
		Visibility bool
	}
	var list []Problem
	for rows.Next() {
		var p Problem
		if err := rows.Scan(&p.ID, &p.Title, &p.Visibility); err != nil {
			continue
		}
		list = append(list, p)
	}
	rows, err = database.DB.Query("SELECT id,title,visibility FROM authors join contests on authors.contest_id=contests.id WHERE author=? ORDER BY id DESC", username)
	if err != nil {
		fmt.Fprint(w, "Databae error: "+err.Error())
		return
	}
	defer rows.Close()

	type Contest struct {
		ID         string
		Title      string
		Visibility bool
	}
	var list2 []Contest
	for rows.Next() {
		var p Contest
		if err := rows.Scan(&p.ID, &p.Title, &p.Visibility); err != nil {
			continue
		}
		list2 = append(list2, p)
	}

	type PageData struct {
		Title     string
		Pusername string
		Logout    string
		Admin     bool
		Problems  []Problem
		Contests  []Contest
	}

	data := PageData{
		Title:     "Administration - NoobOJ",
		Pusername: username,
		Logout:    "Logout",
		Admin:     true,
		Problems:  list,
		Contests:  list2,
	}
	t.ExecuteTemplate(w, "index.html", data)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	id := r.FormValue("id")
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var author string
	err := database.DB.QueryRow("SELECT author FROM problems WHERE id=?", id).Scan(&author)
	if err != nil {
		fmt.Fprint(w, "No problem found")
		return
	}
	if username != author {
		fmt.Fprint(w, "Access denied")
		return
	}

	if r.Method == "GET" {

		t, err := template.ParseFiles(
			"templates/index.html",
			"templates/delete.html",
			"templates/footer.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		type PageData struct {
			Title     string
			Pusername string
			Logout    string
			Admin     bool
			ID        string
		}

		data := PageData{
			Title:     "Administration - NoobOJ",
			Pusername: username,
			Logout:    "Logout",
			Admin:     true,
			ID:        id,
		}
		t.ExecuteTemplate(w, "index.html", data)
		return
	}

	database.DB.Exec("DELETE FROM problems WHERE id=?", id)
	http.Redirect(w, r, "/administration?tab=problem", http.StatusSeeOther)

}

func kola(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		t, err := template.ParseFiles(
			"templates/index.html",
			"templates/new_problem.html",
			"templates/footer.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		session, _ := store.Get(r, "session")
		username, _ := session.Values["username"].(string)

		rows, err := database.DB.Query("SELECT id,name FROM tags")
		if err != nil {
			fmt.Fprint(w, "Error getting tags : "+err.Error())
			return
		}
		defer rows.Close()

		type TagData struct {
			ID   int
			Name string
		}
		var tags []TagData
		for rows.Next() {
			var p TagData
			if err := rows.Scan(&p.ID, &p.Name); err != nil {
				continue
			}
			tags = append(tags, p)
		}

		type PageData struct {
			Title     string
			Pusername string
			Logout    string
			Admin     bool
			Tags      []TagData
		}
		data := PageData{
			Title:     "Create Problem - NoobOJ",
			Pusername: username,
			Logout:    "Logout",
			Admin:     true,
			Tags:      tags,
		}
		t.ExecuteTemplate(w, "index.html", data)
		return
	}
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Parse error", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	statement := r.FormValue("statement")
	inputDesc := r.FormValue("input_desc")
	outputDesc := r.FormValue("output_desc")
	constraints := r.FormValue("constraints")
	tagsRaw := r.FormValue("tags")
	rating := r.FormValue("rating")
	time := r.FormValue("time")
	memory := r.FormValue("memory")
	editorial := r.FormValue("editorial")
	code := r.FormValue("code")

	// Start a Transaction
	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, "Transaction error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Defer a rollback in case of any early returns
	defer tx.Rollback()
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	res, err := tx.Exec("INSERT INTO problems (title, statement, input, output, constraints, author,time_limit,memory_limit,editorial,editorial_code) VALUES (?, ?, ?, ?, ?,?,?,?,?,?)", title, statement, inputDesc, outputDesc, constraints, username, time, memory, editorial, code)
	if err != nil {
		fmt.Fprint(w, "Error inserting problem: "+err.Error())
		return
	}
	problemID, err := res.LastInsertId()
	if err != nil {
		fmt.Fprint(w, "Error getting last ID: "+err.Error())
		return
	}
	res, err = tx.Exec("INSERT INTO ratings(problem_id,rating) VALUES(?,?)", problemID, rating)
	if err != nil {
		fmt.Fprint(w, "Error inserting rating: "+err.Error())
		return
	}
	// Insert Tags (Splitting by comma)

	if tagsRaw != "" {
		tags := strings.Split(tagsRaw, ",")
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				_, err = tx.Exec("INSERT INTO problem_tags (problem_id, tag_id) VALUES (?, ?)", problemID, tag)
				if err != nil {
					fmt.Fprint(w, "Error inserting tags: "+err.Error())
					return
				}
			}
		}
	}

	testInputs := r.Form["test_input[]"]
	testOutputs := r.Form["test_output[]"]
	testTypes := r.Form["test_type[]"]

	for i := 0; i < len(testInputs); i++ {
		if testInputs[i] == "" && testOutputs[i] == "" {
			continue
		}

		_, err = tx.Exec("INSERT INTO test_cases (problem_id, input, output, type) VALUES (?, ?, ?, ?)", problemID, testInputs[i], testOutputs[i], testTypes[i])
		if err != nil {
			fmt.Fprint(w, "Error inserting test case: "+err.Error())
			return
		}
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		fmt.Fprint(w, "Commit error: "+err.Error())
		return
	}

	http.Redirect(w, r, "/administration?tab=problem", http.StatusSeeOther)
}

func editProblemHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var author string
	err := database.DB.QueryRow("SELECT author FROM problems WHERE id=?", id).Scan(&author)
	if err != nil {
		fmt.Fprint(w, "No problem found")
		return
	}
	if username != author {
		fmt.Fprint(w, "Access denied")
		return
	}

	if r.Method == "GET" {
		funcMap := template.FuncMap{
			"eq": func(a, b interface{}) bool {
				return a == b
			},
		}

		t, err := template.New("base").Funcs(funcMap).ParseFiles(
			"templates/index.html",
			"templates/edit_problem.html",
			"templates/footer.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var title, statement, input, output, constraints, editorial, code string
		var time, memory, rating int
		err = database.DB.QueryRow(`SELECT 
title,
IFNULL(statement, ''),
IFNULL(input, ''),
IFNULL(output, ''),
IFNULL(constraints, ''),
time_limit,
memory_limit,
IFNULL(editorial, ''),
IFNULL(editorial_code, '')
FROM problems WHERE id=?`, id).Scan(&title, &statement, &input, &output, &constraints, &time, &memory, &editorial, &code)
		database.DB.QueryRow("SELECT rating FROM ratings WHERE problem_id=?", id).Scan(&rating)
		rows, err := database.DB.Query("SELECT id,name from tags,problem_tags where tags.id=problem_tags.tag_id and problem_id=?", id)
		if err != nil {
			fmt.Fprint(w, "Error getting tags : "+err.Error())
			return
		}
		defer rows.Close()

		type TagData struct {
			ID   int
			Name string
		}
		var selectedTags []TagData
		for rows.Next() {
			var p TagData
			if err := rows.Scan(&p.ID, &p.Name); err != nil {
				continue
			}
			selectedTags = append(selectedTags, p)
		}

		rows, err = database.DB.Query("SELECT id,name FROM tags")
		if err != nil {
			fmt.Fprint(w, "Error getting tags : "+err.Error())
			return
		}
		defer rows.Close()

		var tags []TagData
		for rows.Next() {
			var p TagData
			if err := rows.Scan(&p.ID, &p.Name); err != nil {
				continue
			}
			tags = append(tags, p)
		}

		rows, err = database.DB.Query("SELECT input,output,type From test_cases WHERE problem_id=?", id)
		type Testdata struct {
			Input  string
			Output string
			Type   string
		}
		var test []Testdata
		for rows.Next() {
			var p Testdata
			rows.Scan(&p.Input, &p.Output, &p.Type)
			test = append(test, p)
		}
		type PageData struct {
			Title        string
			Pusername    string
			Logout       string
			Admin        bool
			ID           string
			Ptitle       string
			Statement    string
			Input        string
			Output       string
			Constraints  string
			Rating       int
			SelectedTags []TagData
			Tags         []TagData
			Time         int
			Memory       int
			Editorial    string
			Code         string
			Test         []Testdata
			Hidden       string
		}

		data := PageData{
			Title:        "Administration - NoobOJ",
			Pusername:    username,
			Logout:       "Logout",
			Admin:        true,
			ID:           id,
			Ptitle:       title,
			Statement:    statement,
			Input:        input,
			Output:       output,
			Constraints:  constraints,
			Rating:       rating,
			SelectedTags: selectedTags,
			Tags:         tags,
			Test:         test,
			Time:         time,
			Memory:       memory,
			Editorial:    editorial,
			Code:         code,
			Hidden:       "hidden",
		}
		t.ExecuteTemplate(w, "index.html", data)
	}
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Parse error", http.StatusBadRequest)
			return
		}

		title := r.FormValue("title")
		statement := r.FormValue("statement")
		inputDesc := r.FormValue("input_desc")
		outputDesc := r.FormValue("output_desc")
		constraints := r.FormValue("constraints")
		tagsRaw := r.FormValue("tags")
		rating := r.FormValue("rating")
		time := r.FormValue("time")
		memory := r.FormValue("memory")
		editorial := r.FormValue("editorial")
		code := r.FormValue("code")

		tx, err := database.DB.Begin()
		if err != nil {
			http.Error(w, "Transaction error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Defer a rollback in case of any early returns
		defer tx.Rollback()
		_, err = tx.Exec("UPDATE problems SET title=?, statement=?, input=?, output=?, constraints=?, time_limit=?, memory_limit=?, editorial=?, editorial_code=? WHERE id=?", title, statement, inputDesc, outputDesc, constraints, time, memory, editorial, code, id)
		if err != nil {
			fmt.Fprint(w, "Error updating problem: "+err.Error())
			return
		}
		problemID := id
		_, err = tx.Exec("UPDATE ratings SET rating=? WHERE problem_id=?", rating, problemID)
		if err != nil {
			fmt.Fprint(w, "Error updating rating: "+err.Error())
			return
		}
		// Insert Tags (Splitting by comma)
		_, err = tx.Exec("DELETE FROM problem_tags WHERE problem_id=?", problemID)
		if err != nil {
			fmt.Fprint(w, "Error deleting old tags: "+err.Error())
			return
		}
		if tagsRaw != "" {
			tags := strings.Split(tagsRaw, ",")
			for _, tag := range tags {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					_, err = tx.Exec("INSERT IGNORE INTO problem_tags (problem_id, tag_id) VALUES (?, ?)", problemID, tag)
					if err != nil {
						fmt.Fprint(w, "Error inserting tags: "+err.Error())
						return
					}
				}
			}
		}

		testInputs := r.Form["test_input[]"]
		testOutputs := r.Form["test_output[]"]
		testTypes := r.Form["test_type[]"]

		_, err = tx.Exec("DELETE FROM test_cases WHERE problem_id=?", problemID)
		if err != nil {
			fmt.Fprint(w, "Error deleting old test cases: "+err.Error())
			return
		}

		for i := 0; i < len(testInputs); i++ {
			if testInputs[i] == "" && testOutputs[i] == "" {
				continue
			}

			_, err = tx.Exec("INSERT INTO test_cases (problem_id, input, output, type) VALUES (?, ?, ?, ?)", problemID, testInputs[i], testOutputs[i], testTypes[i])
			if err != nil {
				fmt.Fprint(w, "Error inserting test case: "+err.Error())
				return
			}
		}

		// Commit the transaction
		err = tx.Commit()
		if err != nil {
			fmt.Fprint(w, "Commit error: "+err.Error())
			return
		}

		http.Redirect(w, r, "/administration?tab=problem", http.StatusSeeOther)
	}

}
func editContestHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var role string
	err := database.DB.QueryRow("SELECT role FROM authors WHERE contest_id=? AND author=?", id, username).Scan(&role)
	if err != nil {
		fmt.Fprint(w, "No contest found")
		return
	}
	if role != "owner" && role != "moderator" {
		fmt.Fprint(w, "Access denied")
		return
	}

	if r.Method == "GET" {
		t, err := template.ParseFiles(
			"templates/index.html",
			"templates/edit_contest.html",
			"templates/footer.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var title string
		var startBytes, endBytes []byte
		var visibility bool

		err = database.DB.QueryRow(
			"SELECT title, start_time, end_time, visibility FROM contests WHERE id=?",
			id,
		).Scan(&title, &startBytes, &endBytes, &visibility)
		if err != nil {
			fmt.Fprint(w, "error scanning contest details: "+err.Error())
			return
		}

		layout := "2006-01-02 15:04:05"
		startTimeUTC, err := time.Parse(layout, string(startBytes))
		if err != nil {
			fmt.Fprint(w, "invalid start time from DB: "+err.Error())
			return
		}
		endTimeUTC, err := time.Parse(layout, string(endBytes))
		if err != nil {
			fmt.Fprint(w, "invalid end time from DB: "+err.Error())
			return
		}

		rows, err := database.DB.Query("SELECT id,title from problems WHERE author=?", username)
		if err != nil {
			fmt.Fprint(w, "error retriving available problems : "+err.Error())
			return
		}
		defer rows.Close()

		type Problem struct {
			ID    int
			Title string
			Score int
		}
		var available []Problem
		for rows.Next() {
			var p Problem
			if err := rows.Scan(&p.ID, &p.Title); err != nil {
				continue
			}
			available = append(available, p)
		}

		rows, err = database.DB.Query("SELECT problems.id,problems.title,contest.score FROM problems,(SELECT * FROM tasks WHERE contest_id=?) as contest where problems.id = contest.problem_id", id)
		if err != nil {
			fmt.Fprint(w, "error retriving selected problems : "+err.Error())
			return
		}
		defer rows.Close()
		var selected []Problem
		for rows.Next() {
			var p Problem
			if err := rows.Scan(&p.ID, &p.Title, &p.Score); err != nil {
				continue
			}
			selected = append(selected, p)
		}

		// Send raw UTC with Z suffix — browser JS will convert to local time
		data := struct {
			Title             string
			Pusername         string
			Logout            string
			Admin             bool
			ID                string
			Ptitle            string
			Start             int64
			End               int64
			AvailableProblems []Problem
			SelectedProblems  []Problem
		}{
			Title:             "Administration - NoobOJ",
			Pusername:         username,
			Logout:            "Logout",
			Admin:             true,
			ID:                id,
			Ptitle:            title,
			Start:             startTimeUTC.UTC().UnixMilli(),
			End:               endTimeUTC.UTC().UnixMilli(),
			AvailableProblems: available,
			SelectedProblems:  selected,
		}

		t.ExecuteTemplate(w, "index.html", data)
	}
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Parse error", http.StatusBadRequest)
			return
		}
		// ids := r.Form["problem_ids"]
		// scores := r.Form["problem_scores"]
		// database.DB.Exec("DELETE FROM tasks WHERE contest_id=?")
		// for i := range ids {
		// 	database.DB.Exec("INSERT INTO tasks(problem_id,contest_id,)")
		// }
		return
	}
	// if r.Method == "POST" {
	// 	err := r.ParseForm()
	// 	if err != nil {
	// 		http.Error(w, "Parse error", http.StatusBadRequest)
	// 		return
	// 	}

	// 	title := r.FormValue("title")
	// 	statement := r.FormValue("statement")
	// 	inputDesc := r.FormValue("input_desc")
	// 	outputDesc := r.FormValue("output_desc")
	// 	constraints := r.FormValue("constraints")
	// 	tagsRaw := r.FormValue("tags")
	// 	rating := r.FormValue("rating")
	// 	time := r.FormValue("time")
	// 	memory := r.FormValue("memory")
	// 	editorial := r.FormValue("editorial")
	// 	code := r.FormValue("code")

	// 	tx, err := database.DB.Begin()
	// 	if err != nil {
	// 		http.Error(w, "Transaction error: "+err.Error(), http.StatusInternalServerError)
	// 		return
	// 	}

	// 	// Defer a rollback in case of any early returns
	// 	defer tx.Rollback()
	// 	// session, _ := store.Get(r, "session")
	// 	// username, _ := session.Values["username"].(string)
	// 	_, err = tx.Exec("UPDATE problems SET title=?, statement=?, input=?, output=?, constraints=?, time_limit=?, memory_limit=?, editorial=?, code=? WHERE id=?", title, statement, inputDesc, outputDesc, constraints, time, memory, editorial, code, id)
	// 	if err != nil {
	// 		fmt.Fprint(w, "Error updating problem: "+err.Error())
	// 		return
	// 	}
	// 	problemID := id
	// 	_, err = tx.Exec("UPDATE ratings SET rating=? WHERE problem_id=?", rating, problemID)
	// 	if err != nil {
	// 		fmt.Fprint(w, "Error updating rating: "+err.Error())
	// 		return
	// 	}
	// 	// Insert Tags (Splitting by comma)
	// 	_, err = tx.Exec("DELETE FROM tags WHERE problem_id=?", problemID)
	// 	if err != nil {
	// 		fmt.Fprint(w, "Error deleting old tags: "+err.Error())
	// 		return
	// 	}
	// 	if tagsRaw != "" {
	// 		tags := strings.Split(tagsRaw, ",")
	// 		for _, tag := range tags {
	// 			tag = strings.TrimSpace(tag)
	// 			if tag != "" {
	// 				_, err = tx.Exec("INSERT INTO tags (problem_id, tag) VALUES (?, ?)", problemID, tag)
	// 				if err != nil {
	// 					fmt.Fprint(w, "Error inserting tags: "+err.Error())
	// 					return
	// 				}
	// 			}
	// 		}
	// 	}

	// 	testInputs := r.Form["test_input[]"]
	// 	testOutputs := r.Form["test_output[]"]
	// 	testTypes := r.Form["test_type[]"]

	// 	_, err = tx.Exec("DELETE FROM test_cases WHERE problem_id=?", problemID)
	// 	if err != nil {
	// 		fmt.Fprint(w, "Error deleting old test cases: "+err.Error())
	// 		return
	// 	}

	// 	for i := 0; i < len(testInputs); i++ {
	// 		if testInputs[i] == "" && testOutputs[i] == "" {
	// 			continue
	// 		}

	// 		_, err = tx.Exec("INSERT INTO test_cases (problem_id, input, output, type) VALUES (?, ?, ?, ?)", problemID, testInputs[i], testOutputs[i], testTypes[i])
	// 		if err != nil {
	// 			fmt.Fprint(w, "Error inserting test case: "+err.Error())
	// 			return
	// 		}
	// 	}

	// 	// Commit the transaction
	// 	err = tx.Commit()
	// 	if err != nil {
	// 		fmt.Fprint(w, "Commit error: "+err.Error())
	// 		return
	// 	}

	// 	http.Redirect(w, r, "/administration?tab=problem", http.StatusSeeOther)
	// }

}

func contestsHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var admin bool = false
	var user_type string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&user_type)
	if user_type == "admin" {
		admin = true
	}
	rows, err := database.DB.Query("SELECT id, title, start_time, end_time FROM contests")
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	defer rows.Close()

	type Contest struct {
		ID        int
		Title     string
		StartTime string
		EndTime   string
	}
	var list []Contest
	for rows.Next() {
		var p Contest
		rows.Scan(&p.ID, &p.Title, &p.StartTime, &p.EndTime)

		list = append(list, p)
	}
	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/contests.html",
		"templates/footer.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type PageData struct {
		Title     string
		Contests  []Contest
		Logout    string
		Pusername string
		Admin     bool
	}
	data := PageData{
		Title:     "Contests - NoobOJ",
		Contests:  list,
		Logout:    "Logout",
		Pusername: username,
		Admin:     admin,
	}
	t.ExecuteTemplate(w, "index.html", data)

}

func toggleProblemVisibility(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		session, _ := store.Get(r, "session")
		username, _ := session.Values["username"].(string)
		id := r.FormValue("id")
		var author string
		err := database.DB.QueryRow("SELECT author FROM problems WHERE id=?", id).Scan(&author)
		if err != nil {
			fmt.Fprint(w, "No problem found")
			return
		}
		if username != author {
			fmt.Fprint(w, "Access denied")
			return
		}

		_, err = database.DB.Exec("update problems SET visibility = NOT visibility where id =?", id)
		if err != nil {
			fmt.Fprint(w, "Error updating visibility "+err.Error())
			return
		}

	}
}

func announcementHandler(w http.ResponseWriter, r *http.Request) {
}

func main() {
	// Start 4 parallel judges (Change to 2 or 8 depending on your CPU cores)
	startJudgeWorkers(2)
	fmt.Println("Starting server...")
	database.Connect()
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("templates"))))
	// public pages
	http.HandleFunc("/", home)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/register", registerHandler)
	// admin pages
	http.HandleFunc("/new/problem", requiredAdmin(newProblemHandler))
	http.HandleFunc("/new/contest", requiredAdmin(newContestHandler))
	http.HandleFunc("/admin/toggle", requiredAdmin(toggleProblemVisibility))

	// user pages
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/problemset", requiredLogin(problemsetHandler))
	http.HandleFunc("/problem", requiredLogin(viewProblemHandler))
	http.HandleFunc("/submission", requiredLogin(viewSubmissionHandler))
	http.HandleFunc("/submissions/problem", requiredLogin(problemSubmissionsHandler))
	http.HandleFunc("/submissions/user", requiredLogin(userSubmissionsHandler))
	http.HandleFunc("/submit", requiredLogin(submitHandler))
	http.HandleFunc("/profile", requiredLogin(profileHandler))
	http.HandleFunc("/leaderboard", requiredLogin(leaderboardHandler))
	http.HandleFunc("/administration", requiredAdmin(administrationHandler))
	http.HandleFunc("/delete/problem", requiredAdmin(deleteHandler))
	http.HandleFunc("/edit/problem", requiredAdmin(editProblemHandler))
	http.HandleFunc("/edit/contest", requiredAdmin(editContestHandler))
	http.HandleFunc("/contests", requiredLogin(contestsHandler))
	http.HandleFunc("/announcement", requiredAdmin(announcementHandler))
	http.ListenAndServe(":8080", nil)
}

var submissionQueue = make(chan int, 100)

func submitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	problemID := r.FormValue("problem_id")
	code := r.FormValue("code")
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)

	// 1. Save with "Pending" status so the user sees it's in the system
	res, err := database.DB.Exec("INSERT INTO submissions(problem_id, username, code) VALUES (?, ?, ?)", problemID, username, code)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}

	subID64, _ := res.LastInsertId()
	subID := int(subID64)
	// pushing into the queue
	submissionQueue <- subID
	http.Redirect(w, r, "/submission?id="+strconv.Itoa(subID), http.StatusSeeOther)
}

func startJudgeWorkers(count int) {
	for i := 0; i < count; i++ {
		go func(workerID int) {
			for subID := range submissionQueue {
				processSubmission(subID)
			}
		}(i)
	}
}

func processSubmission(subID int) {
	type Submission struct {
		ID        int
		ProblemID string
		Code      string
	}
	var s Submission
	var username string
	err := database.DB.QueryRow("SELECT id, problem_id, code, username FROM submissions WHERE id = ?", subID).Scan(&s.ID, &s.ProblemID, &s.Code, &username)
	if err != nil {
		return
	}

	database.DB.Exec("UPDATE submissions SET verdict='Judging' WHERE id=?", subID)

	os.MkdirAll("temp_judge", 0755)
	fileName := fmt.Sprintf("temp_judge/sub_%d.cpp", subID)
	exeName := fmt.Sprintf("temp_judge/sub_%d.out", subID)
	os.WriteFile(fileName, []byte(s.Code), 0644)

	defer os.Remove(fileName)
	defer os.Remove(exeName)

	compileCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(compileCtx, "g++", "-O3", fileName, "-o", exeName)
	if err := cmd.Run(); err != nil {
		database.DB.Exec("UPDATE submissions SET verdict='Compilation Error' WHERE id=?", subID)
		return
	}

	rows, _ := database.DB.Query("SELECT input, output FROM test_cases WHERE problem_id = ?", s.ProblemID)
	defer rows.Close()

	finalVerdict := "Accepted"
	for rows.Next() {
		var input, expected string
		rows.Scan(&input, &expected)

		// TLE Protection: 1 second limit
		runCtx, runCancel := context.WithTimeout(context.Background(), 10*time.Second)

		runCmd := exec.CommandContext(runCtx, "./"+exeName)
		runCmd.Stdin = strings.NewReader(input)
		out, err := runCmd.Output()
		runCancel()

		if runCtx.Err() == context.DeadlineExceeded {
			finalVerdict = "Time Limit Exceeded"
			break
		}
		if err != nil {
			finalVerdict = "Runtime Error"
			break
		}
		if strings.TrimSpace(string(out)) != strings.TrimSpace(expected) {
			finalVerdict = "Wrong Answer"
			break
		}
	}
	if finalVerdict == "Accepted" {
		database.DB.Exec("INSERT into streaks(username,problem_id) VALUES(?,?)", username, s.ProblemID)
	}
	database.DB.Exec("UPDATE submissions SET verdict=? WHERE id=?", finalVerdict, subID)
}
