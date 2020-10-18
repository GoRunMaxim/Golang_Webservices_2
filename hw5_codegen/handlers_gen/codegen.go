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

type ServeHttpParams struct {
	FuncParams		[]FuncParams
}

type StructParams struct {
	StructParams	[]ValidParams
}

type FuncParams struct{
	Url				string		`json:"url"`			// /user/create
	Auth 			bool		`json:"auth"`			// true
	Method	 		string		`json:"method"`			// POST
	Name			string								// create
	Param			string								// CreateParams
	ReturnValue 	string								// NewUser(Struct)
	ApiName			string								// MyApi
	StructParams	[]ValidParams
}

type AllParams struct {
	isInt	 	bool
	IsRequired	bool
	ValMin 		string
	ValMax 		string
	ParamName	string
	Default		string
	Enum		[]string
}

type ValidParams struct{
	StructName		string
	Fields	 		map[string]AllParams
}

func (s* ServeHttpParams) addFunc(f FuncParams){
	s.FuncParams = append(s.FuncParams, f)
}

func (s* StructParams) addStruct(f ValidParams){
	s.StructParams = append(s.StructParams, f)
}

func (s* FuncParams) addStruct(f ValidParams){
	s.StructParams = append(s.StructParams, f)
}

var (
	serveHttpTpl = template.Must(template.New("serveHttp").Parse(`
{{range $params := .Info}}
{{range $name, $links := $params.FuncParams}}
// ServeHTTP func. DO NOT EDIT
func (h *{{$name}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path{
		{{range $funcName, $link := $links}}
		case "{{$funcName.Url}}":
			h.{{$link}}(w, r)
		{{end}}
		default:
			js, err := json.Marshal(JsonErrors{errorUnknown.Error()})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNotFound)
			w.Write(js)
			return
	}
}
{{end}}{{end}}
`))

	funcCreateTpl = template.Must(template.New("funcCreateTpl").Parse(`
// Func Wrapper. DO NOT EDIT
func (h * {{.ApiName}}) {{.Name}}(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != "POST"{
		js, err := json.Marshal(JsonErrors{errorBad.Error()})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotAcceptable)
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
		return
	}
	if r.Header.Get("X-Auth") != "100500"{
		js, err := json.Marshal(JsonErrors{errorAuth.Error()})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
		return
	}
	r.ParseForm()
`))

	funcProfileTpl = template.Must(template.New("funcProfileTpl").Parse(`
// Func Wrapper. DO NOT EDIT
func (h * {{.ApiName}}) {{.Name}}(w http.ResponseWriter, r *http.Request) {
`))
)

func main() {

	fset := token.NewFileSet() // positions are relative to fset
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(os.Args[2])

	fmt.Fprintln(out, `package `+node.Name.Name)
	fmt.Fprintln(out)
	fmt.Fprintln(out, `import (`)
	fmt.Fprintln(out, `	"encoding/json"`)
	fmt.Fprintln(out, `	"net/http"`)
	fmt.Fprintln(out, `	"errors"`)
	fmt.Fprintln(out, `	"strconv"`)
	fmt.Fprintln(out, `	"strings"`)
	fmt.Fprintln(out, `)`)
	fmt.Fprintln(out)
	fmt.Fprintln(out, `var (`)
	fmt.Fprintln(out, `	errorBad	 = errors.New("bad method")`)
	fmt.Fprintln(out, `	errorEmpty	 = errors.New("login must me not empty")`)
	fmt.Fprintln(out, `	errorAuth	 = errors.New("unauthorized")`)
	fmt.Fprintln(out, `	errorUnknown = errors.New("unknown method")`)
	fmt.Fprintln(out, `)`)
	fmt.Fprintln(out)
	fmt.Fprintln(out, `type JsonErrors struct{`)
	fmt.Fprint(out, "	Error string	`json:")
	fmt.Fprint(out, `"error"`)
	fmt.Fprintln(out, "`")
	fmt.Fprintln(out, `}`)

	structParams := StructParams{}
	serveParams := ServeHttpParams{}
	funcParams := FuncParams{}

	// Inspect the AST and find our function
	key := 0
	for _, f := range node.Decls {
		g, ok := f.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if g.Doc != nil {
			for _, comment := range g.Doc.List {
				if strings.HasPrefix(comment.Text, "// apigen:api") {

					jsonString := strings.ReplaceAll(comment.Text, "// apigen:api ", "")

					err = json.Unmarshal([]byte(jsonString), &funcParams)

					if err != nil {
						fmt.Println(err)
						return
					}
				}

				if g.Type.Params.List != nil {
					for _, p := range g.Type.Params.List {
						switch a := p.Type.(type) {
						case *ast.Ident:
							funcParams.Param = a.Name
						}
					}
				}

				if g.Type.Results.List != nil && len(g.Type.Results.List) != 0 {
					switch a := g.Type.Results.List[0].Type.(type) {
					case *ast.StarExpr:
						funcParams.ReturnValue = a.X.(*ast.Ident).Name
					}
				}

				if g.Recv != nil {
					switch a := g.Recv.List[0].Type.(type) {
					case *ast.StarExpr:
						funcParams.ApiName = a.X.(*ast.Ident).Name
						key++
					}
				}
				funcParams.Name = strings.Split(funcParams.Url, "/")[strings.Count(funcParams.Url, "/")]

				serveParams.addFunc(funcParams)
			}
		}
	}

	validParams := ValidParams{}

	for _, f := range node.Decls {
		g, ok := f.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range g.Specs {
			currType, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			currStruct := currType.Type.(*ast.StructType)
			for _, field := range currStruct.Fields.List {
				if field.Tag != nil {
					str := fmt.Sprintf("%v", field.Type)
					tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
					if value, ok := tag.Lookup("apivalidator"); ok {
						allParams := AllParams{}
						allParams.isInt = str == "int"
						validator := strings.Split(value, `,`)

						for _, value := range validator {
							//считаем что на вход подаются валидные значения
							if strings.Contains(value, "required") {
								allParams.IsRequired = true
							}

							if strings.Contains(value, "min=") {
								allParams.ValMin = strings.TrimPrefix(value, "min=")
							}

							if strings.Contains(value, "max=") {
								allParams.ValMax = strings.TrimPrefix(value, "max=")
							}

							if strings.Contains(value, "enum=") {
								enum := strings.Split(strings.TrimPrefix(value, "enum="), "|")
								allParams.Enum = enum
							}

							if strings.Contains(value, "default=") {
								allParams.Default = strings.TrimPrefix(value, "default=")
							}

							if strings.Contains(value, "paramname=") {
								allParams.ParamName = strings.TrimPrefix(value, "paramname=")
							}
						}
						validParams.StructName = fmt.Sprintf("%v", currType.Name)
						key := fmt.Sprintf("%s", field.Names)
						//ИСПРАВИТЬ ГОВНО
						key = strings.ToLower(strings.ReplaceAll(key, "[", ""))
						key = strings.ReplaceAll(key, "]", "")
						validParams.Fields = make(map[string]AllParams)
						validParams.Fields[key] = allParams
						structParams.addStruct(validParams)
					}
				}
			}
		}
	}

	//create JsonResponseStruct
	for _, v := range serveParams.FuncParams{
		fmt.Fprintln(out)
		jsonComment := "`json:\"response\"`"
		fmt.Fprintln(out, `//Response Json for `,v.Param, `do not edit`)
		fmt.Fprint(out, `type Json`)
		fmt.Fprintln(out, v.Param, `struct{
	*`, v.ReturnValue, jsonComment, "\n\tJsonErrors\n}")
	}

	hasSeen := false
	//fix, because this is boolshit
	for i := 1; i < len(serveParams.FuncParams); i++ {
		if serveParams.FuncParams[i-1].ApiName == serveParams.FuncParams[i].ApiName {
			if (!hasSeen) {
				fmt.Fprintln(out)
				fmt.Fprintln(out, "func (h *", serveParams.FuncParams[i].ApiName, " ) ServeHTTP(w http.ResponseWriter, r *http.Request) { \n	switch r.URL.Path{")
				for g := i - 1; g <= i; g++ {
					fmt.Fprint(out, `		case "`)
					fmt.Fprint(out, serveParams.FuncParams[g].Url)
					fmt.Fprintln(out, `":`)
					fmt.Fprint(out, `			h.`)
					fmt.Fprint(out, serveParams.FuncParams[g].Name)
					fmt.Fprintln(out, `(w, r)`)
				}
				hasSeen = true
			}
		} else {
			fmt.Fprintln(out, `		default:
			js, err := json.Marshal(JsonErrors{errorUnknown.Error()})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNotFound)
			w.Write(js)
			return
	}
}`)
			if hasSeen {
				fmt.Fprintln(out)
				fmt.Fprintln(out, "func (h * ", serveParams.FuncParams[i].ApiName, " ) ServeHTTP(w http.ResponseWriter, r *http.Request) { \n		switch r.URL.Path{")
				fmt.Fprint(out, `			case "`)
				fmt.Fprint(out, serveParams.FuncParams[i].Url)
				fmt.Fprintln(out, `":`)
				fmt.Fprint(out, `				h.`)
				fmt.Fprint(out, serveParams.FuncParams[i].Name)
				fmt.Fprintln(out, `(w, r)`)
				hasSeen = false
			} else {

			}
		}
	}
	fmt.Fprintln(out, `			default:
				js, err := json.Marshal(JsonErrors{errorUnknown.Error()})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNotFound)
				w.Write(js)
				return
		}
}`)

	for _, wrapper := range serveParams.FuncParams {
		if wrapper.Auth || wrapper.Method == "POST" {
			for _, value := range structParams.StructParams {
				if value.StructName == wrapper.Param {
					wrapper.addStruct(value)
				}
			}
			funcCreateTpl.Execute(out, wrapper)
			//Fix
			for i := 0; i < len(wrapper.StructParams); i++ {
				for k, v := range wrapper.StructParams[i].Fields {
					if (v.isInt) {
						fmt.Fprint(out, `	`, k, `, err := strconv.Atoi(r.Form.Get("`)
						fmt.Fprint(out, k)
						fmt.Fprintln(out, `"))`, "\n")
						fmt.Fprint(out, "\tif err != nil {\n "+
							`		js, _ := json.Marshal(JsonErrors{"`)
						fmt.Fprint(out, k)
						fmt.Fprintln(out, " must be int\"})\n\t\tw.Header().Set(\"Content-Type\", \"application/json\")\n\t\tw.WriteHeader(http.StatusBadRequest)\n\t\tw.Write(js)\n\t\treturn\n\t}")

					} else {
						fmt.Fprint(out, `	`, k, `:= r.Form.Get("`)
						fmt.Fprint(out, k)
						fmt.Fprintln(out, `")`, "\n")
					}

					if v.IsRequired {
						fmt.Fprintln(out, "\n	if ", k, `== ""{
		js, _ := json.Marshal(JsonErrors{errorEmpty.Error()})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}`)
					}

					if v.ValMin != "" {
						if v.isInt {
							min, _ := strconv.Atoi(v.ValMin)
							err := k + ` must be >= ` + v.ValMin
							fmt.Fprint(out, "\n	if ", k, `<`, min, `{
		js, _ := json.Marshal(JsonErrors{"`)
							fmt.Fprint(out, err)
							fmt.Fprint(out, `"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
							`)
						} else {
							err := k + " len must be >= " + v.ValMin
							min, _ := strconv.Atoi(v.ValMin)
							fmt.Fprint(out, "\n	if len(", k, `)<`, min, `{
		js, _ := json.Marshal(JsonErrors{"`)
							fmt.Fprint(out, err)
							fmt.Fprintln(out,`"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}`)
						}
					}

					if v.ValMax != "" {
						if v.isInt {
							max, _ := strconv.Atoi(v.ValMax)
							err := k + ` must be <= ` + v.ValMax
							fmt.Fprint(out, "\n	if ", k, `>`, max, `{
		js, _ := json.Marshal(JsonErrors{"`)
							fmt.Fprint(out, err)
							fmt.Fprintln(out, `"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
							`)
						} else {
							err := k + ` len must be <=` + v.ValMax
							max, _ := strconv.Atoi(v.ValMax)
							fmt.Fprintln(out, "\n	if len(", k, `)>`, max, `{
		js, _ := json.Marshal(JsonErrors{"`, err, `"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}`)
						}
					}

					if v.ParamName != "" {
						fmt.Fprint(out, "\n", `	paramName := r.Form.Get("`)
						fmt.Fprint(out, v.ParamName)
						fmt.Fprintln(out, `")`)
						fmt.Fprintln(out, "\n", `	if paramName == ""{`)
						fmt.Fprintln(out, "\t\tparamName = strings.ToLower(", k, ")\n	}else{ \n", `		`, k, "= paramName\n\t}\n")
					}

					if v.Default != "" {
						fmt.Fprint(out, `	if `, k, `== ""{`, "\n\t\t", k, ` = "`)
						fmt.Fprint(out, v.Default)
						fmt.Fprintln(out, `"`, "\n\t}")
					}
					if len(v.Enum) > 0 {
						fmt.Fprintln(out, `	enum := make(map[string]bool)`)
						for _, value := range v.Enum {
							fmt.Fprint(out, `	enum["`)
							fmt.Fprint(out, ``, value)
							fmt.Fprintln(out, `"] = true`)
						}
						fmt.Fprint(out, `	_, enumName := enum[`)
						fmt.Fprint(out, k)
						fmt.Fprint(out, `]
	if enumName == false{
		js, _ := json.Marshal(JsonErrors{"`)
						fmt.Fprint(out, k)
						fmt.Fprint(out, ` must be one of [`,)
						for key, value := range v.Enum{
							if key != len(v.Enum)-1{
								fmt.Fprint(out, value, `, `)
							}else{
								fmt.Fprint(out, value)
							}
						}
						fmt.Fprint(out,`]"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return`)
						fmt.Fprintln(out, "\n\t}")
					}
				}
			}

			fmt.Fprint(out, "\tcreateParams := ", wrapper.Param, "{")
			for i := 0; i < len(wrapper.StructParams); i++ {
				for k := range wrapper.StructParams[i].Fields {
					fmt.Fprint(out, k, ", ")
				}
			}
			fmt.Fprintln(out, "}")
			fmt.Fprint(out, `	user, err := h.`)
			fmt.Fprintln(out, strings.Title(wrapper.Name), "(ctx, createParams)\n\n\tif err != nil{\n\t\tswitch err.(type){"+
				"\n\t\tcase ApiError:\n\t\t\tjs, _ := json.Marshal(JsonErrors{err.(ApiError).Err.Error()})\n\t\t\tw.Header().Set(\"Content-Type\", \"application/json\")\n\t\t\tw.WriteHeader(err.(ApiError).HTTPStatus)\n\t\t\tw.Write(js)\n\t\t\treturn\n\t\tdefault:\n\t\t\tjs, _ := json.Marshal(JsonErrors{\"bad user\"})\n\t\t\tw.Header().Set(\"Content-Type\", \"application/json\")\n\t\t\tw.WriteHeader(http.StatusInternalServerError)\n\t\t\tw.Write(js)\n\t\t\treturn\n\t\t}\n\t}")
			fmt.Fprint(out, "\tjs, _ := json.Marshal(Json")
			fmt.Fprint(out, wrapper.Param, "{user, JsonErrors{\"\"}})\n\tw.Header().Set(\"Content-Type\", \"application/json\")\n\tw.WriteHeader(http.StatusOK)\n\tw.Write(js)\n}")
		} else {
			for _, value := range structParams.StructParams {
				if value.StructName == wrapper.Param {
					wrapper.addStruct(value)
				}
			}
			funcProfileTpl.Execute(out, wrapper)
			for i := 0; i < len(wrapper.StructParams); i++ {
				for k, _ := range wrapper.StructParams[i].Fields {
					fmt.Fprintln(out, `	var`, k, `string`)
					fmt.Fprint(out, `	switch r.Method{
		case "GET":
			`,k ,` = r.URL.Query().Get("`,k,`")
			if `,k,` == ""{
				js, err := json.Marshal(JsonErrors{errorEmpty.Error()})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				w.Header().Set("Content-Type", "application/json")
				w.Write(js)
				return
			}
		case "POST":
			r.ParseForm()
			`,k ,` = r.Form.Get("`,k,`")
			if `,k ,` == ""{
				js, err := json.Marshal(JsonErrors{errorEmpty.Error()})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				w.Header().Set("Content-Type", "application/json")
				w.Write(js)
				return
			}
		default:
			http.Error(w, "Sorry, only GET and POST methods are supported ",http.StatusInternalServerError)
	}
	ctx := r.Context()
	`)
					fmt.Fprint(out, "profileParams := ", wrapper.Param, "{")
					for i := 0; i < len(wrapper.StructParams); i++ {
						for k := range wrapper.StructParams[i].Fields {
							fmt.Fprint(out, k, ", ")
						}
					}
					fmt.Fprintln(out, "}")
					fmt.Fprint(out, `	user, err := h.`)
					fmt.Fprintln(out, strings.Title(wrapper.Name), "(ctx, profileParams)\n\n\tif err != nil{\n\t\tswitch err.(type){"+
						"\n\t\tcase ApiError:\n\t\t\tjs, _ := json.Marshal(JsonErrors{err.(ApiError).Err.Error()})\n\t\t\tw.Header().Set(\"Content-Type\", \"application/json\")\n\t\t\tw.WriteHeader(err.(ApiError).HTTPStatus)\n\t\t\tw.Write(js)\n\t\t\treturn\n\t\tdefault:\n\t\t\tjs, _ := json.Marshal(JsonErrors{\"bad user\"})\n\t\t\tw.Header().Set(\"Content-Type\", \"application/json\")\n\t\t\tw.WriteHeader(http.StatusInternalServerError)\n\t\t\tw.Write(js)\n\t\t\treturn\n\t\t}\n\t}")
					fmt.Fprint(out, "\tjs, _ := json.Marshal(Json")
					fmt.Fprint(out, wrapper.Param, "{user, JsonErrors{\"\"}})\n\tw.Header().Set(\"Content-Type\", \"application/json\")\n\tw.Write(js)\n}")
				}
			}
		}
	}
}
