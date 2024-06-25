package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	redis "github.com/go-redis/redis/v8"
	redis1 "github.com/gomodule/redigo/redis"
)

type Evaluation struct {
	INTVName string
}

type EvaluateResult struct {
	Name string `json:"name"`
	Result string `json:"result"`
}

type ListcandTempl struct {
	INTVName string
	Candidates []Candidate
}

type Candidate struct {
	CandName string
	Result string
}

type User struct {
	Uname string `json:"uname"`
	Pass  string `json:"pass"`
}

var client = redis.NewClient(&redis.Options{
	Addr: "redis://red-cpt4b96ehbks73etudn0:6379",
})

func main() {
	tmplSignUp := template.Must(template.ParseFiles("templates/signup.html"))
	http.HandleFunc("/sign-up", func(w http.ResponseWriter, r *http.Request) {
		

		tmplSignUp.Execute(w, nil)
	})
	tmplLogin := template.Must(template.ParseFiles("templates/login.html"))
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("uname")

		if err != nil ||  cookie==nil {
			log.Printf("%+v",err)
			
		} 
		log.Println("Crossed nil")
		if cookie!=nil && cookie.Value!="" {
			http.Redirect(w, r, "/dashboard", http.StatusFound)
		}
		tmplLogin.Execute(w, nil)
	})
	http.HandleFunc("/sign-up-redirect", func(w http.ResponseWriter, r *http.Request) {
		uname, pass := r.FormValue("uname"), r.FormValue("pass")
		fmt.Println(uname, pass)
		if uname == "" || pass == "" {
			http.Redirect(w, r, "/sign-up", http.StatusFound)
		}
		ctx := context.Background()
		user := User{
			Uname: uname,
			Pass:  pass,
		}
		jsonBody, err := json.Marshal(user)
		if err != nil {
			panic(err.Error())
		}
		err = client.Set(ctx, "user-cred-"+uname, jsonBody, 0).Err()
		if err != nil {
			panic(err.Error())
		}

		http.Redirect(w, r, "/login", http.StatusFound)

	})
	http.HandleFunc("/login-redirect", func(w http.ResponseWriter, r *http.Request) {
		if r==nil {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
		uname, pass := r.FormValue("uname"), r.FormValue("pass")
		cookie, err := r.Cookie("uname")
		if err != nil || cookie==nil {
			log.Printf("%+v",err)

		}
		log.Println("Crossed nil redirect")
		
		if cookie!=nil && cookie.Value!="" {
			http.Redirect(w, r, "/dashboard", http.StatusFound)
		}
		fmt.Println(uname, uname, pass)
		if uname == "" || pass == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
		ctx := context.Background()

		val, _ := client.Get(ctx, "user-cred-"+uname).Result()
		log.Printf("%+v\n", val)
		var user User
		json.Unmarshal([]byte(val), &user)
		log.Printf("%+v %s\n", user,pass)
		if user.Pass == pass {
			log.Printf("Equal : %+v",user)
			http.SetCookie(w, &http.Cookie{
				Name:  "uname",
				Value: uname,
				MaxAge: 1800,
				Secure: true,
			})
			//time.Sleep(time.Second*10)

			http.Redirect(w, r, "/dashboard", http.StatusFound)
			return
		} else {
			http.Redirect(w, r, "/login", http.StatusNotFound)
		}

	})

	tmplDashboard := template.Must(template.ParseFiles("templates/dashboard.html"))
	http.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {

		log.Println("Entering dashboard")
		cookie, err := r.Cookie("uname")

		log.Println("Entering dashboard")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
		log.Println("Entering dashboard")
		if cookie==nil  || cookie.Value==""  {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
		if cookie!=nil {
		tmplDashboard.Execute(w, Evaluation{
			INTVName: cookie.Value,
		})
	}
	})
	http.HandleFunc("/submitResult", func(w http.ResponseWriter, r *http.Request) {
		if r.Method=="POST" {
			cookie, err := r.Cookie("uname")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
			log.Printf("%+v",r.Body)
			var evalRes EvaluateResult
			json.NewDecoder(r.Body).Decode(&evalRes)
			log.Printf("%+v",evalRes)
			name:=strings.TrimSpace(strings.Split(evalRes.Name,"-")[0])
			ctx:=context.Background()
			log.Printf("%s %s\n",name,evalRes.Result)
			err = client.Set(ctx, "cand-res-"+name, evalRes.Result, 0).Err()
			err = client.Set(ctx, "cand-res-"+cookie.Value+"-"+name, evalRes.Result, 0).Err()
		if err != nil {
			panic(err.Error())
		}
		fmt.Fprintf(w,"Succesfully recorded")
		} else {
		fmt.Fprintf(w,"Not Succesfully recorded")
		}
	})
	tmplCands:=template.Must(template.ParseFiles("templates/ListCands.html"))
	http.HandleFunc("/ListCands", func(w http.ResponseWriter, r *http.Request) {
	    conn, err := redis1.Dial("tcp", "localhost:6379")
		cookie, err := r.Cookie("uname")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	keys, err := redis1.Strings(conn.Do("KEYS", "*"))
	if err != nil {
		fmt.Println(err)
		return
	}
	var cands []Candidate
	ctx:=context.Background()
	for _, key := range keys {
		 if strings.Index(key,"cand-res-"+cookie.Value)!=-1 {
			val, _ := client.Get(ctx, key).Result()
			cands = append(cands,Candidate{
				CandName: strings.Split(key,"-")[3],
				Result: val,
			})
		 }
	}
	tmplCands.Execute(w,ListcandTempl{
		INTVName: cookie.Value,
		Candidates: cands,
	})

	})
	log.Println("Starting")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
