package main

// go build gen/* && ./codegen.exe pack/unpack.go  pack/marshaller.go
// go run pack/*

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"text/template"
)

type TplParam struct {
	Srv        string
	StructName string
	FuncName   string
	Flag       bool
}

var wrapTpl = template.Must(template.New("wrapTpl").Parse(`
func ({{.Srv}} *{{.StructName}}) {{.FuncName}}Wrapped(w http.ResponseWriter, r *http.Request) {
	{{ if .Flag -}}

	if r.Header.Get("X-Auth") != "100500" {
		sendError(w, "unauthorized", http.StatusForbidden)
		return
	}
	if r.Method != "POST" {
		sendError(w, "bad method", http.StatusNotAcceptable)
		return
	}

	{{ end -}}

	// заполнение структуры params
	params := GetParams(r, w)
	// валидирование параметров
	// логин не должен быть пустым
	if params.Login == "" {
		sendError(w, "login must me not empty", http.StatusBadRequest)
		return
	}
	if params.Age == -42 {
		sendError(w, "age must be int", http.StatusBadRequest)
		return
	}
	if params.Age < 0 {
		sendError(w, "age must be >= 0", http.StatusBadRequest)
		return
	}
	if params.Age > 128 {
		sendError(w, "age must be <= 128", http.StatusBadRequest)
		return
	}
	if !(params.Status == "user" || params.Status == "moderator" || params.Status == "admin") {
		sendError(w, "status must be one of [user, moderator, admin]", http.StatusBadRequest)
		return
	}
	user, err := srv.{{.FuncName}}(context.TODO(), ProfileParams{params.Login})
	if err != nil {
		switch err := err.(type) {
		case ApiError:
			sendError(w, err.Error(), err.HTTPStatus)
		default:
			sendError(w, err.Error(), http.StatusBadRequest)
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

func main() {
	fset := token.NewFileSet()
	fmt.Println(os.Args[1])
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		fmt.Println("File parser err", err)
		return
	}

	out, _ := os.Create(os.Args[2])

	for _, f := range node.Decls {
		//A GenDecl node (generic declaration node) represents an import,
		//constant, type or variable declaration.

		g, ok := f.(*ast.FuncDecl)
		if !ok {
			// fmt.Printf("SKIP %T is not *ast.FuncDecl\n", f)
			continue
		}
		if g.Doc == nil {
			// fmt.Printf("SKIP fuc %#v doesnt have comments\n", g.Name.Name)
			continue
		}
		needCodegen := false
		for _, comment := range g.Doc.List {
			needCodegen = needCodegen || strings.HasPrefix(comment.Text, "// apigen")
		}
		if !needCodegen {
			fmt.Printf("SKIP func %#v doesnt have apigen mark\n", g.Name.Name)
			continue
		}
		// func (srv *MyApi) ProfileWrapped(w http.ResponseWriter, r *http.Request) {
		p := TplParam{
			Srv:        g.Recv.List[0].Names[0].Name,
			StructName: g.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name,
			FuncName:   g.Name.Name,
			Flag:       true,
		}
		// fmt.Fprintln(out, "func ("+g.Recv.List[0].Names[0].Name+" *"+g.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name+") "+g.Name.Name+"Wrapped(w http.ResponseWriter, r *http.Request) {")
		err = wrapTpl.Execute(out, p)
		if err != nil {
			fmt.Println(">>>>", err, "<<<<<")
		}
		fmt.Printf("process function %s\n", g.Name.Name)
		for _, i := range g.Doc.List {
			fmt.Printf("	Doc %+v\n", i)
		}
		fmt.Printf("	Recv.List[0].Type %+v\n", g.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident))
		fmt.Printf("	Recv.List.Names %+v\n", g.Recv.List[0].Names[0])
		fmt.Printf("	Recv.List[0].Type %+v\n", g.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident))
		fmt.Println("	Name", g.Name)
		fmt.Printf("	Type.Params.List[0] %+v\n", g.Type.Params.List[0].Type)
		fmt.Printf("	Type.Params.List[1] %+v\n", g.Type.Params.List[1].Type)
		fmt.Printf("	Type.Results.List[0] %+v\n", g.Type.Results.List[0].Type)
		fmt.Printf("	Type.Results.List[1] %+v\n", g.Type.Results.List[1].Type)
		fmt.Println("	Body", g.Body)

	}
}
