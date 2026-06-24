package main

import (
	"NoobOJ/database"
	"NoobOJ/handlers"
	"fmt"
	"net/http"
)

func main() {
	handlers.StartJudgeWorkers(2)
	fmt.Println("Starting server...")
	database.Connect()
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	// public pages
	http.HandleFunc("/", handlers.Home)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/verify-otp", handlers.VerifyOTPHandler)
	// admin pages
	http.HandleFunc("/new/problem", handlers.RequiredAdmin(handlers.NewProblemHandler))
	http.HandleFunc("/new/contest", handlers.RequiredAdmin(handlers.NewContestHandler))
	http.HandleFunc("/toggle/problem", handlers.RequiredAdmin(handlers.ToggleProblemVisibility))
	http.HandleFunc("/toggle/contest", handlers.RequiredAdmin(handlers.ToggleContestVisibility))
	http.HandleFunc("/administration", handlers.RequiredAdmin(handlers.AdministrationHandler))
	http.HandleFunc("/delete/problem", handlers.RequiredAdmin(handlers.DeleteProblemHandler))
	http.HandleFunc("/delete/contest", handlers.RequiredAdmin(handlers.DeleteContestHandler))
	http.HandleFunc("/edit/problem", handlers.RequiredAdmin(handlers.EditProblemHandler))
	http.HandleFunc("/edit/contest", handlers.RequiredAdmin(handlers.EditContestHandler))
	// user pages
	http.HandleFunc("/logout", handlers.LogoutHandler)
	http.HandleFunc("/problemset", handlers.ProblemsetHandler)
	http.HandleFunc("/problem", handlers.ViewProblemHandler)
	http.HandleFunc("/editorial", handlers.EditorialHandler)
	http.HandleFunc("/submission", handlers.ViewSubmissionHandler)
	http.HandleFunc("/submissions/problem", handlers.ProblemSubmissionsHandler)
	http.HandleFunc("/submissions/user", handlers.UserSubmissionsHandler)
	http.HandleFunc("/submit", handlers.RequiredLogin(handlers.SubmitHandler))
	http.HandleFunc("/profile", handlers.ProfileHandler)
	http.HandleFunc("/leaderboard", handlers.LeaderboardHandler)
	http.HandleFunc("/contests", handlers.ContestsHandler)
	http.HandleFunc("/register/contest", handlers.RequiredLogin(handlers.ContestRegisterHandler))
	http.HandleFunc("/unregister/contest", handlers.RequiredLogin(handlers.ContestUnregisterHandler))
	http.HandleFunc("/contest/task", handlers.ContestTaskHandler)
	http.HandleFunc("/contest/standings", handlers.ContestStandingsHandler)
	http.HandleFunc("/contest/status", handlers.ContestStatusHandler)
	http.HandleFunc("/contest/mysubmissions", handlers.RequiredLogin(handlers.ContestMySubmissionsHandler))
	http.HandleFunc("/contest", handlers.ViewContestHandler)
	http.ListenAndServe("0.0.0.0:8080", nil)
}
