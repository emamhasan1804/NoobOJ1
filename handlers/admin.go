package handlers

import (
	"NoobOJ/database"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func NewProblemHandler(w http.ResponseWriter, r *http.Request) {
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
func NewContestHandler(w http.ResponseWriter, r *http.Request) {
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

func AdministrationHandler(w http.ResponseWriter, r *http.Request) {
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

	problemVisibility := strings.TrimSpace(r.URL.Query().Get("problem_visibility"))
	if problemVisibility != "published" && problemVisibility != "unpublished" {
		problemVisibility = "all"
	}
	contestVisibility := strings.TrimSpace(r.URL.Query().Get("contest_visibility"))
	if contestVisibility != "published" && contestVisibility != "unpublished" {
		contestVisibility = "all"
	}
	contestTime := strings.TrimSpace(r.URL.Query().Get("contest_time"))
	if contestTime != "running" && contestTime != "upcoming" && contestTime != "past" {
		contestTime = "all"
	}

	problemQuery := "SELECT id,title, visibility FROM problems WHERE author=?"
	problemArgs := []interface{}{username}
	if problemVisibility == "published" {
		problemQuery += " AND visibility=1"
	} else if problemVisibility == "unpublished" {
		problemQuery += " AND visibility=0"
	}
	problemQuery += " ORDER BY id DESC"

	rows, err := database.DB.Query(problemQuery, problemArgs...)
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
	contestQuery := "SELECT contests.id,title,visibility,start_time,end_time FROM authors join contests on authors.contest_id=contests.id WHERE author=?"
	contestArgs := []interface{}{username}
	if contestVisibility == "published" {
		contestQuery += " AND visibility=1"
	} else if contestVisibility == "unpublished" {
		contestQuery += " AND visibility=0"
	}
	switch contestTime {
	case "running":
		contestQuery += " AND start_time<=UTC_TIMESTAMP() AND end_time>=UTC_TIMESTAMP()"
	case "upcoming":
		contestQuery += " AND start_time>UTC_TIMESTAMP()"
	case "past":
		contestQuery += " AND end_time<UTC_TIMESTAMP()"
	}
	contestQuery += " ORDER BY contests.id DESC"

	rows, err = database.DB.Query(contestQuery, contestArgs...)
	if err != nil {
		fmt.Fprint(w, "Databae error: "+err.Error())
		return
	}
	defer rows.Close()

	type Contest struct {
		ID         string
		Title      string
		Visibility bool
		StartTime  string
		EndTime    string
	}
	var list2 []Contest
	for rows.Next() {
		var p Contest
		if err := rows.Scan(&p.ID, &p.Title, &p.Visibility, &p.StartTime, &p.EndTime); err != nil {
			continue
		}
		list2 = append(list2, p)
	}

	type PageData struct {
		Title             string
		Pusername         string
		Logout            string
		Admin             bool
		Problems          []Problem
		Contests          []Contest
		ProblemVisibility string
		ContestVisibility string
		ContestTime       string
	}

	data := PageData{
		Title:             "Administration - NoobOJ",
		Pusername:         username,
		Logout:            "Logout",
		Admin:             true,
		Problems:          list,
		Contests:          list2,
		ProblemVisibility: problemVisibility,
		ContestVisibility: contestVisibility,
		ContestTime:       contestTime,
	}
	t.ExecuteTemplate(w, "index.html", data)
}

func DeleteProblemHandler(w http.ResponseWriter, r *http.Request) {
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

	database.DB.Exec("DELETE FROM problems WHERE id=?", id)
	http.Redirect(w, r, "/administration?tab=problem", http.StatusSeeOther)

}
func DeleteContestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request", http.StatusMethodNotAllowed)
		return
	}

	id := r.FormValue("id")
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)

	var author string
	err := database.DB.QueryRow("SELECT author FROM authors WHERE contest_id=? AND role='owner'", id).Scan(&author)
	if err != nil {
		http.Error(w, "Contest not found", http.StatusNotFound)
		return
	}
	if username != author {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	_, err = database.DB.Exec("DELETE FROM contests WHERE id=?", id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func EditProblemHandler(w http.ResponseWriter, r *http.Request) {
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
		_, err = tx.Exec("DELETE FROM ratings WHERE problem_id=?", problemID)
		if err != nil {
			fmt.Fprint(w, "Error deleting rating: "+err.Error())
			return
		}
		_, err = tx.Exec("INSERT INTO ratings(problem_id,rating) VALUES(?,?)", problemID, rating)
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
func EditContestHandler(w http.ResponseWriter, r *http.Request) {
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
			ID int

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

		rows, err = database.DB.Query("SELECT problems.id,problems.title,contest.score FROM problems,(SELECT * FROM tasks WHERE contest_id=?) as contest where problems.id = contest.problem_id ORDER BY serial", id)

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

		type Author struct {
			Username string

			Role string
		}

		var authors []Author

		rows, err = database.DB.Query("SELECT author, role FROM authors WHERE contest_id=?", id)

		if err != nil {

			fmt.Fprint(w, "Error retriving authors: "+err.Error())

			return

		}

		defer rows.Close()

		for rows.Next() {

			var p Author

			if err := rows.Scan(&p.Username, &p.Role); err != nil {

				continue

			}

			authors = append(authors, p)

		}

		currentUserRole := ""

		for _, a := range authors {

			if a.Username == username {

				currentUserRole = a.Role

				break

			}

		}

		// Send raw UTC with Z suffix — browser JS will convert to local time

		data := struct {
			Title string

			Pusername string

			Logout string

			Admin bool

			ID string

			Ptitle string

			Start int64

			End int64

			AvailableProblems []Problem

			SelectedProblems []Problem

			Authors []Author

			CurrentUserRole string
		}{

			Title: "Administration - NoobOJ",

			Pusername: username,

			Logout: "Logout",

			Admin: true,

			ID: id,

			Ptitle: title,

			Start: startTimeUTC.UTC().UnixMilli(),

			End: endTimeUTC.UTC().UnixMilli(),

			AvailableProblems: available,

			SelectedProblems: selected,

			Authors: authors,

			CurrentUserRole: currentUserRole,
		}

		t.ExecuteTemplate(w, "index.html", data)
	}

	if r.Method == "POST" {
		r.ParseForm()
		id := r.URL.Query().Get("id")

		// 1. Fetch existing data to handle disabled fields and validation
		var dbTitle string
		var dbStartBytes, dbEndBytes []byte
		err := database.DB.QueryRow("SELECT title, start_time, end_time FROM contests WHERE id=?", id).Scan(&dbTitle, &dbStartBytes, &dbEndBytes)
		if err != nil {
			fmt.Fprint(w, "Contest not found")
			return
		}

		dbLayout := "2006-01-02 15:04:05"
		oldStartTimeUTC, _ := time.Parse(dbLayout, string(dbStartBytes))

		name := r.FormValue("title")
		startStr := r.FormValue("start") // Will be empty if frontend disabled the input
		endStr := r.FormValue("end")
		tzStr := r.FormValue("timezone")

		ids := r.Form["problem_ids"]
		scores := r.Form["problem_scores"]
		serials := r.Form["problem_serials"]
		authors := r.Form["author_usernames"]
		roles := r.Form["author_roles"]

		if len(ids) != len(scores) || len(ids) != len(serials) {
			fmt.Fprint(w, "Mismatched problem input lengths")
			return
		}

		// 3. Handle Timezone and Parsing
		loc, err := time.LoadLocation(tzStr)
		if err != nil {
			loc = time.UTC
		}
		inputLayout := "2006-01-02T15:04"

		// Handle Start Time: If empty (disabled), use DB value. Otherwise, parse new value.
		var finalStartTime time.Time
		if startStr == "" {
			finalStartTime = oldStartTimeUTC
		} else {
			parsedStart, err := time.ParseInLocation(inputLayout, startStr, loc)
			if err != nil {
				fmt.Fprint(w, "Invalid start time format")
				return
			}
			finalStartTime = parsedStart.UTC()

			// Safety: Double check if they tried to move a start time that already passed
			if time.Now().UTC().After(oldStartTimeUTC) {
				finalStartTime = oldStartTimeUTC // Force original if contest already started
			}
		}

		// Handle End Time: Always required
		finalEndTime, err := time.ParseInLocation(inputLayout, endStr, loc)
		if err != nil {
			fmt.Fprint(w, "Invalid end time format")
			return
		}
		finalEndTime = finalEndTime.UTC()

		// 4. Database Transaction
		tx, err := database.DB.Begin()
		if err != nil {
			http.Error(w, "Transaction error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// Update Contest Header
		_, err = tx.Exec("UPDATE contests SET title = ?, start_time = ?, end_time = ? WHERE id = ?",
			name, finalStartTime, finalEndTime, id)
		if err != nil {
			fmt.Fprint(w, "Error updating contest details: "+err.Error())
			return
		}

		// Sync Tasks (Problems)
		_, err = tx.Exec("DELETE FROM tasks WHERE contest_id=?", id)
		if err != nil {
			fmt.Fprint(w, "Error clearing old tasks: "+err.Error())
			return
		}

		for i := range ids {
			pID, _ := strconv.Atoi(ids[i])
			pScore, _ := strconv.Atoi(scores[i])
			pSerial := serials[i]
			_, err = tx.Exec("INSERT INTO tasks(contest_id, problem_id, score, serial) VALUES(?,?,?,?)",
				id, pID, pScore, pSerial)
			if err != nil {
				fmt.Fprint(w, "Error inserting task: "+err.Error())
				return
			}
		}

		// Sync Authors (Moderators)
		// We delete everyone except the 'owner' to allow role re-assignment/removal
		_, err = tx.Exec("DELETE FROM authors WHERE role<>'owner' AND contest_id=?", id)
		if err != nil {
			fmt.Fprint(w, "Error clearing moderators: "+err.Error())
			return
		}

		for i := range authors {
			uName := authors[i]
			uRole := roles[i]
			// Use UPSERT logic to handle the owner if they were included in the list
			_, err = tx.Exec(`
                INSERT INTO authors (contest_id, author, role) 
                VALUES (?, ?, ?) 
                ON DUPLICATE KEY UPDATE role = VALUES(role)`,
				id, uName, uRole)
			if err != nil {
				continue // Skip failures for individual authors
			}
		}

		// 5. Finalize
		if err = tx.Commit(); err != nil {
			fmt.Fprint(w, "Commit error: "+err.Error())
			return
		}

		http.Redirect(w, r, "/administration?tab=contest", http.StatusSeeOther)
	}
}

func ToggleProblemVisibility(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	id := r.FormValue("id")

	var author string
	err := database.DB.QueryRow("SELECT author FROM problems WHERE id=?", id).Scan(&author)
	if err != nil {
		http.Error(w, "No problem found", http.StatusNotFound)
		return
	}
	if username != author {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	_, err = database.DB.Exec("UPDATE problems SET visibility = NOT visibility WHERE id=?", id)
	if err != nil {
		http.Error(w, "Error updating visibility: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func ToggleContestVisibility(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	id := r.FormValue("id")

	var author string
	err := database.DB.QueryRow("SELECT author FROM authors WHERE contest_id=? AND role='owner'", id).Scan(&author)
	if err != nil {
		http.Error(w, "No contest found", http.StatusNotFound)
		return
	}
	if username != author {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	_, err = database.DB.Exec("UPDATE contests SET visibility = NOT visibility WHERE id=?", id)
	if err != nil {
		http.Error(w, "Error updating visibility: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
