package handlers

import (
	"NoobOJ/database"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

func Home(w http.ResponseWriter, r *http.Request) {
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
func LeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	var userType string
	database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&userType)
	admin := userType == "admin"

	rows, err := database.DB.Query(`SELECT u.username, u.name,
		COUNT(DISTINCT CASE WHEN s.verdict='Accepted' AND p.visibility=1 THEN s.problem_id END) AS cnt
		FROM users u
		LEFT JOIN submissions s ON u.username=s.username
		LEFT JOIN problems p ON s.problem_id=p.id
		GROUP BY u.username, u.name`)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}
	defer rows.Close()

	type Leaderboard struct {
		Username string
		Name     string
		Count    int
		Rank     int
	}

	var list []Leaderboard
	for rows.Next() {
		var p Leaderboard
		if err := rows.Scan(&p.Username, &p.Name, &p.Count); err != nil {
			continue
		}
		list = append(list, p)
	}

	ranked := append([]Leaderboard(nil), list...)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Count == ranked[j].Count {
			return strings.ToLower(ranked[i].Username) < strings.ToLower(ranked[j].Username)
		}
		return ranked[i].Count > ranked[j].Count
	})
	rankByUsername := make(map[string]int, len(ranked))
	for i, item := range ranked {
		rankByUsername[item.Username] = i + 1
	}

	search := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))
	if sortBy == "" {
		sortBy = "solved_desc"
	}
	page, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("page")))
	if page < 1 {
		page = 1
	}

	filtered := make([]Leaderboard, 0, len(list))
	for _, p := range list {
		if search == "" || strings.Contains(strings.ToLower(p.Username), search) || strings.Contains(strings.ToLower(p.Name), search) {
			p.Rank = rankByUsername[p.Username]
			filtered = append(filtered, p)
		}
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		a, b := filtered[i], filtered[j]
		switch sortBy {
		case "solved_asc":
			if a.Count == b.Count {
				return strings.ToLower(a.Username) < strings.ToLower(b.Username)
			}
			return a.Count < b.Count
		case "username_asc":
			return strings.ToLower(a.Username) < strings.ToLower(b.Username)
		case "username_desc":
			return strings.ToLower(a.Username) > strings.ToLower(b.Username)
		case "name_asc":
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		case "name_desc":
			return strings.ToLower(a.Name) > strings.ToLower(b.Name)
		default:
			if a.Count == b.Count {
				return strings.ToLower(a.Username) < strings.ToLower(b.Username)
			}
			return a.Count > b.Count
		}
	})

	const pageSize = 50
	total := len(filtered)
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		page = 1
	} else if page > totalPages {
		page = totalPages
	}

	pageItems := []Leaderboard{}
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
		"templates/leaderboard.html",
		"templates/footer.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type PageData struct {
		Title      string
		Items      []Leaderboard
		Pusername  string
		Logout     string
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
		MyRankPage int
	}

	myRankPage := 0
	if rank := rankByUsername[username]; rank > 0 {
		myRankPage = (rank + pageSize - 1) / pageSize
	}

	data := PageData{
		Title:      "Leaderboard - NoobOJ",
		Items:      pageItems,
		Pusername:  username,
		Logout:     "Logout",
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
		MyRankPage: myRankPage,
	}
	t.ExecuteTemplate(w, "index.html", data)
}
func ProfileHandler(w http.ResponseWriter, r *http.Request) {
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
