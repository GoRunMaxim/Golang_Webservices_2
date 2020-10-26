package main

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	router := http.NewServeMux()
	router.HandleFunc("/", serveHttp(db))

	return router, nil
}

func serveHttp(db *sql.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			r.ParseForm()
			queryParam := r.URL.Query()
			url := r.URL.Path
			u, _ := r.URL.Parse(url)
			if len(queryParam) > 0{
				var limit, offset int
				var ok error
				for key, value := range queryParam {
					switch key {
					case "limit":
						limit, ok = strconv.Atoi(value[0])
						if ok != nil{
							limit = 5
						}
					case "offset":
						offset, ok = strconv.Atoi(value[0])
						if ok != nil{
							offset = 0
						}
					default:
						//error
					}
				}
				table, errString := showQueryTable(db, strings.TrimLeft(u.Path, "/"), limit, offset)
				if errString != ""{
					resp := map[string]interface{}{
						"error": errString,
					}
					js, err := json.Marshal(resp)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusNotFound)
					w.Write(js)
				}else{
					resp := map[string]interface{}{
						"response": map[string]interface{}{
							"records": table,
						},
					}
					js, err := json.Marshal(resp)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusOK)
					w.Write(js)
				}
				return
			}
			if u.Path == "/"{
				tables := showTables(db)
				resp := map[string]interface{}{
					"response": map[string]interface{}{
						"tables": tables,
					},
				}
				js, err := json.Marshal(resp)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				w.Write(js)
				return
			} else if strings.Count(u.Path, "/") == 1{
				table, errString := showTable(db, strings.TrimLeft(u.Path, "/"))
				if errString != ""{
					resp := map[string]interface{}{
						"error": errString,
					}
					js, err := json.Marshal(resp)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusNotFound)
					w.Write(js)
				}else{
					resp := map[string]interface{}{
						"response": map[string]interface{}{
							"records": table,
						},
					}
					js, err := json.Marshal(resp)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusOK)
					w.Write(js)
				}
				return
			}else if strings.Count(u.Path, "/") ==2 {
				args := strings.Split(strings.TrimLeft(u.Path, "/"),"/")
				path := args[0]
				offset , _ := strconv.Atoi(args[1])
				table, err := showQueryTable(db, path, 1, offset-1)
				if err != ""{
					resp := map[string]interface{}{
						"error": "record not found",
					}
					js, err := json.Marshal(resp)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusNotFound)
					w.Write(js)
				}else{
					resp := map[string]interface{}{
						"response": map[string]interface{}{
							"record": table[0],	//fix!
						},
					}
					js, err := json.Marshal(resp)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusOK)
					w.Write(js)
				}
				return
			}
		case "PUT":
			r.ParseForm()
			url := r.URL.Path

			p := map[string]interface{}{}
			err := json.NewDecoder(r.Body).Decode(&p)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			vName ,id, err := addToTable(db, strings.Trim(url, "/"), p)
			if err != nil{
				log.Fatal(err)
			}

			resp := map[string]interface{}{
				"response": map[string]interface{}{
					vName: id,
				},
			}
			js, err := json.Marshal(resp)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			w.Write(js)
			return
		case "POST":
			r.ParseForm()
			url := r.URL.Path
			u, _ := r.URL.Parse(url)
			args := strings.Split(strings.TrimLeft(u.Path, "/"),"/")
			path := args[0]
			offset , _ := strconv.Atoi(args[1])
			p := map[string]interface{}{}
			err := json.NewDecoder(r.Body).Decode(&p)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			errString := isValid(db, path, p)
			if errString != ""{
				resp := map[string]string{
					"error": errString,
				}
				js, err := json.Marshal(resp)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				w.Write(js)
				return
			}

			id, errString := updateTable(db, path, offset, p)
			if errString != ""{
				resp := map[string]string{
					"error": errString,
				}
				js, err := json.Marshal(resp)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				w.Write(js)
				return
			}

			resp := map[string]interface{}{
				"response": map[string]interface{}{
					"updated": id,
				},
			}
			js, err := json.Marshal(resp)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			w.Write(js)
			return
		case "DELETE":
			r.ParseForm()
			url := r.URL.Path
			u, _ := r.URL.Parse(url)
			args := strings.Split(strings.TrimLeft(u.Path, "/"),"/")
			path := args[0]
			offset , _ := strconv.Atoi(args[1])

			id, err := deleteTable(db, path, offset)
			if err != nil{
				resp := map[string]string{
					"error": err.Error(),
				}
				js, err := json.Marshal(resp)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				w.Write(js)
				return
			}

			resp := map[string]interface{}{
				"response": map[string]interface{}{
					"deleted": id,
				},
			}
			js, err := json.Marshal(resp)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			w.Write(js)
			return
		default:
			js, err := json.Marshal("Error: non supported method")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			w.Write(js)
		}
	}
}

func getKey(db *sql.DB, left string) (string){
	queryParams := fmt.Sprintf("SELECT * FROM %s", left)

	rows, err := db.Query(queryParams)
	if err != nil{
		return ""
	}

	defer rows.Close()

	if rows.Next() {
		columnTypes, _ := rows.ColumnTypes()

		return columnTypes[0].Name()
	}

	return ""
}

//get functions
func showQueryTable(db *sql.DB, left string, limit int, offset int) ([]map[string]interface{}, string) {

	queryParams := fmt.Sprintf("SELECT * FROM %s", left)

	rows, err := db.Query(queryParams)
	if err != nil{
		return nil, "unknown table"
	}

	defer rows.Close()

	resp := []map[string]interface{}{}
	for key := 0; rows.Next() && key != limit; key++ {
		for offset != 0{
			rows.Next()
			offset--
		}

		columnTypes, _ := rows.ColumnTypes()

		values := make([]interface{}, len(columnTypes))
		value := map[string]interface{}{}

		for i, column := range columnTypes {
			v := reflect.New(column.ScanType()).Interface()
			switch v.(type) {
			case *[]uint8:
				v = new(*string)
			case *int32:
				v = new(*int32)
			case *sql.RawBytes:
				v = new(*string)
			default:
				values[i] = v
			}

			value[column.Name()] = v
			values[i] = v
		}

		ok := rows.Scan(values...)
		if ok != nil {
			return nil, fmt.Sprintf("%s", ok)
		}
		resp = append(resp, value)
	}
	return resp, ""
}

func showTable(db *sql.DB, param string) ([]map[string]interface{}, string) {
	queryParams := fmt.Sprintf("SELECT * FROM %s", param)

	rows, err := db.Query(queryParams)
	if err != nil{
		return nil, "unknown table"
	}

	defer rows.Close()

	//https://stackoverflow.com/questions/59913936/golang-get-raw-json-from-postgres
	resp := []map[string]interface{}{}

	for key := 0; rows.Next(); key++ {
		columnTypes, _ := rows.ColumnTypes()

		values := make([]interface{}, len(columnTypes))
		value := map[string]interface{}{}

		for i, column := range columnTypes {

			v := reflect.New(column.ScanType()).Interface()
			switch v.(type) {
			case *[]uint8:
				v = new(*string)
			case *int32:
				v = new(*int32)
			case *sql.RawBytes:
				v = new(*string)
			default:
				values[i] = v
			}

			value[column.Name()] = v
			values[i] = v
		}

		ok := rows.Scan(values...)
		if ok != nil {
			return nil, fmt.Sprintf("%s", ok)
		}
		resp = append(resp, value)
	}

	return resp, ""
}

func key(db *sql.DB, tableName string) (string, error) {
	var objects []map[string]interface{}

	rows, ok := db.Query(fmt.Sprintf("SHOW FULL COLUMNS FROM %s", tableName))
	if ok != nil {
		return "", ok
	}
	defer rows.Close()

	for rows.Next() {
		columnTypes, _ := rows.ColumnTypes()
		values := make([]interface{}, len(columnTypes))
		object := map[string]interface{}{}
		for i, column := range columnTypes {
			v := reflect.New(column.ScanType()).Interface()
			switch v.(type) {
			case *[]uint8:
				v = new(*string)
			case *int32:
				v = new(*int32)
			case *sql.RawBytes:
				v = new(*string)
			default:
				values[i] = v
			}

			object[column.Name()] = v
			values[i] = v
		}

		ok := rows.Scan(values...)
		if ok != nil {
			return "", ok
		}

		objects = append(objects, object)
	}

	var key string
	for _, v := range objects {
		k := **v["Key"].(**string)
		if strings.Contains(k, "PRI") {
			s := **v["Field"].(**string)
			key = s
			break
		}
	}

	return key, nil
}

func showTables(db *sql.DB) []string {
	tables := []string{}

	rows, err := db.Query("SHOW TABLES")

	defer rows.Close()

	if err != nil{
		log.Fatal(err)
	}
	var table string

	for rows.Next(){
		rows.Scan(&table)
		tables = append(tables, table)
	}

	return tables
}

//put func
func addToTable(db *sql.DB, path string, body map[string]interface{}) (string, int, error){
	var NewBody map[string]interface{}

	NewBody = checkUnkField(db, path, body)

	NewBody = justForPut(db, path, body)

	var values	[]interface{}
	var fields, name, returnField string
	i :=0
	for k, v := range NewBody {
		//Dirty Hack. Need to parse Extra field(AutoIncrement) and increase value
		if k == "id" {
			i++
		}else{
			if i == len(NewBody)-1 {
				fields += k
				name += "?"
				values = append(values, fmt.Sprintf("%v", v))
			}else{
				values = append(values, fmt.Sprintf("%v", v))
				fields += k + ", "
				name += "?, "
				i++
			}
		}
	}
	queryParams := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", path, fields, name)

	insForm, err := db.Exec(queryParams, values...)
	if err != nil{
		return "", 0, err
	}

	id, err := insForm.LastInsertId()
	if err != nil{
		panic("Unknown error happened by LastInsertId:")
	}

	returnField = getKey(db, path)

	return returnField, int(id), nil
}

//post func
func updateTable(db *sql.DB, path string, id int, body map[string]interface{}) (int64, string) {
	keyField, err := key(db, path)

	var NewBody map[string]interface{}

	NewBody = checkUnkField(db, path, body)
	var values	[]interface{}
	var fields string
	i :=0
	for k, v := range NewBody {
		if k == keyField {
			return -1, fmt.Sprintf("field %s have invalid type", keyField)
		}
		if i == len(body)-1 {
			fields += k + " = ?"
			if v == nil{
				values = append(values, sql.NullString{})
			}else{
				values = append(values, fmt.Sprintf("%v", v))
			}
			break
		}
		if v == nil{
			values = append(values, sql.NullString{})
		}else{
			values = append(values, fmt.Sprintf("%v", v))
		}
		fields += k +  " = ? , "
		i++
	}
	values = append(values, fmt.Sprintf("%v", id))

	returnField := getKey(db, path)

	queryParams := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?", path, fields, returnField)
	row, err := db.Exec(queryParams, values...)
	if err != nil{
		panic(err)
	}

	rowsAmount, err := row.RowsAffected()
	if err != nil{
		return -1, fmt.Sprintf("%s", err)
	}
	return rowsAmount, ""
}

//check fields before post them to table
func isValid(db *sql.DB, param string, body map[string]interface{}) string{

	queryParams := fmt.Sprintf("SELECT * FROM %s", param)

	rows, err := db.Query(queryParams)
	if err != nil{
		return ""
	}
	defer rows.Close()

	if rows.Next(){
		columnType, _ := rows.ColumnTypes()

		for _, column := range columnType{
			if val, ok := body[column.Name()]; ok{
				switch column.DatabaseTypeName(){
				case "VARCHAR":
					if reflect.TypeOf(val) == nil{
						if nullable, ok := column.Nullable(); ok{
							if nullable{
								return ""
							}else{
								return fmt.Sprintf("field %v have invalid type", column.Name())
							}
						}
					}
					if reflect.TypeOf(val) != reflect.TypeOf(""){
						return fmt.Sprintf("field %v have invalid type", column.Name())
					}
				}
			}
		}
	}
	return ""
}

//delete func
func deleteTable(db *sql.DB, path string, id int) (int64, error){

	keyField, err := key(db, path)

	queryParams := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", path, keyField)

	rows, err := db.Exec(queryParams, id)
	if err != nil{
		return -1, err
	}

	affected, err := rows.RowsAffected()
	if err != nil{
		return -1, err
	}

	return affected, nil
}

//if has unknownField
func checkUnkField(db *sql.DB, param string, body map[string]interface{}) map[string]interface{}{
	queryParams := fmt.Sprintf("SELECT * FROM %s", param)

	rows, err := db.Query(queryParams)
	if err != nil{
		return nil
	}

	defer rows.Close()

	if rows.Next(){
		columnType, _ := rows.ColumnTypes()

		var allNames []string

		for _, column := range columnType{
			allNames = append(allNames, column.Name())
		}

		for k := range body{
			if !contains(allNames, k){
				delete(body, k)
			}
		}
	}
	return body
}

func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func justForPut(db *sql.DB, param string, body map[string]interface{})map[string]interface{}{
	queryParams := fmt.Sprintf("SELECT * FROM %s", param)

	rows, err := db.Query(queryParams)
	if err != nil{
		return nil
	}

	defer rows.Close()

	if rows.Next(){
		columnType, _ := rows.ColumnTypes()

		var allNames []string

		for _, column := range columnType{

			allNames = append(allNames, column.Name())
			if nullable, ok := column.Nullable(); ok{
				if !nullable{
					if _, ok := body[column.Name()]; !ok{
						body[column.Name()] = ""
					}
				}
			}
		}
	}
	return body
}


//Extra = AutoIncrement
//Type = type
//Privilages -insert, ect
//Null - no
//Comment =
//Key = PRI

//Collation-error
//Default-error
