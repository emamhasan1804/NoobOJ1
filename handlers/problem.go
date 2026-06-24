package handlers

import (
	"NoobOJ/database"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

func ProblemsetHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var userType string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&userType)
	admin := userType == "admin"

	rows, err := database.DB.Query("SELECT id, title FROM problems WHERE visibility=1")
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
		Rating int
		Tried  int
	}

	var list []Problem
	for rows.Next() {
		var p Problem
		if err := rows.Scan(&p.ID, &p.Title); err != nil {
			continue
		}
		database.DB.QueryRow("SELECT COUNT(DISTINCT username) FROM submissions WHERE problem_id=? AND verdict='Accepted'", p.ID).Scan(&p.Solved)

		var rawRating sql.NullInt64
		err = database.DB.QueryRow("SELECT rating FROM ratings WHERE problem_id=? LIMIT 1", p.ID).Scan(&rawRating)
		if err == nil && rawRating.Valid {
			p.Rating = int(rawRating.Int64)
			if p.Rating >= 0 {
				p.Rating = ((p.Rating + 50) / 100) * 100
			} else {
				p.Rating = ((p.Rating - 50) / 100) * 100
			}
		}

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

	search := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))
	if sortBy == "" {
		sortBy = "id_asc"
	}
	page, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("page")))
	if page < 1 {
		page = 1
	}

	filtered := make([]Problem, 0, len(list))
	for _, p := range list {
		if search == "" {
			filtered = append(filtered, p)
			continue
		}

		matched := strings.Contains(strings.ToLower(p.Title), search) ||
			strings.Contains(strconv.Itoa(p.ID), search) ||
			strings.Contains(strconv.Itoa(p.Rating), search)

		if !matched {
			for _, tag := range p.Tags {
				if strings.Contains(strings.ToLower(tag), search) {
					matched = true
					break
				}
			}
		}
		if matched {
			filtered = append(filtered, p)
		}
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		a, b := filtered[i], filtered[j]
		switch sortBy {
		case "id_desc":
			return a.ID > b.ID
		case "title_asc":
			return strings.ToLower(a.Title) < strings.ToLower(b.Title)
		case "title_desc":
			return strings.ToLower(a.Title) > strings.ToLower(b.Title)
		case "rating_asc":
			if a.Rating == b.Rating {
				return a.ID < b.ID
			}
			return a.Rating < b.Rating
		case "rating_desc":
			if a.Rating == b.Rating {
				return a.ID < b.ID
			}
			return a.Rating > b.Rating
		case "solved_asc":
			if a.Solved == b.Solved {
				return a.ID < b.ID
			}
			return a.Solved < b.Solved
		case "solved_desc":
			if a.Solved == b.Solved {
				return a.ID < b.ID
			}
			return a.Solved > b.Solved
		default:
			return a.ID < b.ID
		}
	})

	const pageSize = 25
	total := len(filtered)
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		page = 1
	} else if page > totalPages {
		page = totalPages
	}

	pageItems := []Problem{}
	if total > 0 {
		start := (page - 1) * pageSize
		end := start + pageSize
		if end > total {
			end = total
		}
		pageItems = filtered[start:end]
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
		Title      string
		Problems   []Problem
		Logout     string
		Pusername  string
		Admin      bool
		Search     string
		Sort       string
		Page       int
		PrevPage   int
		NextPage   int
		HasPrev    bool
		HasNext    bool
		Total      int
		TotalPages int
		PageSize   int
	}

	data := PageData{
		Title:      "Problemset - NoobOJ",
		Problems:   pageItems,
		Logout:     "Logout",
		Pusername:  username,
		Admin:      admin,
		Search:     strings.TrimSpace(r.URL.Query().Get("q")),
		Sort:       sortBy,
		Page:       page,
		PrevPage:   page - 1,
		NextPage:   page + 1,
		HasPrev:    page > 1,
		HasNext:    totalPages > 0 && page < totalPages,
		Total:      total,
		TotalPages: totalPages,
		PageSize:   pageSize,
	}
	t.ExecuteTemplate(w, "index.html", data)
}
func ViewProblemHandler(w http.ResponseWriter, r *http.Request) {
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
		Number int
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
		p.Number = len(list) + 1
		list = append(list, p)
	}
	rows, err = database.DB.Query("SELECT name FROM tags,problem_tags WHERE problem_id=? AND tags.id=problem_tags.tag_id", id)
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
	if strings.TrimSpace(rating) != "" {
		tags = append(tags, "*"+rating)
	}

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
func ViewSubmissionHandler(w http.ResponseWriter, r *http.Request) {
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
	var contest bool
	err = database.DB.QueryRow("SELECT EXISTS (select * from contests where visibility=1 and UTC_TIMESTAMP BETWEEN start_time and end_time limit 1)").Scan(&contest)
	if err != nil {
		fmt.Fprint(w, "Problem title not found")
		return
	}
	if contest && user != username {
		referer := r.Header.Get("Referer")
		if referer == "" {
			referer = "/"
		}
		http.Redirect(w, r, referer, http.StatusSeeOther)
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
func ProblemSubmissionsHandler(w http.ResponseWriter, r *http.Request) {
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
func UserSubmissionsHandler(w http.ResponseWriter, r *http.Request) {
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
func EditorialHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var admin bool = false
	var user_type string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&user_type)
	if user_type == "admin" {
		admin = true
	}
	id := r.URL.Query().Get("id")
	row := database.DB.QueryRow("SELECT title, statement,input,output,constraints,author,time_limit,memory_limit, editorial, editorial_code FROM problems WHERE id = ? and visibility=1", id)
	var title, statement, input, output, constraints, author, time, memory, editorial, editorial_code string
	err := row.Scan(&title, &statement, &input, &output, &constraints, &author, &time, &memory, &editorial, &editorial_code)
	if err != nil {
		fmt.Fprint(w, "Problem not found")
		return
	}
	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/editorial.html",
		"templates/footer.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type PageData struct {
		Title         string
		Ptitle        string
		Pusername     string
		Logout        string
		Author        string
		ID            string
		Admin         bool
		One           string
		Editorial     string
		EditorialCode string
	}
	data := PageData{
		Title:         "Editorial - " + id + " - NoobOJ",
		Ptitle:        title,
		Pusername:     username,
		Logout:        "Logout",
		Author:        author,
		ID:            id,
		Admin:         admin,
		One:           "1",
		Editorial:     editorial,
		EditorialCode: editorial_code,
	}
	t.ExecuteTemplate(w, "index.html", data)
}
