package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	"github.com/tidwall/gjson"
)

const (
	BASEURL  = "http://localhost:8080/api/v1/"
	USERNAME = "admin"
	PASSWORD = "admin"
)

var (
	// MeshSession used to login on the mesh backend
	MeshSession string
)

// templateData is the struct that we pass to our HTML templates, containing
// necessary data to render pages
type templateData struct {
	Breadcrumb []gjson.Result
	Category   *gjson.Result
	Products   *[]gjson.Result
}

// LoadChildren takes a nodes uuid and returns its children.
func LoadChildren(uuid string) *[]gjson.Result {
	r := MeshGetRequest("demo/nodes/" + uuid + "/children?expandAll=true&resolveLinks=short")
	defer r.Body.Close()
	bytes, _ := ioutil.ReadAll(r.Body)
	json := gjson.ParseBytes(bytes).Get("data").Array()
	return &json
}

// LoadBreadcrumb retrieves the top level nodes used to display the navigation
func LoadBreadcrumb() []gjson.Result {
	r := MeshGetRequest("demo/navroot/?maxDepth=1&resolveLinks=short")
	defer r.Body.Close()
	bytes, _ := ioutil.ReadAll(r.Body)
	json := gjson.ParseBytes(bytes).Get("root.children").Array()
	return json
}

// MeshGetRequest issues a logged in request to the mesh backend
func MeshGetRequest(path string) *http.Response {
	url := BASEURL + path
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(&http.Cookie{
		Name:  "mesh.session",
		Value: MeshSession,
	})
	client := http.Client{}
	resp, _ := client.Do(req)
	return resp
}

// MeshLogin logs into the mesh backend and sets the session id
func MeshLogin(username string, password string) {
	body := map[string]string{
		"username": USERNAME,
		"password": PASSWORD,
	}
	payload, _ := json.Marshal(body)
	r, _ := http.Post(BASEURL+"auth/login", "application/json", bytes.NewBuffer(payload))
	for _, cookie := range r.Cookies() {
		if cookie.Name == "mesh.session" {
			MeshSession = cookie.Value
		}
	}
}

func main() {
	// Log into mesh backend to retrieve session cookie
	MeshLogin(USERNAME, PASSWORD)

	// Set up router handling incoming requests
	router := mux.NewRouter()
	router.HandleFunc("/", IndexHandler)
	router.HandleFunc("/{path:.*}", PathHandler)

	// Start http server
	http.Handle("/", router)
	http.ListenAndServe(":8081", nil)
}

// IndexHandler handles requests to the webroot
func IndexHandler(w http.ResponseWriter, req *http.Request) {
	t, _ := template.ParseFiles("templates/base.html", "templates/navigation.html", "templates/welcome.html")
	data := templateData{
		Breadcrumb: LoadBreadcrumb(),
	}

	t.Execute(w, data)
}

// PathHandler handles requests all pages except the index
func PathHandler(w http.ResponseWriter, req *http.Request) {
	// Use the requested path on the webroot endpoint to get a node
	path := mux.Vars(req)["path"]
	r := MeshGetRequest("demo/webroot/" + path + "?resolveLinks=short")
	defer r.Body.Close()

	// Check if the loaded node is an image and simply pass through the data if
	// it is.
	if match, _ := regexp.MatchString("^image/.*", r.Header["Content-Type"][0]); match {
		w.Header().Set("Content-Type", r.Header["Content-Type"][0])
		io.Copy(w, r.Body)

	} else {
		// Otherwise parse the body to json
		bytes, _ := ioutil.ReadAll(r.Body)
		node := gjson.ParseBytes(bytes)

		// If the loaded node is a vehicle, render the product
		// detail page.
		if node.Get("schema.name").String() == "vehicle" {
			t, _ := template.ParseFiles("templates/base.html", "templates/navigation.html", "templates/productDetail.html")
			data := templateData{
				Breadcrumb: LoadBreadcrumb(),
				Products:   &[]gjson.Result{node},
			}
			t.Execute(w, data)
		} else {
			// In all other cases the node is a category, render product
			// list.
			t, _ := template.ParseFiles("templates/base.html", "templates/navigation.html", "templates/productList.html")
			data := templateData{
				Breadcrumb: LoadBreadcrumb(),
				Category:   &node,
				Products:   LoadChildren(node.Get("uuid").String()),
			}

			t.Execute(w, data)
		}
	}
}
