// Command cli runs a local dev server for the palletizer's operator web app.
// It serves the static frontend in ./static and injects the machine
// credentials (read from the VIAM_* environment variables) as the cookies the
// browser Viam SDK reads — so the page authenticates to your cell with no
// separate auth proxy.
//
// Put the machine's credentials in a .env file next to where you run the
// command (copy env.example to .env and fill in the values):
//
//	VIAM_ROBOT_FQDN=<machine-main-part-fqdn>
//	VIAM_API_KEY_ID=<api-key-id>
//	VIAM_API_KEY=<api-key>
//
// then run:
//
//	go run ./cmd/cli
//
// and open http://localhost:8080. Variables already set in the real
// environment take precedence over the .env file, so you can still export
// them by hand if you prefer.
//
// The credential cookies are readable by JavaScript and carry an API key in
// clear text, so this is a local-development tool — serve it only on your own
// machine.
package main

import (
	"bufio"
	"embed"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/viam-labs/viamkit/operatorapp"
)

//go:embed all:static
var assets embed.FS

func main() {
	if err := loadEnv(".env"); err != nil {
		log.Fatal(err)
	}

	static, err := fs.Sub(assets, "static")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("operator app on http://localhost:8080")
	log.Fatal(operatorapp.ListenAndServe(":8080", static))
}

// loadEnv reads KEY=VALUE pairs from the file at path and sets them in the
// process environment. Lines that are blank or start with '#' are ignored, and
// surrounding quotes are stripped from values. Variables already present in the
// environment are left untouched, so a real export always wins over the file.
// A missing file is not an error — exporting the VIAM_* variables by hand is a
// supported alternative.
func loadEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			continue
		}
		if _, set := os.LookupEnv(key); set {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return scanner.Err()
}
