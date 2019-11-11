package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	_ "net/url"
	"strconv"
	"strings"
	"text/template"

	rice "github.com/GeertJohan/go.rice"
	_ "github.com/lib/pq"
	"github.com/tidwall/gjson"
)

var db *sql.DB

type geocode struct {
	Id      int
	Address string
	X       float64
	Y       float64
}

// Ключ Яндекс API
const APIKey = "023f7c73-40fd-4d88-8ab6-9bc2fde16a08"

func init() {
	var login, password, host, port string
	fmt.Println("Вход в PostrgreSQL")
	fmt.Println("Введите логин:")
	fmt.Scan(&login)
	fmt.Println("Введите пароль:")
	fmt.Scan(&password)
	fmt.Println("Введите хост")
	fmt.Scan(&host)
	fmt.Println("Введите порт")
	fmt.Scan(&port)
	sqllogin := "user=" + login + " password=" + password + " sslmode=disable host=" + host + " port=" + port
	fmt.Println(sqllogin)
	var err error
	db, err = sql.Open("postgres", sqllogin)
	if err != nil {
		log.Fatal(err)
	}
	result, err := db.Exec("Create Database geolocation;")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Создана новая база данных geolocation", result.RowsAffected)
	}
	db, err = sql.Open("postgres", sqllogin+" dbname=geolocation")
	if err != nil {
		log.Fatal(err)
	}
	result, err = db.Exec("drop table if exists geocode;")
	if err != nil {
		fmt.Println(err)
	}
	result, err = db.Exec("create table if not exists geocode(id serial,address varchar(50),X double precision,Y double precision,primary key (id));")
	if err != nil {
		fmt.Println(err)
	}
}

type vec2 struct {
	x float64
	y float64
}

func index(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmplBox := rice.MustFindBox("StaticWeb")
		tmplString, _ := tmplBox.String("html/index.html")
		tmpl, _ := template.New("index").Parse(tmplString)
		tmpl.Execute(w, nil)
	}
}

func list(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("Select distinct on (address,x,y) * from geocode;")
	if err != nil {
		fmt.Println(err)
	}

	var geocodes []geocode
	for rows.Next() {
		obj := new(geocode)
		err := rows.Scan(&obj.Id, &obj.Address, &obj.X, &obj.Y)
		if err != nil {
			fmt.Println(err)
		}
		geocodes = append(geocodes, *obj)
	}
	if r.Method == "GET" {

		tmplBox := rice.MustFindBox("StaticWeb")
		tmplString, _ := tmplBox.String("html/list.html")
		tmpl, _ := template.New("index").Parse(tmplString)
		tmpl.Execute(w, geocodes)
	}
}

func processing(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		address := r.FormValue("Address")
		AddressURL := url.PathEscape(address)
		YandexRequest := "https://geocode-maps.yandex.ru/1.x/?format=json&apikey=" + APIKey + "&geocode=" + AddressURL
		YandexResonse, _ := http.Get(YandexRequest)
		geodata, _ := ioutil.ReadAll(YandexResonse.Body)
		result := gjson.Get(string(geodata), "response.GeoObjectCollection.featureMember.#.GeoObject.Point.pos")
		str := []rune(result.String())
		pointStr := string(str[2 : len(result.String())-2])
		point := strings.Fields(pointStr)
		var coord vec2
		coord.x, _ = strconv.ParseFloat(point[1], 64)
		coord.y, _ = strconv.ParseFloat(point[0], 64)
		fmt.Println("Point = ", coord)
		resultSQL, err := db.Exec("Insert into geocode (address,X,Y) values($1,$2,$3);", address, coord.x, coord.y)
		if err != nil {
			fmt.Println(err)
		}
		rowsAffected := resultSQL.RowsAffected
		fmt.Println("Строка добалена", rowsAffected)
		// fmt.Println(string(geodata))
	}

}

func clearlist(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		db.Query("Delete from geocode where id != 0;")
	}
}

func main() {
	box := rice.MustFindBox("StaticWeb")
	StaticWebFileServer := http.StripPrefix("/files/", http.FileServer(box.HTTPBox()))
	http.Handle("/files/", StaticWebFileServer)

	http.HandleFunc("/", index)
	http.HandleFunc("/clearlist", clearlist)
	http.HandleFunc("/list", list)
	http.HandleFunc("/processing", processing)
	http.ListenAndServe(":8080", nil)
}
