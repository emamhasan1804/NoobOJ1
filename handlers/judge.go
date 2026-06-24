package handlers

import (
	"NoobOJ/database"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var submissionQueue = make(chan int, 100) // buffered queue go channel

func SubmitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	problemID := r.FormValue("problem_id")
	code := r.FormValue("code")
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	contest := r.FormValue("contest")
	ID := r.FormValue("ID")
	if contest == "true" {
		var registered bool
		err := database.DB.QueryRow("SELECT EXISTS (SELECT * from participants where participant=? and contest_id=?);", username, ID).Scan(&registered)
		if err != nil {
			return
		}
		if registered == false {
			return
		}
	}
	res, err := database.DB.Exec("INSERT INTO submissions(problem_id, username, code,submitted_at) VALUES (?, ?, ?,UTC_TIMESTAMP())", problemID, username, code)
	if err != nil {
		fmt.Fprint(w, "Database error: "+err.Error())
		return
	}

	subID64, _ := res.LastInsertId()
	subID := int(subID64)

	// pushing into the queue
	submissionQueue <- subID
	if contest == "true" {
		http.Redirect(w, r, "/contest/mysubmissions?id="+ID, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/submission?id="+strconv.Itoa(subID), http.StatusSeeOther)
	}

}

func StartJudgeWorkers(count int) { // goroutine
	for i := 0; i < count; i++ {
		go func(workerID int) {
			for subID := range submissionQueue {
				ProcessSubmission(subID)
			}
		}(i) // annono
	}
}

func ProcessSubmission(subID int) {
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
