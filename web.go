package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var db *sql.DB
var err error

type User struct {
	userId   string
	userLv   int
	userName string
	Password string
}

func home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, "index.html")
}

// login은 /login 주소의 로그인 화면과 폼 전송을 처리합니다.
func login(w http.ResponseWriter, r *http.Request) {
	_, err := r.Cookie("session")
	switch r.Method {
	case http.MethodGet:
		if err == http.ErrNoCookie {
			http.ServeFile(w, r, "login.html")
			return
		} else if err != http.ErrNoCookie {
			http.ServeFile(w, r, "index.html")
			return
		}
	case http.MethodPost:
		user := User{
			userName: r.FormValue("user_name"),
			Password: r.FormValue("password"),
		}
		row := db.QueryRow(
			"Select user_lv, user_id, user_name, password from login where user_id = $1",
			user.userName,
		)

		var userId string
		var userLv int
		var userName string
		var Password string

		err := row.Scan(&userLv, &userId, &userName, &Password)
		if err == sql.ErrNoRows {
			http.ServeFile(w, r, "login.html")
			return
		} else if err != nil {
			log.Fatal(err)
		}

		if user.userName == userName && user.Password == Password {
			cookie := http.Cookie{
				Name:     "session",
				Value:    "hello",
				Path:     "/",
				MaxAge:   3600,
				HttpOnly: true, // xss로 쿠키 탈취 막기.
			}
			w.Header().Set("Set-Cookie", cookie.String())

			http.ServeFile(w, r, "index.html")
			return
		} else {
			http.ServeFile(w, r, "login.html")
		}

		// 학습용 예시입니다. 실제 서비스에서는 비밀번호를 코드에 저장하지 말고,
		// 데이터베이스의 해시된 비밀번호와 비교해야 합니다.

	default:
		http.Error(w, "허용되지 않은 메서드입니다.", http.StatusMethodNotAllowed)
	}
}

func API(w http.ResponseWriter, r *http.Request) {

}

func signup(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		http.ServeFile(w, r, "sign.html")
	case http.MethodPost:
		user := r.FormValue("sign_name")
		password := r.FormValue("sign_password")

		_, err := db.Exec(
			"INSERT INTO login (user_id, password, user_lv, user_name) VALUES ($1, $2, 0, $1)",
			user, password,
		)

		var target *pgconn.PgError

		if err != nil {
			if errors.As(err, &target) && target.Code == "23505" {
				http.ServeFile(w, r, "sign.html")
				return
			}
			log.Fatal(err)
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)

	default:
		http.Error(w, "허용되지 않은 메서드입니다.", http.StatusMethodNotAllowed)
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	cookie := http.Cookie{
		Name:     "session",
		Value:    "hello",
		HttpOnly: true,
		MaxAge:   -1,  // 쿠키를 즉시 만료(삭제)시킨다.
		Path:     "/", // 경로와 이름을 같게 해야 전에 받았던 쿠키가 삭제된다.
	}
	w.Header().Set("Set-Cookie", cookie.String())
	http.ServeFile(w, r, "login.html")
}

func main() {
	mux := http.NewServeMux()
	dbpassword := os.Getenv("DB_PASSWORD")
	db, err = sql.Open("pgx", "postgres://postgres:"+dbpassword+"@localhost:5432/postgres")
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("PostgreSQL 연결 성공")

	// 각 페이지의 주소를 명확히 분리합니다.
	mux.HandleFunc("/", home)
	mux.HandleFunc("/login", login)
	mux.HandleFunc("/sign", signup)
	mux.HandleFunc("/logout", logout)
	mux.HandleFunc("/security.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "security.html")
	})
	mux.HandleFunc("/web.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web.html")
	})
	mux.HandleFunc("/sign.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "sign.html")
	})

	// 예전에 사용한 /login.html 주소로 와도 /login으로 이동시킵니다.
	mux.HandleFunc("/login.html", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
	mux.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("images"))))

	log.Println("서버 실행: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
