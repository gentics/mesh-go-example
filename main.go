package main

import (
	"bytes"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"

	"github.com/tidwall/gjson"
)

const (
	// BASEURL contains user, password and path to the mesh backend
	BASEURL = "http://localhost:8080/api/v1/"
)

var (
	// MeshSession used to login on the mesh backend
	MeshSession string
)

// LoadChildren returns takes a nodes uuid and returns its children.
func LoadChildren(uuid string) *[]gjson.Result {
	r := MeshGetRequest("demo/nodes/" + uuid + "/children?expandAll=true&resolveLinks=short")
	defer r.Body.Close()
	bytes, _ := ioutil.ReadAll(r.Body)
	json := gjson.ParseBytes(bytes).Get("data").Array()
	return &json
}

// LoadBreadcrumb retrieves the top level nodes used to display the navigation
func LoadBreadcrumb() *[]gjson.Result {
	r := MeshGetRequest("demo/navroot/?maxDepth=1&resolveLinks=short")
	defer r.Body.Close()
	bytes, _ := ioutil.ReadAll(r.Body)
	json := gjson.ParseBytes(bytes).Get("root.children").Array()
	return &json
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
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	return resp
}

// MeshLogin logs into the mesh backend and sets the session id
func MeshLogin() {
	r, err := http.Post(BASEURL+"auth/login", "application/json", bytes.NewBuffer([]byte(`{"username":"admin", "password":"admin"}`)))
	if err != nil {
		log.Fatal(err)
	}
	for _, cookie := range r.Cookies() {
		if cookie.Name == "mesh.session" {
			MeshSession = cookie.Value
		}
	}
}

func main() {
	// Log into mesh backend to retrieve session cookie
	MeshLogin()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Render welcome.html on
		if r.RequestURI == "/" {
			breadcrumb := LoadBreadcrumb()
			t, _ := template.ParseFiles("templates/base.html", "templates/navigation.html", "templates/welcome.html")
			data := struct {
				Breadcrumb *[]gjson.Result
			}{
				breadcrumb,
			}
			t.Execute(w, data)
		} else {
			// Handle rest of page using WebRoot endpoint to resolve the path
			// to a node. The path will later be used to determine which
			// template to use in order to render a page.
			r := MeshGetRequest("demo/webroot/" + r.RequestURI + "?resolveLinks=short")
			defer r.Body.Close()

			// Check if the loaded nodes is an image and simply pass through
			// the data if it is.
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
					data := struct {
						Breadcrumb *[]gjson.Result
						Product    gjson.Result
					}{
						LoadBreadcrumb(),
						node,
					}
					t.Execute(w, data)
				} else {
					// In all other cases the node is a category, render product
					// list.
					t, _ := template.ParseFiles("templates/base.html", "templates/navigation.html", "templates/productList.html")
					data := struct {
						Breadcrumb *[]gjson.Result
						Category   gjson.Result
						Products   *[]gjson.Result
					}{
						LoadBreadcrumb(),
						node,
						LoadChildren(node.Get("uuid").String()),
					}
					t.Execute(w, data)
				}
			}
		}
	})
	http.ListenAndServe(":8081", nil)
}
