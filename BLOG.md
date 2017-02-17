## Getting ready

Make sure you have Go installed or set it up as described [here](https://golang.org/doc/install). Now get and start the example from Github:
```
# Download example and change to directory
go get github.com/gentics/mesh-go-example
cd $GOPATH/src/github.com/gentics/mesh-go-example

# Download Gentics Mesh from http://getmesh.io/Download and start it in another terminal
java -jar mesh-demo-0.6.xx.jar

# Run the example
go run main.go
```

## The Example
Navigate your browser to http://localhost:8081/. The example web app is simply a Golang reimplementation of our previous examples in [PHP](http://getmesh.io/Blog/Building+an+API-first+Web+App+with+Gentics+Mesh+and+the+PHP+Microframework+Silex) and [NodeJS](http://getmesh.io/Blog/Getting+started+with+Express+and+the+API-first+CMS+Gentics+Mesh). A small website listing vehicles from our demo data set, grouping them into categories and generating detail pages.

### Main Logic
While our PHP and Node.js examples did only use one http route handler, there are two handler functions in this application. I like to use the popular [Gorilla toolkits](http://www.gorillatoolkit.org/) excellent routing via `mux` instead of checking if the request path equals `/` in the handler function. The first `IndexHandler` simply generates the welcome page. The only dynamic content is the breadcrumb navigation. The second `PathHandler` is more complex: it handles every request besides the welcome page, including requests to images. It uses the request path to retrieve a node from Gentics Mesh, first determining via the content type header whether the requested node is actually an image. If thats the case, the binary data is simply forwarded to the requesting client. Else, the node is decoded to JSON and depending on its schema - vehicle or category - the handler renders either a product detail page or product list page. 

```
func main() {
	// Log into mesh backend to retrieve session cookie
	MeshLogin(USERNAME, PASSWORD)

	// Set up router handling incoming requests
	router := mux.NewRouter()
	router.HandleFunc("/", IndexHandler)
	router.HandleFunc("/{path:.*}", PathHandler)
	loggedRouter := handlers.LoggingHandler(os.Stdout, router)

	// Start http server
	log.Print("Starting HTTP Server on \"http://localhost:8081\"")
	err := http.ListenAndServe(":8081", loggedRouter)
	log.Print(err)
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
```

### Using a session cookie
In this example I'm using a session cookie instead of basic auth to authenticate every reuqest to the Gentics Mesh backend. The main advantage is that Gentics Mesh only needs to check my username and password once at login, resulting in a noticeable performance increase for all subsequent requests. 
```
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
```

### Note on using GJSON
The Go programming language is a strong and statically typed language. Working with a struct for every object to unmarshal from JSON is usally the way to go. For small applications with disparate data structures delivered from a backend API like Gentics Mesh, it is often more convenient to treat the API responses as a nested JSON map with arbitrary depth. Thus the [GJSON](https://github.com/tidwall/gjson) library provides a very fast way of indexing JSON and is used in our functions and templates to parse Gentics Mesh node objects. 
