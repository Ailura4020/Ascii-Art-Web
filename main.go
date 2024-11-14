package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
)

const port = ":8080"

func main() {
	http.HandleFunc("/", Home)
	http.HandleFunc("/result", Result)
	http.HandleFunc("/download", Download)
	http.HandleFunc("/error", ErrorHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	fmt.Println("(http://localhost:8081) - Server started on port", port)
	http.ListenAndServe(port, nil)
}

func RenderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	page, err := template.ParseFiles("templates/" + tmpl + ".html")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		http.Error(w, "error 400", http.StatusBadRequest)
		log.Printf("error template %v", err)
		return
	}
	err = page.Execute(w, data)
	if err != nil {
		http.Error(w, "Error 500, Internal server error", http.StatusInternalServerError)
		log.Printf("error template %v", err)
		return
	}
}

func Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		RenderTemplate(w, "error", nil)
	} else {
		RenderTemplate(w, "index", nil)
	}
}

func ErrorHandler(w http.ResponseWriter, r *http.Request) {
	status, err := strconv.Atoi(r.URL.Query().Get("status"))
	if err != nil {
		status = http.StatusInternalServerError
	}
	message := r.URL.Query().Get("message")
	Error(w, status, message)
}

func Error(w http.ResponseWriter, status int, message string) {
	tmpl, err := template.ParseFiles("templates/error.html")
	if err != nil {
		http.Error(w, "Server Error", http.StatusInternalServerError)
		log.Printf("Error parsing template: %v", err)
		return
	}
	data := struct {
		Status  int
		Message string
	}{
		Status:  status,
		Message: message,
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

func Result(w http.ResponseWriter, r *http.Request) {
	text := strings.ReplaceAll(r.FormValue("text"), "\r\n", `\n`)
	banner := r.FormValue("banner")
	fmt.Println("the banner used:", banner)
	fmt.Println("the text", text)
	if text != "" {
		if !isValidASCII(text) {
			http.Redirect(w, r, "/error?status=400&message=Invalid+input:+only+ASCII+characters+between+32+and+126+are+allowed.", http.StatusSeeOther)
			return
		}

		result, err := ascii(text, banner)
		if err != nil {
			http.Redirect(w, r, "/error?status=500&message="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			return
		}

		tempFile, err := os.CreateTemp("", "ascii-art-*.txt")
		if err != nil {
			http.Redirect(w, r, "/error?status=500&message=Server+Error", http.StatusSeeOther)
			return
		}
		defer tempFile.Close()
		_, err = tempFile.WriteString(result)
		if err != nil {
			http.Redirect(w, r, "/error?status=500&message=Server+Error", http.StatusSeeOther)
			return
		}
		RenderTemplate(w, "result", map[string]interface{}{
			"Result":   result,
			"FilePath": tempFile.Name(),
			"Text":     text,
			"Banner":   banner,
		})
	} else {
		RenderTemplate(w, "index", nil)
	}
}

func Download(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		http.Error(w, "file not found", http.StatusBadRequest)
		return
	}
	Openfile, err := os.Open(filePath) // Open the file to be downloaded
	if err != nil {
		http.Error(w, "File not found.", http.StatusInternalServerError) // Return 500 if file is not found
		return
	}
	defer Openfile.Close()                             // Close after function returns
	FileStat, _ := Openfile.Stat()                     // Get info from file
	FileSize := strconv.FormatInt(FileStat.Size(), 10) // Get file size as a string
	w.Header().Set("Content-Disposition", "attachment; filename=ascii-art.txt")
	w.Header().Set("Content-Length", FileSize)
	w.Header().Set("Content-Type", "text/plain")
	Openfile.Seek(0, 0)
	io.Copy(w, Openfile)
}

func ascii(text string, banner string) (string, error) {
	file, err := os.Open(banner + ".txt")
	if err != nil {
		return "", fmt.Errorf("banner file not found: %s", banner)
	}
	defer file.Close()

	var tableau []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		tableau = append(tableau, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading banner file: %v", err)
	}
	tableau = tableau[1:]
	tableau = append(tableau, " ")

	words := strings.Split(text, `\n`)
	result := ""
	for _, word := range words {
		if word != "" {
			result += Printascii(tableau, string(word))
		} else {
			result += "\n"
		}
	}

	return result, nil
}

func isValidASCII(text string) bool {
	for _, char := range text {
		if char < 32 || char > 126 {
			return false
		}
	}
	return true
}

func Printascii(tableau []string, text string) string {
	result := "\n"
	for i := 0; i < 9; i++ {
		for _, letter := range text {
			if rune(letter) >= 32 && rune(letter) <= 126 {
				index := (int(letter) - 32) * 9
				output := (tableau[index+i])
				result = result + output
			}
		}
		if i == 8 {
			break
		} else {
			result = result + "\n"
		}
	}
	return result
}
