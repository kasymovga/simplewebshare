package main;

import (
	"net/http"
	"net/url"
	"fmt"
	"os"
	"io/ioutil"
	"io"
	"html"
)

var base_directory string;
var code string;

func print_page(w http.ResponseWriter, title string, body string) {
	w.Header().Add("Content-type", "text/html");
	fmt.Fprintf(w,
`<html>
	<head>
		<title>` + title + `</title>
	</head>
	<body>
` + body + `
	</body>
</html>`);
}

func show_error(w http.ResponseWriter, msg string, err error) {
	print_page(w, "Error", msg + ": " + err.Error());
}

func get_file(w http.ResponseWriter, file_path string, disposition string) {
	file, err := os.OpenFile(file_path, os.O_RDONLY, 0)
	if err != nil  {
		show_error(w, "File open error", err);
		return;
	}
	defer file.Close();
	w.Header().Add("Content-type", "application/x-octet-stream");
	w.Header().Add("Content-disposition", "inline; filename=\"" + url.QueryEscape(disposition) + "\"");
	io.Copy(w, file);
}

func process(w http.ResponseWriter, r *http.Request) {
	var auth = false;
	code_cookie, _ := r.Cookie("code");
	if code_cookie != nil {
		auth = code_cookie.Value == code;
	}
	if (!auth && r.FormValue("code") == code) {
		http.SetCookie(w, &http.Cookie{Name: "code", Value: code});
		auth = true;
	}
	if !auth {
		print_page(w, "Code required",
`		<form method='POST' action='/'><input name='code' type='text'><input type='submit' value='Enter'></form>`);
		return;
	}
	if r.Method == "POST" && r.Header.Get("Content-Type") == "multipart/form-data" {
		r.ParseMultipartForm(32 << 20);
		file, handler, err := r.FormFile("uploadfile");
		if (err != nil) {
			show_error(w, "Cannot get form file", err);
			return;
		}
		defer file.Close();
		file_dest, err := os.OpenFile(base_directory + "/" + handler.Filename, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			show_error(w, "Cannot open file", err);
			return;
		}
		defer file_dest.Close()
		io.Copy(file_dest, file)
	} else {
		file_name := r.FormValue("file");
		if file_name != "" {
			get_file(w, base_directory + "/" + file_name, file_name);
			return;
		}
	}
	files, err := ioutil.ReadDir(base_directory)
	if err != nil {
		show_error(w, "Cannot list directory", err);
		return;
	}
	var file_name_escaped string;
	var body string = "";
	for _, f := range files {
		file_name_escaped = html.EscapeString(f.Name());
		body += "<a href='/?file=" + url.QueryEscape(f.Name()) + "'>" + file_name_escaped + "</a><br>\n";
	}
	body += "<form method='POST' action='/' enctype='multipart/form-data'><input name='uploadfile' type='file' value='Choose File'><input type='submit' value='Upload'></form>";
	print_page(w, "File swapper", body);
}

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage: " + os.Args[0] + " [directory] [code]\n");
		return;
	}
	base_directory = os.Args[1];
	code = os.Args[2];
	http.HandleFunc("/", process);
	http.ListenAndServe(":8080", nil);
}
