# generate-go-code-by-db

## Overview

* This is a tool application that generates golang source code in units of tables by reading the database table structure
* It can automatically generate database table mapping and related CRUD codes
* The generated .go file will be named after the table name
* If the file already exists will not overwrite
* It can reduce the repetitive work of writing database table struct code to a certain extent
* This tool supports MySQL or PostgreSQL 
* I will continue to optimize and make it support more databases
* Hope you can make some suggestions to make it better

## How to use

* You need to set the configuration of the database in the main function, and specify the target path for code generation
* For example:

```go
package main

func main() {
	dbType := "mysql"
	host := "0.0.0.0"
	port := "3306"
	user := "root"
	password := "123456"
	database := "test"
	// You can specify the name of the table that needs to be read, if not filled it will read all database tables
	tables := []string{"test"} 
	// Notice: this path must exist
	// Ex. usr/workspace/model
	path := "usr/workspace/model" 
}
```

* If you think that the automatically generated code cannot meet your development needs
* you can also directly edit the `template.txt` file to control the content of the code generation