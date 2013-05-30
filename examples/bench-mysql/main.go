package main

import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"

    "github.com/yjh0502/gb"
    "fmt"
    "hash/fnv"
    "io"
)

const query_create_table = `create table if not exists test
(
    id bigint,
    k text,
    v text,
    primary key(id)
) engine=InnoDB partition by hash(id) partitions 16;`

type mysqlBench struct {
    db *sql.DB
    stmt *sql.Stmt
    count int
}

func (b *mysqlBench) Execute() (done bool, err error) {
    b.count++
    if b.count > 1000000 {
        return true, nil
    }

    hashfunc := fnv.New64()

    k := gb.GetRandStr(10)
    v := gb.GetRandStr(100)
    io.WriteString(hashfunc, k)
    id := hashfunc.Sum64()

    _, err = b.stmt.Exec(int64(id), k, v)
    if err != nil {
        return false, fmt.Errorf("Failed to execute: %s\n", err.Error())
    }

    return false, nil
}

func benchInit() (gb.BenchmarkRunner, error) {
    var err error
    b := new(mysqlBench)

    b.db, err = sql.Open("mysql", "test:test@tcp(1.237.186.213:3306)/test?charset=utf8")
    if err != nil {
        return nil, fmt.Errorf("Failed to connect database: %s", err.Error())
    }


    if _, err = b.db.Exec(query_create_table); err != nil {
        return nil, fmt.Errorf("Failed to create table: %s", err.Error())
    }

    b.stmt, err = b.db.Prepare("insert into test(id, k, v) values (?, ?, ?)")
    if err != nil {
        return nil, fmt.Errorf("Failed to create prepared statement: %s", err.Error())
    }

    return b, nil
}

func main() {
	b := gb.NewBench()
	b.Run(benchInit)
}
