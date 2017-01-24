package main

import (
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"

	"github.com/tidwall/gjson"
)

const (
	// BASEURL hue
	BASEURL = "http://admin:admin@localhost:8080/api/v1/"
)

type category struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func loadChildren(uuid string) *[]gjson.Result {
	url := BASEURL + "demo/nodes/" + uuid + "/children?expandAll=true&resolveLinks=short"
	r, _ := http.Get(url)
	defer r.Body.Close()
	bytes, _ := ioutil.ReadAll(r.Body)
	json := gjson.ParseBytes(bytes).Get("data").Array()
	return &json

}
func loadBreadcrumb() *[]gjson.Result {
	url := BASEURL + "demo/navroot/?maxDepth=1&resolveLinks=short"
	r, _ := http.Get(url)
	defer r.Body.Close()
	bytes, _ := ioutil.ReadAll(r.Body)
	json := gjson.ParseBytes(bytes).Get("root.children").Array()
	return &json
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/" {
			// Handle index page
			breadcrumb := loadBreadcrumb()
			t, err := template.ParseFiles("templates/base.html", "templates/navigation.html", "templates/welcome.html")
			if err != nil {
				log.Fatal(err)
			}
			data := struct {
				Breadcrumb *[]gjson.Result
			}{
				breadcrumb,
			}
			err = t.Execute(w, data)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			/* Handle rest of page using WebRoot endpoint to resolve the path
			 * to a node. The path will later be used to determine which
			 * template to use in order to render a page.
			 */
			url := BASEURL + "demo/webroot/" + r.RequestURI + "?resolveLinks=short"
			r, _ := http.Get(url)
			defer r.Body.Close()

			if match, _ := regexp.MatchString("^image/.*", r.Header["Content-Type"][0]); match {
				// Check if the loaded nodes is an image
				w.Header().Set("Content-Type", r.Header["Content-Type"][0])
				io.Copy(w, r.Body)

			} else {
				// Otherwise load the body as json
				bytes, _ := ioutil.ReadAll(r.Body)
				node := gjson.ParseBytes(bytes)

				if node.Get("schema.name").String() == "vehicle" {
					/* If the loaded node is a vehicle, render the product
					 * detail page.
					 */
					t, err := template.ParseFiles("templates/base.html", "templates/navigation.html", "templates/productDetail.html")
					if err != nil {
						log.Fatal(err)
					}
					data := struct {
						Breadcrumb *[]gjson.Result
						Product    gjson.Result
					}{
						loadBreadcrumb(),
						node,
					}
					err = t.Execute(w, data)
					if err != nil {
						log.Fatal(err)
					}

				} else {
					/* In all other cases the node is a category, render product
					 * list.
					 */
					t, err := template.ParseFiles("templates/base.html", "templates/navigation.html", "templates/productList.html")
					if err != nil {
						log.Fatal(err)
					}
					data := struct {
						Breadcrumb *[]gjson.Result
						Category   gjson.Result
						Products   *[]gjson.Result
					}{
						loadBreadcrumb(),
						node,
						loadChildren(node.Get("uuid").String()),
					}
					err = t.Execute(w, data)
					if err != nil {
						log.Fatal(err)
					}

				}
			}

		}

	})

	http.ListenAndServe(":8081", nil)
}
