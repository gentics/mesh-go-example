# Gentics Mesh Go Example

This example combines Gentics Mesh with Golang. It uses [GJSON](https://github.com/tidwall/gjson) to easily access arbitrary JSON values and a litte bit of the [Gorilla toolkit](http://www.gorillatoolkit.org/) for HTTP routing and logging.

## Download and setup
Make sure you have go [installed and set up](https://golang.org/doc/install).

```
# Download example and change to directory
go get github.com/gentics/mesh-go-example

# Download Gentics Mesh from http://getmesh.io/Download and start it in another terminal
java -jar mesh-demo-0.9.xx.jar
```

## Running the example
```
# Change to repository directory
cd $GOPATH/src/github.com/gentics/mesh-go-example

# Start the example and point your Browser to http://localhost:8081/
go run main.go
```