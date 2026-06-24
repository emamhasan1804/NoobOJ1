package handlers

import (
	"NoobOJ/database"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

func ContestsHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var admin bool
	var userType string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&userType)
	if userType == "admin" {
		admin = true
	}

	parsePage := func(raw string) int {
		page, err := strconv.Atoi(raw)
		if err != nil || page < 1 {
			return 1
		}
		return page
	}

	search := strings.TrimSpace(r.URL.Query().Get("q"))
	currentPage := parsePage(r.URL.Query().Get("curr_page"))
	pastPage := parsePage(r.URL.Query().Get("past_page"))

	const currentPageSize = 10
	const pastPageSize = 100

	type Contest struct {
		ID         int
		Title      string
		StartTime  string
		EndTime    string
		Authors    []string
		Registered bool
		Running    bool
	}
	var currentTotal int
	if err := database.DB.QueryRow(`
		SELECT COUNT(*) FROM contests c
		WHERE c.visibility = 1
		AND c.end_time >= UTC_TIMESTAMP()
		AND (c.title LIKE ? OR EXISTS (SELECT 1 FROM authors a WHERE a.contest_id = c.id AND a.author LIKE ?))
	`, "%"+search+"%", "%"+search+"%").Scan(&currentTotal); err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}

	var pastTotal int
	if err := database.DB.QueryRow(`
		SELECT COUNT(*) FROM contests c
		WHERE c.visibility = 1
		AND c.end_time < UTC_TIMESTAMP()
		AND (c.title LIKE ? OR EXISTS (SELECT 1 FROM authors a WHERE a.contest_id = c.id AND a.author LIKE ?))
	`, "%"+search+"%", "%"+search+"%").Scan(&pastTotal); err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}

	rows, err := database.DB.Query(`
		SELECT c.id, c.title, c.start_time, c.end_time, (c.start_time <= UTC_TIMESTAMP()) AS running
		FROM contests c
		WHERE c.visibility = 1
		AND c.end_time >= UTC_TIMESTAMP()
		AND (c.title LIKE ? OR EXISTS (SELECT 1 FROM authors a WHERE a.contest_id = c.id AND a.author LIKE ?))
		ORDER BY c.start_time
		LIMIT ? OFFSET ?
	`, "%"+search+"%", "%"+search+"%", currentPageSize, (currentPage-1)*currentPageSize)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}

	var contests []Contest
	for rows.Next() {
		var p Contest
		p.Running = false
		if err := rows.Scan(&p.ID, &p.Title, &p.StartTime, &p.EndTime, &p.Running); err != nil {
			rows.Close()
			fmt.Fprint(w, "Database error: "+err.Error())
			return
		}
		authorsRows, err := database.DB.Query("SELECT author FROM authors WHERE contest_id=?", p.ID)
		if err != nil {
			rows.Close()
			fmt.Fprint(w, "Database error: "+err.Error())
			return
		}
		for authorsRows.Next() {
			var author string
			if err := authorsRows.Scan(&author); err != nil {
				authorsRows.Close()
				rows.Close()
				fmt.Fprint(w, "Database error: "+err.Error())
				return
			}
			p.Authors = append(p.Authors, author)
		}
		authorsRows.Close()
		if err := database.DB.QueryRow("SELECT EXISTS (SELECT 1 FROM participants WHERE contest_id=? AND participant=?) AS exists_flag", p.ID, username).Scan(&p.Registered); err != nil {
			rows.Close()
			fmt.Fprint(w, "Database error: "+err.Error())
			return
		}
		contests = append(contests, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	rows.Close()

	pastRows, err := database.DB.Query(`
		SELECT c.id, c.title, c.start_time, c.end_time
		FROM contests c
		WHERE c.visibility = 1
		AND c.end_time < UTC_TIMESTAMP()
		AND (c.title LIKE ? OR EXISTS (SELECT 1 FROM authors a WHERE a.contest_id = c.id AND a.author LIKE ?))
		ORDER BY c.end_time DESC
		LIMIT ? OFFSET ?
	`, "%"+search+"%", "%"+search+"%", pastPageSize, (pastPage-1)*pastPageSize)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	defer pastRows.Close()

	var pastcontests []Contest
	for pastRows.Next() {
		var p Contest
		if err := pastRows.Scan(&p.ID, &p.Title, &p.StartTime, &p.EndTime); err != nil {
			fmt.Fprint(w, "Database error: "+err.Error())
			return
		}
		authorsRows, err := database.DB.Query("SELECT author FROM authors WHERE contest_id=?", p.ID)
		if err != nil {
			fmt.Fprint(w, "Database error: "+err.Error())
			return
		}
		for authorsRows.Next() {
			var author string
			if err := authorsRows.Scan(&author); err != nil {
				authorsRows.Close()
				fmt.Fprint(w, "Database error: "+err.Error())
				return
			}
			p.Authors = append(p.Authors, author)
		}
		authorsRows.Close()
		pastcontests = append(pastcontests, p)
	}
	if err := pastRows.Err(); err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
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
		Title           string
		Contests        []Contest
		PastContests    []Contest
		Logout          string
		Pusername       string
		Admin           bool
		Search          string
		CurrentPage     int
		PastPage        int
		NextCurrentPage int
		NextPastPage    int
		HasNextCurrent  bool
		HasNextPast     bool
		CurrentPageSize int
		PastPageSize    int
		CurrentTotal    int
		PastTotal       int
	}

	data := PageData{
		Title:           "Contests - NoobOJ",
		Contests:        contests,
		PastContests:    pastcontests,
		Logout:          "Logout",
		Pusername:       username,
		Admin:           admin,
		Search:          search,
		CurrentPage:     currentPage,
		PastPage:        pastPage,
		NextCurrentPage: currentPage + 1,
		NextPastPage:    pastPage + 1,
		HasNextCurrent:  currentPage*currentPageSize < currentTotal,
		HasNextPast:     pastPage*pastPageSize < pastTotal,
		CurrentPageSize: currentPageSize,
		PastPageSize:    pastPageSize,
		CurrentTotal:    currentTotal,
		PastTotal:       pastTotal,
	}
	if err := t.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ContestRegisterHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		id := r.FormValue("id")
		username := r.FormValue("username")
		var idCheck string
		err := database.DB.QueryRow("SELECT id FROM contests WHERE id=? AND start_time>UTC_TIMESTAMP AND visibility=1", id).Scan(&idCheck)
		if err != nil || idCheck != id {
			fmt.Fprint(w, "no contest found")
			return
		}
		database.DB.Exec("INSERT INTO participants(contest_id, participant) VALUES(?,?)", id, username)
		http.Redirect(w, r, "/contests", http.StatusSeeOther)
	}
}
func ContestUnregisterHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		id := r.FormValue("id")
		username := r.FormValue("username")
		database.DB.Exec("DELETE FROM participants WHERE contest_id=? AND participant=?", id, username)
		http.Redirect(w, r, "/contests", http.StatusSeeOther)
	}
}

func ViewContestHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var admin bool = false
	var user_type string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&user_type)
	if user_type == "admin" {
		admin = true
	}
	id := r.URL.Query().Get("id")

	var name, endtime string
	var isVisible, isStarted, isEnded, isRegistered bool
	err := database.DB.QueryRow(`
    SELECT
        c.title,
        c.end_time,
        c.visibility = 1,
        NOW() >= c.start_time,
        NOW() >= c.end_time,
        EXISTS (
            SELECT 1 FROM participants p
            WHERE p.contest_id = c.id AND p.participant = ?
        )
    FROM contests c
    WHERE c.id = ?
`, username, id).Scan(&name, &endtime, &isVisible, &isStarted, &isEnded, &isRegistered)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	if isVisible == false || isStarted == false {
		fmt.Fprint(w, "No contest found")
		return
	}

	rows, err := database.DB.Query("SELECT serial,problem_id,score FROM tasks WHERE contest_id=? ORDER BY serial", id)
	if err != nil {
		fmt.Fprint(w, "Database baler error: "+err.Error())
		return
	}
	defer rows.Close()

	type Problem struct {
		Serial   string
		ID       int
		Name     string
		Time     int
		Memory   int
		Solved   int
		SolvedBy int
		Score    int
	}
	var problems []Problem
	for rows.Next() {
		var p Problem
		rows.Scan(&p.Serial, &p.ID, &p.Score)
		//
		err = database.DB.QueryRow("SELECT COUNT(DISTINCT username) FROM submissions WHERE problem_id =? AND verdict='Accepted'", p.ID).Scan(&p.SolvedBy)
		if err != nil {
			continue
		}
		var cnt int = 0
		err = database.DB.QueryRow("SELECT COUNT(*) FROM submissions WHERE problem_id=? AND username=? AND verdict='Accepted'", &p.ID, username).Scan(&cnt)
		if err != nil {
			continue
		}
		if cnt > 0 {
			p.Solved = 1
		} else {
			err = database.DB.QueryRow("SELECT COUNT(*) FROM submissions WHERE problem_id=? AND username=?", &p.ID, username).Scan(&cnt)
			if err != nil {
				continue
			}
			if cnt > 0 {
				p.Solved = 2
			} else {
				p.Solved = 0
			}
		}
		err = database.DB.QueryRow("SELECT title,time_limit,memory_limit FROM problems WHERE id=?", p.ID).Scan(&p.Name, &p.Time, &p.Memory)
		if err != nil {
			continue
		}
		problems = append(problems, p)
	}

	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/contest.html",
		"templates/footer.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type PageData struct {
		Title       string
		ID          string
		Pusername   string
		Logout      string
		Admin       bool
		Problems    []Problem
		ContestName string
		EndTime     string
		Ended       bool
	}
	data := PageData{
		Title:       "contest - " + id + " - NoobOJ",
		Pusername:   username,
		Logout:      "Logout",
		Admin:       admin,
		ID:          id,
		Problems:    problems,
		ContestName: name,
		EndTime:     endtime,
		Ended:       isEnded,
	}
	t.ExecuteTemplate(w, "index.html", data)

}

func ContestTaskHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var admin bool = false
	var user_type string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&user_type)
	if user_type == "admin" {
		admin = true
	}
	id := r.URL.Query().Get("id")
	task := r.URL.Query().Get("task")

	var name, endtime string
	var isVisible, isStarted, isRegistered bool
	err := database.DB.QueryRow(`

		SELECT
			c.title,
			c.end_time,
			c.visibility = 1,
			NOW() >= c.start_time,
			EXISTS (
				SELECT 1 FROM participants p
				WHERE p.contest_id = c.id AND p.participant = ?
			)
		FROM contests c
		WHERE c.id = ?

	`, username, id).Scan(&name, &endtime, &isVisible, &isStarted, &isRegistered)

	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}

	if isVisible == false || isStarted == false {
		fmt.Fprint(w, "No contest found")
		return
	}
	var problem_id int
	err = database.DB.QueryRow("SELECT problem_id FROM tasks WHERE contest_id=? AND serial=?", id, task).Scan(&problem_id)
	if err == sql.ErrNoRows {
		fmt.Fprint(w, "No task found")
		return
	} else if err != nil {
		fmt.Fprint(w, "Databse error: "+err.Error())
		return
	}

	row := database.DB.QueryRow("SELECT title, statement,input,output,constraints,author,time_limit,memory_limit FROM problems WHERE id = ?", problem_id)
	var title, statement, input, output, constraints, author, time, memory string
	err = row.Scan(&title, &statement, &input, &output, &constraints, &author, &time, &memory)
	if err != nil {
		fmt.Fprint(w, "Problem not found")
		return
	}
	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/task.html",
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
	rows, err := database.DB.Query("SELECT input,output FROM test_cases WHERE problem_id=? AND type='sample'", problem_id)
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

	rows, err = database.DB.Query("SELECT id,submitted_at,verdict FROM submissions WHERE problem_id=? AND username=? ORDER BY id DESC", problem_id, username)
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
		Subs        []Submissions
		Author      string
		ID          string
		Admin       bool
		One         string
		Serial      string
		Name        string
		EndTime     string
		PID         int
		Registered  bool
	}
	data := PageData{
		Title:       "contest - " + name + " - NoobOJ",
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
		Subs:        subs,
		Author:      author,
		ID:          id,
		Admin:       admin,
		One:         "1",
		Serial:      task,
		Name:        name,
		EndTime:     endtime,
		PID:         problem_id,
		Registered:  isRegistered,
	}
	t.ExecuteTemplate(w, "index.html", data)

}

func ContestStandingsHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var admin bool = false
	var user_type string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&user_type)
	if user_type == "admin" {
		admin = true
	}
	id := r.URL.Query().Get("id")

	var name, endtime string
	var isVisible, isStarted, isRegistered bool
	err := database.DB.QueryRow(`

		SELECT
			c.title,
			c.end_time,
			c.visibility = 1,
			NOW() >= c.start_time,
			EXISTS (
				SELECT 1 FROM participants p
				WHERE p.contest_id = c.id AND p.participant = ?
			)
		FROM contests c
		WHERE c.id = ?

	`, username, id).Scan(&name, &endtime, &isVisible, &isStarted, &isRegistered)

	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}

	if isVisible == false || isStarted == false {
		fmt.Fprint(w, "No contest found")
		return
	}
	type Problem struct {
		Serial string
		ID     int
	}
	var problems []Problem
	rows, err := database.DB.Query("SELECT serial,problem_id FROM tasks where contest_id=? order by serial", id)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	defer rows.Close()
	for rows.Next() {
		var p Problem
		err := rows.Scan(&p.Serial, &p.ID)
		if err != nil {
			continue
		}
		problems = append(problems, p)
	}

	leaderboardRows, err := database.DB.Query(`
WITH contest AS ( SELECT id, start_time, end_time
				FROM contests
				WHERE id = ?
),
first_accepted AS (
    SELECT p.participant, t.problem_id, t.score, MIN(s.submitted_at) AS accepted_at
    FROM participants p
    JOIN contest c ON p.contest_id = c.id
    JOIN tasks t ON t.contest_id = p.contest_id
    JOIN submissions s ON s.problem_id = t.problem_id AND s.username = p.participant AND s.verdict = 'Accepted' AND s.submitted_at BETWEEN c.start_time AND c.end_time
    GROUP BY p.participant, t.problem_id, t.score
),
unsuccessfull_submissions AS (
    SELECT fa.participant, fa.problem_id, fa.score, fa.accepted_at, COUNT(s.id) AS wrong_attempts
    FROM first_accepted fa
    JOIN contest c ON 1 = 1
    JOIN submissions s ON s.problem_id = fa.problem_id AND s.username = fa.participant AND s.verdict != 'Accepted' AND s.submitted_at BETWEEN c.start_time AND fa.accepted_at
    GROUP BY fa.participant, fa.problem_id, fa.score, fa.accepted_at
),
score_per_problem AS (
    -- For every solved problem: points earned + time penalty for that problem
    SELECT fa.participant, fa.problem_id, GREATEST(fa.score / 2, fa.score - COALESCE(w.wrong_attempts, 0) * 50) AS points, TIMESTAMPDIFF(MINUTE, c.start_time, fa.accepted_at) + COALESCE(w.wrong_attempts, 0) * 2                                AS problem_penalty
    FROM first_accepted fa
    JOIN contest c ON 1 = 1
    LEFT JOIN unsuccessfull_submissions w ON w.participant = fa.participant AND w.problem_id  = fa.problem_id
),
standings AS (
    SELECT participant, COUNT(*)              AS solved, SUM(points)           AS total_score, SUM(problem_penalty)  AS total_penalty
    FROM score_per_problem
    GROUP BY participant
)
SELECT
    RANK() OVER (
        ORDER BY total_score DESC, total_penalty ASC
    ) AS rank, participant, solved, total_score      AS score, total_penalty    AS penalty
FROM standings
ORDER BY rank, participant`, id)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	defer leaderboardRows.Close()

	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/standings.html",
		"templates/footer.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type Leaderboard struct {
		Username string
		Rank     int
		Score    string
		Penalty  string
		Solved   []bool
		Count    int
	}

	var list []Leaderboard

	for leaderboardRows.Next() {
		var p Leaderboard

		// rank username solved score penalty
		err := leaderboardRows.Scan(&p.Rank, &p.Username, &p.Count, &p.Score, &p.Penalty)
		if err != nil {
			continue
		}
		for _, problem := range problems {
			flag := false
			database.DB.QueryRow(`
    SELECT EXISTS(
        SELECT 1 FROM submissions
        WHERE problem_id = ? AND username = ? AND verdict = 'Accepted' AND submitted_at BETWEEN  (SELECT start_time FROM contests WHERE id = ?) AND (SELECT end_time FROM contests WHERE id = ?) ) AS solved `, problem.ID, p.Username, id, id).Scan(&flag)
			p.Solved = append(p.Solved, flag)
		}
		list = append(list, p)
	}

	type PageData struct {
		Title     string
		Pusername string
		Logout    string
		ID        string
		Admin     bool
		One       string
		Name      string
		EndTime   string
		List      []Leaderboard
	}
	data := PageData{
		Title:     "contest - " + name + " - NoobOJ",
		Pusername: username,
		Logout:    "Logout",
		ID:        id,
		Admin:     admin,
		One:       "1",
		Name:      name,
		EndTime:   endtime,
		List:      list,
	}
	t.ExecuteTemplate(w, "index.html", data)
}

func ContestStatusHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var admin bool = false
	var user_type string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&user_type)
	if user_type == "admin" {
		admin = true
	}
	id := r.URL.Query().Get("id")

	var name, endtime string
	var isVisible, isStarted, isRegistered bool
	err := database.DB.QueryRow(`

		SELECT
			c.title,
			c.end_time,
			c.visibility = 1,
			NOW() >= c.start_time,
			EXISTS (
				SELECT 1 FROM participants p
				WHERE p.contest_id = c.id AND p.participant = ?
			)
		FROM contests c
		WHERE c.id = ?

	`, username, id).Scan(&name, &endtime, &isVisible, &isStarted, &isRegistered)

	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}

	if isVisible == false || isStarted == false {
		fmt.Fprint(w, "No contest found")
		return
	}

	rows, err := database.DB.Query(`SELECT
    submissions.id,
    submissions.submitted_at,
    submissions.username,
    tasks.serial,
    problems.title,
    submissions.verdict
from contests,tasks,submissions,participants,problems
where contests.id=? and contests.id=tasks.contest_id and tasks.problem_id=submissions.problem_id and submissions.username=participants.participant and participants.contest_id=contests.id and problems.id=submissions.problem_id  order by submissions.id desc;`, id)

	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	defer rows.Close()
	type Status struct {
		ID            int
		Time          string
		Username      string
		ProblemSerial string
		ProblemTitle  string
		Verdict       string
	}
	var list []Status
	for rows.Next() {
		var p Status
		err := rows.Scan(&p.ID, &p.Time, &p.Username, &p.ProblemSerial, &p.ProblemTitle, &p.Verdict)
		if err != nil {
			continue
		}
		list = append(list, p)
	}

	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/status.html",
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
		ID        string
		Admin     bool
		One       string
		Name      string
		EndTime   string
		List      []Status
	}
	data := PageData{
		Title:     "contest - " + name + " - NoobOJ",
		Pusername: username,
		Logout:    "Logout",
		ID:        id,
		Admin:     admin,
		One:       "1",
		Name:      name,
		EndTime:   endtime,
		List:      list,
	}
	t.ExecuteTemplate(w, "index.html", data)
}

func ContestMySubmissionsHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var admin bool = false
	var user_type string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&user_type)
	if user_type == "admin" {
		admin = true
	}
	id := r.URL.Query().Get("id")

	var name, endtime string
	var isVisible, isStarted, isRegistered bool
	err := database.DB.QueryRow(`

		SELECT
			c.title,
			c.end_time,
			c.visibility = 1,
			NOW() >= c.start_time,
			EXISTS (
				SELECT 1 FROM participants p
				WHERE p.contest_id = c.id AND p.participant = ?
			)
		FROM contests c
		WHERE c.id = ?

	`, username, id).Scan(&name, &endtime, &isVisible, &isStarted, &isRegistered)

	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}

	if isVisible == false || isStarted == false {
		fmt.Fprint(w, "No contest found")
		return
	}

	rows, err := database.DB.Query(`SELECT
    s.id,
    s.submitted_at,
    s.username,
    t.serial,
    p.title,
    s.verdict
FROM submissions s
JOIN tasks t
    ON s.problem_id = t.problem_id
JOIN problems p
    ON s.problem_id = p.id
JOIN contests c
    ON c.id = t.contest_id
WHERE t.contest_id = ?
   AND s.username=?
ORDER BY s.id DESC;`, id, username)

	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	defer rows.Close()
	type Status struct {
		ID            int
		Time          string
		Username      string
		ProblemSerial string
		ProblemTitle  string
		Verdict       string
	}
	var list []Status
	for rows.Next() {
		var p Status
		err := rows.Scan(&p.ID, &p.Time, &p.Username, &p.ProblemSerial, &p.ProblemTitle, &p.Verdict)
		if err != nil {
			continue
		}
		list = append(list, p)
	}

	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/mysubmissions.html",
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
		ID        string
		Admin     bool
		One       string
		Name      string
		EndTime   string
		List      []Status
	}
	data := PageData{
		Title:     "contest - " + name + " - NoobOJ",
		Pusername: username,
		Logout:    "Logout",
		ID:        id,
		Admin:     admin,
		One:       "1",
		Name:      name,
		EndTime:   endtime,
		List:      list,
	}
	t.ExecuteTemplate(w, "index.html", data)
}
