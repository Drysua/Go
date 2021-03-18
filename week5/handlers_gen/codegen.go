// go build handlers_gen/codegen.go && codegen.exe api.go api_handlers.go
// go test -v
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"
)

// {{range .StructFields.FList}} //{{.Name}}
// 	{{ $map := index $.JsonStructs.FList .Name}}
// 	params.{{.Name}} = v.Get("{{.Name)}}")
// 	params.{{.Name}} = v.Get("{{$map.Tag}}")
// 	{{end}}

type Fields struct {
	FList []Field
}

type JsonFields struct {
	FList map[string]Field
}

type Field struct {
	Name string
	Tag  Rules
	Type string
}

type TplParam struct {
	Srv          string
	ApiName      string
	StructName   string
	StructFields Fields
	FuncName     string
	User         string
	JsonStructs  JsonFields
	Method       string
}

type apiMeta struct {
	Url    string
	Auth   bool
	Method string
}

type Rules struct {
	ParamName string
	Required  bool
	Min       bool
	MinValue  int
	Max       bool
	MaxValue  int
	Enum      []string
	Default   string
}

var wrapTpl = template.Must(template.New("wrapTpl").Parse(`
func ({{.Srv}} *{{.ApiName}}) {{.FuncName}}Wrapped(w http.ResponseWriter, r *http.Request) {
	{{- if eq .Method "POST"}}
	if r.Header.Get("X-Auth") != "100500" {
		sendError(w, "unauthorized", http.StatusForbidden)
		return
	}
	if r.Method != "POST" {
		sendError(w, "bad method", http.StatusNotAcceptable)
		return
	}
	{{- end }}
	var q url.Values
	err := errors.New("")
	// заполнение структуры params
	params := {{.StructName}}{}
	if r.Method == "GET" {
		q = r.URL.Query()
	} else {
		reqBody, _ := ioutil.ReadAll(r.Body)
		q, _ = url.ParseQuery(string(reqBody))
	}
	{{range .StructFields.FList -}}//{{.Name -}}
	
	{{ if eq .Type "int" }}
	params.{{.Name}}, err = strconv.Atoi(q.Get("{{.Tag.ParamName}}"))
	if err != nil {
		sendError(w, "age must be int", http.StatusBadRequest)
		return	
	}
	{{ else }}
	params.{{.Name}} = q.Get("{{.Tag.ParamName}}") 
	{{ end }} 

	{{- if .Tag.Default -}}
	if params.{{ .Name }} == "" {
		params.{{ .Name }} = "{{ .Tag.Default }}"
	}

	{{- end -}}

	{{- if .Tag.Required -}}
	if params.{{.Name}} == "" {
		sendError(w, "{{.Tag.ParamName}} must me not empty", http.StatusBadRequest)
		return	
	}
	{{- end }}

	{{- if and .Tag.Min (eq .Type "int") -}}
	if params.{{.Name}} < {{ .Tag.MinValue }} {
		sendError(w, "{{ .Tag.ParamName }} must be >= {{ .Tag.MinValue }}", http.StatusBadRequest)
		return	
	}
	{{- end -}}

	{{- if and .Tag.Min (eq .Type "string") }}
	if len(params.{{.Name}}) < {{.Tag.MinValue}} {
		sendError(w, "{{ .Tag.ParamName }} len must be >= {{ .Tag.MinValue }}", http.StatusBadRequest)
		return	
	}
	{{- end -}}

	{{- if and .Tag.Max (eq .Type "int") }}
	if params.{{.Name}} > {{.Tag.MaxValue}} {
		sendError(w, "{{ .Tag.ParamName }} must be <= {{ .Tag.MaxValue }}", http.StatusBadRequest)
		return	
	}
	{{- end -}}

	{{if and .Tag.Max (eq .Type "string")}}
	if len(params.{{.Name}}) > {{.Tag.MaxValue}} {
		sendError(w, "{{ .Tag.ParamName }} len must be <= {{ .Tag.MaxValue }}", http.StatusBadRequest)
		return	
	}
	{{- end -}}

	{{- if .Tag.Enum }}
	enum{{ .Name }}Valid := false
	enum{{ .Name }} := []string{ {{- range $index, $element := .Tag.Enum }}{{ if $index }}, {{ end }}"{{ $element }}"{{ end -}} }
	for _, valid := range enum{{ .Name }} {
		if valid == params.{{ .Name }} {
			enum{{ .Name }}Valid = true
			break
		}
	}

	if !enum{{ .Name }}Valid {
		sendError(w, "{{ .Tag.ParamName }} must be one of [" + strings.Join(enum{{ .Name }}, ", ") + "]", http.StatusBadRequest)
		return
	}
	{{ end }}
	{{- end }}


	user, err := srv.{{.FuncName}}(context.TODO(), params)
	if err != nil {
		switch err := err.(type) {
		case ApiError:
			sendError(w, err.Error(), err.HTTPStatus)
		default:
			sendError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	result := CR{
		"error":    "",
		"response": user,
	}

	b, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, string(b))
}
`))

type Api struct {
	Srv     string
	ApiName string
	Methods []MR
}

type MR struct {
	Method string
	Route  string
}

var serveTpl = template.Must(template.New("serveTpl").Parse(`
func ({{.Srv}} *{{.ApiName}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	{{ range .Methods }}
	case "{{.Route}}":
		{{$.Srv}}.{{ .Method }}Wrapped(w, r)
	{{ end }}default:
		sendError(w, "unknown method", http.StatusNotFound)
	}
}
`))

func main() {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(os.Args[2])

	fmt.Fprintln(out, `package `+node.Name.Name)
	fmt.Fprintln(out)
	fmt.Fprintln(out, `import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)
`)
	fmt.Fprintln(out, `
func sendError(w http.ResponseWriter, error string, code int) {
	js, err := json.Marshal(CR{"error": error})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
		
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintln(w, string(js))
}
`)

	Structs := make(map[string]Fields)
	jsonStructs := make(map[string]JsonFields)

	// var ApiList []Api
	MRmap := make(map[string][]MR)
	apiNames := make(map[string]bool)
	for _, f := range node.Decls {
		//Struct parsing
		if g, ok := f.(*ast.GenDecl); ok {
			if g.Tok != token.TYPE {
				// fmt.Printf("SKIP %s is not TYPE\n", g.Tok)
				continue
			}

			for _, spec := range g.Specs {
				currStruct, ok := spec.(*ast.TypeSpec).Type.(*ast.StructType)
				if !ok {
					// fmt.Printf("SKIP %T is not ast.StructType\n", currStruct)
					continue
				}
				structName := spec.(*ast.TypeSpec).Name.Name
				// fmt.Println(structName)
				// for _, field := range currStruct.Fields.List {
				// 	 fmt.Printf("	Struct fields %#v\n", field.Names[0].Name)
				// }
				var F Fields
				J := JsonFields{FList: make(map[string]Field)}
				needVal := false
				needJson := false

			FIELDS_LOOP:
				for _, field := range currStruct.Fields.List {

					if field.Tag == nil {
						// fmt.Printf("SKIP %T is not ast.StructType\n", currStruct)
						continue FIELDS_LOOP
					}

					tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])

					//search json tag
					t, ok := tag.Lookup("json")
					if !ok {
						// fmt.Printf("No json tag, tag is %s\n", tag)
					}

					needJson = true
					// J.FList = append(J.FList, Field{
					// 	Name: field.Names[0].Name,
					// 	Tag:  t,
					// 	Type: field.Type.(*ast.Ident).Name,
					// })
					J.FList[field.Names[0].Name] = Field{
						Name: field.Names[0].Name,
						Tag:  Rules{ParamName: t},
						Type: field.Type.(*ast.Ident).Name,
					}

					//search apivalidator tag
					t, ok = tag.Lookup("apivalidator")
					if !ok {
						// fmt.Printf("No validation tag, tag is %s\n", tag)
						continue FIELDS_LOOP
					}

					rules := Rules{}
					for _, rule := range strings.Split(t, ",") {
						tParts := strings.Split(rule, "=")
						switch tParts[0] {
						case "required":
							rules.Required = true
						case "paramname":
							rules.ParamName = tParts[1]
						case "min":
							rules.Min = true
							rules.MinValue, _ = strconv.Atoi(tParts[1])
						case "max":
							rules.Max = true
							rules.MaxValue, _ = strconv.Atoi(tParts[1])
						case "enum":
							rules.Enum = strings.Split(tParts[1], "|")
						case "default":
							rules.Default = tParts[1]
						}
					}

					if rules.ParamName == "" {
						rules.ParamName = strings.ToLower(field.Names[0].Name)
					}

					needVal = true
					fmt.Printf("%+v\n", rules)
					F.FList = append(F.FList, Field{
						Name: field.Names[0].Name,
						Tag:  rules,
						Type: field.Type.(*ast.Ident).Name,
					})
				}
				if needVal == true {
					Structs[structName] = F
				}

				if needJson == true {
					jsonStructs[structName] = J
				}
			}
		}
		//Func parsing
		if g, ok := f.(*ast.FuncDecl); ok {
			if g.Doc == nil {
				// fmt.Printf("SKIP func %#v doesnt have comments\n", g.Name.Name)
				continue
			}
			fmt.Printf("\nfunc %#v\n", g.Name.Name)
			needCodegen := false
			api := apiMeta{}
			jsonStr := ""
			for _, comment := range g.Doc.List {
				jsonStr = comment.Text[len("// apigen:api"):]
				if err := json.Unmarshal([]byte(jsonStr), &api); err == nil {
					break
				}
				needCodegen = needCodegen || strings.HasPrefix(comment.Text, "// apigen")
			}

			apiName := g.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
			structName := g.Type.Params.List[1].Type.(*ast.Ident).Name
			user := g.Type.Results.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
			fmt.Println(">>>>>>>>>>> ", g.Type.Results.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name, "<<<<<<<<<<")
			p := TplParam{
				Srv:          g.Recv.List[0].Names[0].Name, //srv
				ApiName:      apiName,                      //MyApi
				StructName:   structName,                   //CreateParams
				StructFields: Structs[structName],
				FuncName:     g.Name.Name, //Profile, Create
				User:         user,        //User, NewUser
				JsonStructs:  jsonStructs[user],
				Method:       api.Method,
			}
			err = wrapTpl.Execute(out, p)
			if err != nil {
				fmt.Println(">>>>", err, "<<<<<")
			}
			apiNames[apiName] = true
			MRmap[apiName] = append(MRmap[apiName], MR{
				Method: g.Name.Name,
				Route:  api.Url,
			})
		}

	}
	for api, _ := range apiNames {
		Api := Api{
			Srv:     "srv",
			ApiName: api,
			Methods: MRmap[api],
		}
		err = serveTpl.Execute(out, Api)
		if err != nil {
			fmt.Println(err)
		}
	}
	// PRINT(Structs, "CreateParams")
	// jsonPRINT(jsonStructs, "User")
	// fmt.Printf("%#v", Structs)

}

func PRINT(s map[string]Fields, key string) {
	fmt.Println()
	empJSON, err := json.MarshalIndent(s[key], "", "  ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Printf("MarshalIndent funnction output\n %s\n", string(empJSON))
}

func jsonPRINT(s map[string]JsonFields, key string) {
	fmt.Println()
	empJSON, err := json.MarshalIndent(s[key], "", "  ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Printf("MarshalIndent funnction output\n %s\n", string(empJSON))
}
