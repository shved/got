package misc

import (
	"io/ioutil"
	"log"
	"os"
)

// CreateDummyApp creates dummy application in current directory.
// Current directory should be set before use this function.
func CreateDummyApp() {
	err := os.Mkdir("app", 0755)
	err = os.Mkdir("app/lib", 0755)
	err = os.Mkdir("app/views", 0755)

	err = ioutil.WriteFile("app/sample.file", []byte("some data here"), 0644)
	err = ioutil.WriteFile("app/lib/sample.php", []byte("some data here"), 0644)
	err = ioutil.WriteFile("app/views/sample.html", []byte("<body>some data here</body>"), 0644)

	if err != nil {
		log.Fatalf("failed to create dummy app: %v", err)
	}
}

// CreateDummyApp removes dummy application and .got repo folder from current path.
func RemoveDummyApp() {
	err := os.RemoveAll("app")
	err = os.RemoveAll(".got")
	if err != nil {
		log.Fatalf("filed to remove dummy app: %v", err)
	}
}
