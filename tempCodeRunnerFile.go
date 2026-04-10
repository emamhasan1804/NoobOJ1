func problemsetHandler(w http.ResponseWriter, r *http.Request) {
// 	// session, _ := store.Get(r, "session")
// 	// username, _ := session.Values["username"].(string)
// 	// var user_type string
// 	// database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", username).Scan(&user_type)
// 	// var admin bool = false
// 	// if user_type == "admin" {
// 	// 	admin = true
// 	// }
// 	// rows, err := database.DB.Query("SELECT id, title FROM problems where visibility=1")
// 	// if err != nil {
// 	// 	fmt.Fprint(w, "Database error: "+err.Error())
// 	// 	return
// 	// }
// 	// defer rows.Close()

// 	// type Problem struct {
// 	// 	ID     int
// 	// 	Title  string
// 	// 	Solved int
// 	// 	Tags   []string
// 	// 	Rating string
// 	// 	Tried  int
// 	// }
// 	// var list []Problem
// 	// for rows.Next() {
// 	// 	var p Problem
// 	// 	rows.Scan(&p.ID, &p.Title)
// 	// 	database.DB.QueryRow("SELECT COUNT(DISTINCT username) FROM submissions WHERE problem_id=? AND verdict='Accepted'", p.ID).Scan(&p.Solved)
// 	// 	database.DB.QueryRow("SELECT rating FROM ratings WHERE problem_id=? LIMIT 1", p.ID).Scan(&p.Rating)
// 	// 	tags, err := database.DB.Query("SELECT name FROM tags,problem_tags WHERE problem_id=? AND tags.id=problem_tags.tag_id", p.ID)
// 	// 	if err != nil {
// 	// 		fmt.Fprint(w, "Database error: "+err.Error())
// 	// 		return
// 	// 	}

//		// 	for tags.Next() {
//		// 		temp := ""
//		// 		tags.Scan(&temp)
//		// 		p.Tags = append(p.Tags, temp)
//		// 	}
//		// 	tags.Close()
//		// 	p.Tried = 0
//		// 	var cnt int
//		// 	err = database.DB.QueryRow("SELECT COUNT(*) FROM submissions WHERE problem_id=? AND username=?", p.ID, username).Scan(&cnt)
//		// 	if err == nil && cnt > 0 {
//		// 		p.Tried = 1
//		// 	}
//		// 	err = database.DB.QueryRow("SELECT COUNT(*) FROM submissions WHERE problem_id=? AND username=? AND verdict='Accepted'", p.ID, username).Scan(&cnt)
//		// 	if err == nil && cnt > 0 {
//		// 		p.Tried = 2
//		// 	}
//		// 	list = append(list, p)
//		// }
//		// t, err := template.ParseFiles(
//		// 	"templates/index.html",
//		// 	"templates/problemset.html",
//		// 	"templates/footer.html",
//		// )
//		// if err != nil {
//		// 	http.Error(w, err.Error(), http.StatusInternalServerError)
//		// 	return
//		// }
//		// type PageData struct {
//		// 	Title     string
//		// 	Problems  []Problem
//		// 	Logout    string
//		// 	Pusername string
//		// 	Admin     bool
//		// }
//		// data := PageData{
//		// 	Title:     "Problemset - NoobOJ",
//		// 	Problems:  list,
//		// 	Logout:    "Logout",
//		// 	Pusername: username,
//		// 	Admin:     admin,
//		// }
//		// t.ExecuteTemplate(w, "index.html", data)
//		fmt.Fprint(w, "bal")
//	}