package handlers

import (
	"NoobOJ/database"
	"crypto/rand"
	"database/sql"
	"fmt"
	"html/template"
	"math/big"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

var store = sessions.NewCookieStore([]byte("super-secret-key"))

const otpTimeLimit = 10 * time.Minute
const maxOTPAttempts = 5

type authPageData struct {
	Title     string
	Pusername string
	Logout    string
	Admin     bool
	Error     string
	Message   string
	Email     string
}

func renderAuthTemplate(w http.ResponseWriter, page string, data authPageData) {
	t, err := template.ParseFiles(
		"templates/index.html",
		"templates/"+page,
		"templates/footer.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.ExecuteTemplate(w, "index.html", data)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderAuthTemplate(w, "register.html", authPageData{
			Title:     "Register - NoobOJ",
			Pusername: "Login",
			Logout:    "Register",
		})
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	username := strings.TrimSpace(r.FormValue("username"))
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")
	confirm := r.FormValue("confirm_password")

	if name == "" || username == "" || email == "" || password == "" {
		renderRegisterError(w, "Please fill in all fields.")
		return
	}
	if password != confirm {
		renderRegisterError(w, "Passwords do not match.")
		return
	}
	if !validEmail(email) {
		renderRegisterError(w, "Please enter a valid email address.")
		return
	}
	if len(username) < 5 || len(username) > 20 {
		renderRegisterError(w, "Username must be between 5 and 20 characters.")
		return
	}

	var exists int
	if err := database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username=? OR email=?", username, email).Scan(&exists); err != nil {
		renderRegisterError(w, "Database error: "+err.Error())
		return
	}
	if exists > 0 {
		renderRegisterError(w, "Username or email already exists.")
		return
	}

	otp, err := generateOTP()
	if err != nil {
		renderRegisterError(w, "Could not generate OTP. Please try again.")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		renderRegisterError(w, "Could not prepare your password. Please try again.")
		return
	}
	otpHash, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		renderRegisterError(w, "Could not prepare OTP. Please try again.")
		return
	}

	expiresAt := time.Now().UTC().Add(otpTimeLimit)
	if _, err := database.DB.Exec("DELETE FROM registration_otps WHERE expires_at <= UTC_TIMESTAMP() OR username=? OR email=?", username, email); err != nil {
		renderRegisterError(w, "Database error: "+err.Error())
		return
	}
	_, err = database.DB.Exec(`INSERT INTO registration_otps(username, name, email, password_hash, otp_hash, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)`, username, name, email, string(hash), string(otpHash), expiresAt.Format("2006-01-02 15:04:05"))
	if err != nil {
		renderRegisterError(w, "Database error: "+err.Error())
		return
	}

	if err := Email(email, "Your NoobOJ verification code", verificationEmailHTML(name, otp)); err != nil {
		database.DB.Exec("DELETE FROM registration_otps WHERE email=?", email)
		renderRegisterError(w, "Could not send OTP email. Please check the email address and try again.")
		return
	}

	http.Redirect(w, r, "/verify-otp?email="+url.QueryEscape(email)+"&sent=1", http.StatusSeeOther)
}

func renderRegisterError(w http.ResponseWriter, message string) {
	renderAuthTemplate(w, "register.html", authPageData{
		Title:     "Register - NoobOJ",
		Pusername: "Login",
		Logout:    "Register",
		Error:     message,
	})
}

func VerifyOTPHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		message := "Enter the OTP sent to your email. It expires 10 minutes after registration."
		if r.URL.Query().Get("sent") == "1" {
			message = "We sent a 6-digit OTP to your email: " + strings.TrimSpace(strings.ToLower(r.URL.Query().Get("email"))) + ". Check your mail's spam box."
		}
		renderAuthTemplate(w, "verify_otp.html", authPageData{
			Title:     "Verify email - NoobOJ",
			Pusername: "Login",
			Logout:    "Register",
			Email:     strings.TrimSpace(strings.ToLower(r.URL.Query().Get("email"))),
			Message:   message,
		})
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	otp := strings.TrimSpace(r.FormValue("otp"))
	if email == "" || otp == "" {
		renderVerifyError(w, email, "Email and OTP are required.")
		return
	}
	if !validEmail(email) {
		renderVerifyError(w, email, "Please enter a valid email address.")
		return
	}
	if !validOTP(otp) {
		renderVerifyError(w, email, "OTP must be exactly 6 digits.")
		return
	}

	var username, name, passwordHash, otpHash string
	var attempts int
	var expired bool
	err := database.DB.QueryRow(`SELECT username, name, password_hash, otp_hash, attempts, expires_at <= UTC_TIMESTAMP()
		FROM registration_otps WHERE email=?`, email).Scan(&username, &name, &passwordHash, &otpHash, &attempts, &expired)
	if err == sql.ErrNoRows {
		renderVerifyError(w, email, "No pending registration found. Please register again.")
		return
	}
	if err != nil {
		renderVerifyError(w, email, "Database error: "+err.Error())
		return
	}
	if expired {
		database.DB.Exec("DELETE FROM registration_otps WHERE email=?", email)
		renderVerifyError(w, email, "OTP expired. Please register again.")
		return
	}
	if attempts >= maxOTPAttempts {
		database.DB.Exec("DELETE FROM registration_otps WHERE email=?", email)
		renderVerifyError(w, email, "Too many wrong attempts. Please register again.")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(otpHash), []byte(otp)) != nil {
		if attempts+1 >= maxOTPAttempts {
			database.DB.Exec("DELETE FROM registration_otps WHERE email=?", email)
			renderVerifyError(w, email, "Too many wrong attempts. Please register again.")
			return
		}
		database.DB.Exec("UPDATE registration_otps SET attempts=attempts+1 WHERE email=?", email)
		renderVerifyError(w, email, "Invalid OTP. Please check the code and try again.")
		return
	}

	var exists int
	if err := database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username=? OR email=?", username, email).Scan(&exists); err != nil {
		renderVerifyError(w, email, "Database error: "+err.Error())
		return
	}
	if exists > 0 {
		database.DB.Exec("DELETE FROM registration_otps WHERE email=?", email)
		renderVerifyError(w, email, "Username or email already exists. Please log in.")
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		renderVerifyError(w, email, "Database error: "+err.Error())
		return
	}
	_, err = tx.Exec("INSERT INTO users(name, username, email, password, user_type) VALUES (?, ?, ?, ?, 'user')", name, username, email, passwordHash)
	if err == nil {
		_, err = tx.Exec("DELETE FROM registration_otps WHERE email=?", email)
	}
	if err != nil {
		tx.Rollback()
		renderVerifyError(w, email, "Database error: "+err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		renderVerifyError(w, email, "Database error: "+err.Error())
		return
	}

	session, _ := store.Get(r, "session")
	session.Values["username"] = username
	session.Values["user_type"] = "user"
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func renderVerifyError(w http.ResponseWriter, email, message string) {
	renderAuthTemplate(w, "verify_otp.html", authPageData{
		Title:     "Verify email - NoobOJ",
		Pusername: "Login",
		Logout:    "Register",
		Email:     email,
		Error:     message,
	})
}

func generateOTP() (string, error) {
	max := big.NewInt(900000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}

func validEmail(email string) bool {
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email
}

func validOTP(otp string) bool {
	if len(otp) != 6 {
		return false
	}
	for _, ch := range otp {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func verificationEmailHTML(name, otp string) string {
	return fmt.Sprintf(`
<div style="margin:0;padding:24px;background:#f4f7fb;font-family:Arial,Helvetica,sans-serif;color:#162335;">
  <div style="max-width:520px;margin:0 auto;background:#ffffff;border:1px solid #d7dee9;border-radius:14px;overflow:hidden;box-shadow:0 8px 24px rgba(34,53,80,0.08);">
    <div style="padding:22px 24px;border-bottom:1px solid #e3e9f2;background:#fbfdff;">
      <h1 style="margin:0;font-size:22px;color:#162335;">NoobOJ email verification</h1>
      <p style="margin:8px 0 0;color:#4e5d73;font-size:14px;">Hi %s, use this OTP to finish creating your account.</p>
    </div>
    <div style="padding:24px;text-align:center;">
      <div style="display:inline-block;letter-spacing:8px;font-size:34px;font-weight:800;color:#1f4e7a;background:#edf4fb;border:1px solid #c7d8ec;border-radius:12px;padding:14px 18px;">%s</div>
      <p style="margin:18px 0 0;color:#4e5d73;font-size:14px;line-height:1.5;">This code expires in <strong>10 minutes</strong>. If you did not request this, you can ignore this email.</p>
    </div>
  </div>
</div>`, template.HTMLEscapeString(name), otp)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderAuthTemplate(w, "login.html", authPageData{
			Title:     "Login - NoobOJ",
			Pusername: "Login",
			Logout:    "Register",
		})
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	var username string
	var hash string
	var userType string

	err := database.DB.QueryRow("SELECT  username, password, user_type FROM users WHERE email=?", email).
		Scan(&username, &hash, &userType)
	if err == sql.ErrNoRows {
		renderLoginError(w, "Email not registered.")
		return
	} else if err != nil {
		renderLoginError(w, "Database error: "+err.Error())
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		renderLoginError(w, "Incorrect password.")
		return
	}

	session, _ := store.Get(r, "session")
	session.Values["username"] = username
	session.Values["user_type"] = userType
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func renderLoginError(w http.ResponseWriter, message string) {
	renderAuthTemplate(w, "login.html", authPageData{
		Title:     "Login - NoobOJ",
		Pusername: "Login",
		Logout:    "Register",
		Error:     message,
	})
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	if session.Values["username"] == nil {
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func Email(to, subject, body string) error {
	from := "support.nooboj@gmail.com"
	appPassword := "cdnigpnmdrpjducm"

	message := []byte(
		"To: " + to + "\r\n" +
			"From: NoobOJ <" + from + ">\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"\r\n" +
			body,
	)

	auth := smtp.PlainAuth(
		"",
		from,
		appPassword,
		"smtp.gmail.com",
	)

	err := smtp.SendMail(
		"smtp.gmail.com:587",
		auth,
		from,
		[]string{to},
		message,
	)

	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
