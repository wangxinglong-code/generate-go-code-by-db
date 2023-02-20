package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
)

const (
	DB_TYPE_MYSQL      = "mysql"
	DB_TYPE_POSTGRESQL = "postgres"
)

func main() {
	// #################### Config #####################
	dbType := DB_TYPE_MYSQL
	host := "0.0.0.0"
	port := "3306"
	user := "root"
	password := "123456"
	database := "test"
	tables := []string{"test"}
	// set path
	// Ex. usr/workspace/model
	path := "usr/workspace/model"
	// #################### Config #####################

	dbType = strings.ToLower(dbType)
	db, err := initDbConn(dbType, host, port, user, password, database)
	if err != nil {
		fmt.Println("init db connection failed, ", err.Error())
		return
	}

	list := readDb(dbType, database, tables, db)

	result := create(dbType, path, list)

	fmt.Println(result)
	return
}

func initDbConn(dbType, host, port, user, password, database string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error
	switch dbType {
	case DB_TYPE_MYSQL:
		db, err = gorm.Open(mysql.Open(fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, database)))
	case DB_TYPE_POSTGRESQL:
		db, err = gorm.Open(postgres.Open(fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, database)))
	}
	if err != nil {
		return nil, err
	}
	return db, nil
}

type Base struct {
	TableName    string `json:"table_name" gorm:"table_name"`
	TableComment string `json:"table_comment" gorm:"table_comment"`
	FieldName    string `json:"field_name" gorm:"field_name"`
	FieldType    string `json:"field_type" gorm:"field_type"`
	FieldComment string `json:"field_comment" gorm:"field_comment"`
	NotNull      string `json:"not_null" gorm:"not_null"`
}

type FieldBase struct {
	TableComment string `json:"table_comment" gorm:"table_comment"`
	FieldName    string `json:"field_name" gorm:"field_name"`
	FieldType    string `json:"field_type" gorm:"field_type"`
	FieldComment string `json:"field_comment" gorm:"field_comment"`
	NotNull      string `json:"not_null" gorm:"not_null"`
}

func readDb(dbType, database string, tables []string, db *gorm.DB) []*Base {
	list := []*Base{}
	switch dbType {
	case DB_TYPE_MYSQL:
		list = readMySQL(database, tables, db)
	case DB_TYPE_POSTGRESQL:
		list = readPostgreSQL(database, tables, db)
	}
	return list
}

func readPostgreSQL(database string, tables []string, db *gorm.DB) []*Base {
	var list []*Base
	sql := `SELECT c.relname as table_name,cast(obj_description(relfilenode,'pg_class') as varchar) as table_comment,a.attname AS field_name,t.typname AS field_type,a.attnotnull AS not_null,b.description AS field_comment
	FROM pg_class c,pg_attribute a LEFT JOIN pg_description b ON a.attrelid = b.objoid AND a.attnum = b.objsubid, pg_type t 
	WHERE c.relkind = 'r' and c.relname not like 'pg_%' and c.relname not like 'sql_%' AND a.attnum > 0 AND a.attrelid = c.oid AND a.atttypid = t.oid`
	if len(tables) > 0 {
		sql += " AND c.relname in ('" + strings.Join(tables, "','") + "')"
	}
	db.Raw(sql).Scan(&list)
	return list
}

func readMySQL(database string, tables []string, db *gorm.DB) []*Base {
	var list []*Base
	sql := `SELECT t.table_name,t.table_comment,c.column_name as field_name,c.data_type as field_type,c.column_comment as field_comment 
	FROM information_schema.COLUMNS c,information_schema.TABLES t 
	WHERE c.TABLE_NAME = t.TABLE_NAME AND t.TABLE_SCHEMA = '` + database + "'"
	if len(tables) > 0 {
		sql += " AND t.table_name in ('" + strings.Join(tables, "','") + "')"
	}
	db.Raw(sql).Scan(&list)
	return list
}

func create(dbType, path string, list []*Base) string {
	if !isExist(path) {
		return "file create failed, directory does not exist"
	}

	tableMap := make(map[string][]*FieldBase)
	for _, i := range list {
		tableMap[i.TableName] = append(tableMap[i.TableName], &FieldBase{
			TableComment: i.TableComment,
			FieldName:    i.FieldName,
			FieldType:    i.FieldType,
			FieldComment: i.FieldComment,
			NotNull:      i.NotNull,
		})
	}

	for k, v := range tableMap {
		tableNameTitle := getTitle(k)
		filePath := filepath.Join(path, k) + ".go"
		if !isExist(filePath) {

			// create .go file
			_, err := os.Create(filePath)
			if err != nil {
				fmt.Println(filePath, " create failed, error: ", err.Error())
				continue
			}

			// create struct
			tableStruct := getTableStruct(v, dbType, tableNameTitle, v[0].TableComment)

			// get template
			// template, err := getTableTemplate()
			// if err != nil {
			// 	fmt.Println(filePath, " create failed, get table template failed, error: ", err.Error())
			// 	continue
			// }

			// set template
			template = setTableTemplate(template, getPackageName(path), tableStruct, tableNameTitle, k)
			if template == "" {
				fmt.Println(filePath, " create failed, set table template failed")
				continue
			}

			// write .go file
			err = write(filePath, template)
			if err != nil {
				fmt.Println(filePath, " create failed, write file failed, error: ", err.Error())
				continue
			}
		}
	}
	return "success"
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func getTableStruct(list []*FieldBase, dbType, tableNameTitle, tableComment string) string {
	fieldContent := ""
	for _, i := range list {
		fieldContent += getTitle(i.FieldName) + " " + getFieldType(dbType, i.FieldType) + " " + getFieldTag(i.FieldName) + " //" + i.FieldComment + "\n"
	}
	tableStruct := fmt.Sprintf(`
				// %s
				type %s struct {
					%s}
			`, tableComment, tableNameTitle, fieldContent)
	return tableStruct
}

func getFieldType(dbType, fieldType string) string {
	switch dbType {
	case DB_TYPE_MYSQL:
		if v, ok := MysqlFieldType2GoType[fieldType]; ok {
			return v
		}
	case DB_TYPE_POSTGRESQL:
		if v, ok := PostgreSqlFieldType2GoType[fieldType]; ok {
			return v
		}
	}
	return "string"
}

func getFieldTag(fieldName string) string {
	return fmt.Sprintf("`json:\"%s\" gorm:\"column:%s\"`", fieldName, fieldName)
}

// func getTableTemplate() (string, error) {
// 	wd, _ := os.Getwd()
// 	path := filepath.Join(wd, "template.txt")
// 	f, err := ioutil.ReadFile(path)
// 	if err != nil {
// 		path = filepath.Join(getCurrentAbPath(), "template.txt")
// 		f, err = ioutil.ReadFile(path)
// 	}
// 	return string(f), err
// }

func setTableTemplate(template string, packageName, tableStruct, tableNameTitle, tableName string) string {
	template = strings.ReplaceAll(template, "{package_name}", packageName)
	template = strings.ReplaceAll(template, "{table_struct}", tableStruct)
	template = strings.ReplaceAll(template, "{table_name_title}", tableNameTitle)
	template = strings.ReplaceAll(template, "{table_name}", tableName)
	return template
}

func write(path, template string) error {
	return ioutil.WriteFile(path, []byte(template), 0777)
}

// func getCurrentAbPath() string {
// 	dir := getCurrentAbPathByExecutable()
// 	if strings.Contains(dir, getTmpDir()) {
// 		return getCurrentAbPathByCaller()
// 	}
// 	return dir
// }

// func getTmpDir() string {
// 	dir := os.Getenv("TEMP")
// 	if dir == "" {
// 		dir = os.Getenv("TMP")
// 	}
// 	res, _ := filepath.EvalSymlinks(dir)
// 	return res
// }

// func getCurrentAbPathByExecutable() string {
// 	exePath, err := os.Executable()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	res, _ := filepath.EvalSymlinks(filepath.Dir(exePath))
// 	return res
// }

// func getCurrentAbPathByCaller() string {
// 	var abPath string
// 	_, filename, _, ok := runtime.Caller(0)
// 	if ok {
// 		abPath = path.Dir(filename)
// 	}
// 	return abPath
// }

func getPackageName(path string) string {
	slice := strings.Split(path, string(os.PathSeparator))
	length := len(slice)
	return slice[length-1]
}

func getTitle(name string) string {
	title := ""
	for _, i := range strings.Split(name, "_") {
		title += strings.Title(i)
	}
	return title
}

var MysqlFieldType2GoType = map[string]string{
	"int":                "int",
	"integer":            "int",
	"tinyint":            "int8",
	"smallint":           "int16",
	"mediumint":          "int32",
	"bigint":             "int64",
	"int unsigned":       "uint",
	"integer unsigned":   "uint",
	"tinyint unsigned":   "uint8",
	"smallint unsigned":  "uint16",
	"mediumint unsigned": "uint32",
	"bigint unsigned":    "uint64",
	"bit":                "string",
	"bool":               "bool",
	"enum":               "string",
	"set":                "string",
	"varchar":            "string",
	"char":               "string",
	"tinytext":           "string",
	"mediumtext":         "string",
	"text":               "string",
	"longtext":           "string",
	"blob":               "string",
	"tinyblob":           "string",
	"mediumblob":         "string",
	"longblob":           "string",
	"date":               "time.Time",
	"datetime":           "time.Time",
	"timestamp":          "time.Time",
	"time":               "time.Time",
	"float":              "float64",
	"double":             "float64",
	"decimal":            "float64",
	"binary":             "string",
	"varbinary":          "string",
}

var PostgreSqlFieldType2GoType = map[string]string{
	"bit":           "string",
	"bool":          "bool",
	"box":           "string",
	"bytea":         "string",
	"char":          "string",
	"cidr":          "string",
	"circle":        "string",
	"date":          "time.Time",
	"decimal":       "float64",
	"float4":        "float32",
	"float8":        "float64",
	"inet":          "string",
	"int2":          "int16",
	"int4":          "int32",
	"int8":          "int64",
	"interval":      "string",
	"json":          "string",
	"jsonb":         "string",
	"line":          "string",
	"lseg":          "string",
	"macaddr":       "string",
	"money":         "float64",
	"numeric":       "float64",
	"path":          "string",
	"point":         "string",
	"polygon":       "string",
	"serial2":       "int16",
	"serial4":       "int32",
	"serial8":       "int64",
	"text":          "string",
	"time":          "time.Time",
	"timestamp":     "time.Time",
	"timestamptz":   "time.Time",
	"timetz":        "time.Time",
	"tsquery":       "string",
	"tsvector":      "string",
	"txid_snapshot": "string",
	"uuid":          "string",
	"varbit":        "string",
	"varchar":       "string",
	"xml":           "string",
}

var template = `
package {package_name}

{table_struct}

func (table *{table_name_title}) TableName() string {
	return "{table_name}"
}

func (table *{table_name_title}) Get(id int) (*{table_name_title}, error) {
	var m {table_name_title}
	return &m, nil
}

func (table *{table_name_title}) List() ([]*{table_name_title}, int64, error) {
	var list []*{table_name_title}
	var count int64
	var err error
	return list, count, err
}

func (table *{table_name_title}) Save(m *{table_name_title}) error {
	return nil
}

func (table *{table_name_title}) Delete(m *{table_name_title}) error {
	return nil
}

`
