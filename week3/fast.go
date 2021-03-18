package main

//go test -bench . -benchmem -cpuprofile=cpu.out -memprofile=mem.out -memprofilerate=1 main_test.go fast.go common.go
//go tool pprof main.test.exe mem.out
import (
	"bufio"
	json "encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjson9f2eff5fDecodeJson(in *jlexer.Lexer, out *User) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "browsers":
			if in.IsNull() {
				in.Skip()
				out.Browsers = nil
			} else {
				in.Delim('[')
				if out.Browsers == nil {
					if !in.IsDelim(']') {
						out.Browsers = make([]string, 0, 4)
					} else {
						out.Browsers = []string{}
					}
				} else {
					out.Browsers = (out.Browsers)[:0]
				}
				for !in.IsDelim(']') {
					var v1 string
					v1 = string(in.String())
					out.Browsers = append(out.Browsers, v1)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "email":
			out.Email = string(in.String())
		case "name":
			out.Name = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson9f2eff5fEncodeJson(out *jwriter.Writer, in User) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"browsers\":"
		out.RawString(prefix[1:])
		if in.Browsers == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v2, v3 := range in.Browsers {
				if v2 > 0 {
					out.RawByte(',')
				}
				out.String(string(v3))
			}
			out.RawByte(']')
		}
	}
	{
		const prefix string = ",\"email\":"
		out.RawString(prefix)
		out.String(string(in.Email))
	}
	{
		const prefix string = ",\"name\":"
		out.RawString(prefix)
		out.String(string(in.Name))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v User) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson9f2eff5fEncodeJson(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v User) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson9f2eff5fEncodeJson(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *User) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson9f2eff5fDecodeJson(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *User) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson9f2eff5fDecodeJson(l, v)
}

// const filePath string = "./data/users.txt"

type User struct {
	Browsers []string `json:"browsers"`
	Company  string   `json:"-"`
	Country  string   `json:"-"`
	Email    string   `json:"email"`
	Job      string   `json:"-"`
	Name     string   `json:"name"`
	Phone    string   `json:"-"`
}

//FastSearch
func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	// fileContents, err := ioutil.ReadAll(file)
	// if err != nil {
	// 	panic(err)
	// }

	// fileContents, err := ioutil.ReadFile(filePath)
	// if err != nil {
	// 	panic(err)
	// }

	// r := regexp.MustCompile("@")
	// foundUsers := ""

	set := make(map[string]bool)

	// lines := strings.Split(string(fileContents), "\n")

	// users := make([]User, 0)

	// for _, line := range lines {
	// 	user := User{}
	// 	err := user.UnmarshalJSON([]byte(line))
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	users = append(users, user)
	// }
	i := -1
	user := User{}
	fmt.Fprintln(out, "found users:")
	for scanner.Scan() {
		i++
		err := user.UnmarshalJSON(scanner.Bytes())
		if err != nil {
			panic(err)
		}
		// }
		// for i, user := range users {

		isAndroid := false
		isMSIE := false

		for _, browser := range user.Browsers {
			exist := set[browser]
			if strings.Contains(browser, "MSIE") {
				isMSIE = true
				if !exist {
					set[browser] = true
				}
			}
			if strings.Contains(browser, "Android") {
				isAndroid = true
				if !exist {
					set[browser] = true
				}
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}
		// log.Println("Android and MSIE user:", user.Name, user.Email)
		// email := r.ReplaceAllString(user.Email, " [at] ")
		email := strings.Replace(user.Email, "@", " [at] ", 1)
		// foundUsers += fmt.Sprintf("[%d] %s <%s>\n", i, user.Name, email)
		fmt.Fprintf(out, "[%d] %s <%s>\n", i, user.Name, email)
	}
	// fmt.Fprintln(out, "found users:\n"+foundUsers)
	fmt.Fprintln(out, "\nTotal unique browsers", len(set))
}
func main() {
	FastSearch(ioutil.Discard)
}
