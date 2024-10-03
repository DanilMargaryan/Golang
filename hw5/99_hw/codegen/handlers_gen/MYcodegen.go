package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"math"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "api.go", nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create("MY_api.go")
	defer out.Close()

	fmt.Fprintln(out, `package `+node.Name.Name)
	fmt.Fprintln(out) // empty line
	fmt.Fprintln(out, `import "net/http"`)
	fmt.Fprintln(out, `import "net/url"`)
	fmt.Fprintln(out) // empty line

	structGenerator(out, node)

	methodsMap := requiresFunc(node) // Собираем методы
	generatorFunc(out, methodsMap)
}

func requiresFunc(node *ast.File) *map[string][]*ast.FuncDecl {
	var methodsMap = make(map[string][]*ast.FuncDecl)

	for _, decl := range node.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			fmt.Printf("SKIP %#T is not *ast.FunclDecl\n", funcDecl)
			continue
		}

		if funcDecl.Doc == nil {
			fmt.Printf("SKIP %#T is not *ast.FuncDecl.Recv\n", funcDecl)
			continue
		}

		if funcDecl.Recv == nil || len(funcDecl.Recv.List) != 1 {
			fmt.Printf("SKIP %#T is not *ast.FuncDecl.Recv\n", funcDecl)
			continue
		}

		firstReceiver := funcDecl.Recv.List[0]

		receiverType, ok := firstReceiver.Type.(*ast.StarExpr)
		if !ok {
			fmt.Printf("SKIP: %s receiver is not a pointer to a struct\n", funcDecl.Name.Name)
			continue
		}

		ident, ok := receiverType.X.(*ast.Ident)

		if !ok {
			fmt.Printf("SKIP %#T is not *ast.StartExpr\n", funcDecl)
			continue
		}

		methodsList, ok := methodsMap[ident.Name]
		if !ok {
			methodsList = []*ast.FuncDecl{funcDecl}
		} else {
			methodsList = append(methodsList, funcDecl)
		}
		methodsMap[ident.Name] = methodsList
	}
	return &methodsMap
}

func generatorFunc(out *os.File, methodsMap *map[string][]*ast.FuncDecl) {
	for structName, methods := range *methodsMap {
		fmt.Fprintf(out, `func (srv *%s) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {`, structName)

		for _, fn := range methods {
			re := regexp.MustCompile(`apigen:api\s+({.*})`)
			match := re.FindStringSubmatch(fn.Doc.Text())

			if len(match) < 0 {
				continue
			}

			var result map[string]interface{}

			err := json.Unmarshal([]byte(match[1]), &result)
			if err != nil {
				fmt.Printf("Error parsing JSON: %v\n", err)
				continue
			}

			requestMethod := result["method"]
			if requestMethod == nil {
				requestMethod = ""
			}
			_, _ = fmt.Fprintln(out)
			_, _ = fmt.Fprintf(out, `	case "%s":
		requestValues, err := validRequest(w, r, "%v", %v)
		if err != nil {
			MarshalAndWrite(w, err)
			return
		}
`, result["url"], requestMethod, result["auth"])
			fmt.Fprintf(out, `
		param := %s{}
		if err := param.Valid(requestValues); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			MarshalAndWrite(w, err)
			return
		}
		response, err := srv.%s(r.Context(), param)
		if err != nil {
			SetFuncError(w, err)
			return	
		}
		w.WriteHeader(http.StatusOK)
		MarshalAndWrite(w, &ResponseError{"", response})
`, fn.Type.Params.List[1].Type, fn.Name.Name)

		}
		fmt.Fprintln(out, `
	default:
		w.WriteHeader(http.StatusNotFound)
		MarshalAndWrite(w, &ResponseError{"unknown method", nil})
	}
}`)
		fmt.Fprintln(out)
	}
}

func structGenerator(out *os.File, node *ast.File) {
	for _, decl := range node.Decls {
		g, ok := decl.(*ast.GenDecl)
		if !ok {
			fmt.Printf("SKIP %#T is not *ast.GenDecl\n", decl)
			continue
		}
		for _, spec := range g.Specs {
			currType, ok := spec.(*ast.TypeSpec)
			if !ok {
				fmt.Printf("SKIP %#T is not ast.TypeSpec\n", spec)
				continue
			}
			currStruct, ok := currType.Type.(*ast.StructType)
			if !ok {
				fmt.Printf("SKIP %#T is not ast.StructType\n", currStruct)
				continue
			}

			isRequire := false
			for _, field := range currStruct.Fields.List {
				if field.Tag == nil {
					continue
				}
				tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
				apivalidator := tag.Get("apivalidator")
				if apivalidator == "-" || apivalidator == "" {
					continue
				}
				isRequire = true
			}
			if isRequire {
				generateValidMethod(out, currStruct, currType)
			}
		}
	}
}

func generateValidMethod(out *os.File, currStruct *ast.StructType, currType *ast.TypeSpec) {
	fmt.Fprintf(out, "func (s *%s) Valid(query url.Values) error {\n", currType.Name.Name)
	fmt.Fprintln(out, "\tvar err error")
	for _, field := range currStruct.Fields.List {
		fieldType, ok := field.Type.(*ast.Ident)
		if !ok {
			continue
		}

		name := field.Names[0].Name
		validator := field.Tag.Value[15 : len(field.Tag.Value)-2]
		validators := strings.Split(validator, ",")
		isRequire := false
		paramName := strings.ToLower(name)
		minValue := math.MinInt
		maxValue := math.MaxInt
		defualt := ""
		var enums []string
		for _, rule := range validators {
			switch {
			case strings.HasPrefix(rule, "required"):
				isRequire = true
			case strings.Contains(rule, "paramname="):
				paramName = strings.Replace(rule, "paramname=", "", 1)
			case strings.Contains(rule, "enum="):
				for _, enum := range strings.Split(rule[5:], "|") {
					enums = append(enums, strings.TrimSpace(enum))
				}
			case strings.Contains(rule, "min="):
				minValue, _ = strconv.Atoi(rule[4:])
			case strings.Contains(rule, "max="):
				maxValue, _ = strconv.Atoi(rule[4:])
			case strings.Contains(rule, "default="):
				defualt = rule[8:]
			}
		}

		switch fieldType.Name {
		case "int":
			fmt.Fprintf(out, "	if s.%v, err = validInt("+
				"\n\t\tquery, "+
				"\n\t\t\"%s\", "+
				"\n\t\t%v, "+
				"\n\t\t%v,"+
				"\n\t\t%d,"+
				"\n\t\t%d,"+
				"\n\t\t); err != nil {\n",
				name, paramName, isRequire, convertEnumsToIntString(enums), minValue, maxValue)
		default:
			fmt.Fprintf(out, "	if s.%v, err = validString("+
				"\n\t\tquery, "+
				"\n\t\t\"%s\", "+
				"\n\t\t%v, "+
				"\n\t\t%v,"+
				"\n\t\t%d,"+
				"\n\t\t%d,"+
				"\n\t\t\"%s\","+
				"\n\t); err != nil {\n",
				name, paramName, isRequire, convertEnumsToString(enums), minValue, maxValue, defualt)
		}
		fmt.Fprint(out, "\t\treturn err\n\t}\n")
	}
	fmt.Fprint(out, "	return nil\n}\n\n")
}

func convertEnumsToString(enums []string) string {
	if len(enums) == 0 {
		return "nil"
	}

	quotedEnums := make([]string, len(enums))
	for i, v := range enums {
		quotedEnums[i] = fmt.Sprintf("\"%s\"", v)
	}

	return "[]string{" + strings.Join(quotedEnums, ", ") + "}"
}

func convertEnumsToIntString(enums []string) string {
	if len(enums) == 0 {
		return "nil"
	}

	indexes := make([]string, len(enums))
	for i := range enums {
		indexes[i] = strconv.Itoa(i)
	}

	return "[]int{" + strings.Join(indexes, ", ") + "}"
}
