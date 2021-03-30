package main

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func EnvString(key string, _default string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return _default
	}
	return val
}

type Entry struct {
	StartTime    string
	UserHost     string
	QueryTime    string
	LockTime     string
	RowsSent     string
	RowsExamined string
	Db           string
	LastInsertId string
	InsertId     string
	ServerId     string
	SqlText      string
	ThreadId     string
}

func main() {
	dsn := EnvString("MYSQL_DSN", "slowlog:slowlog@tcp(127.0.0.1:3306)/mysql")
	db, err := sql.Open("mysql", dsn)
	PanicIf(err)
	defer db.Close()

	tick := time.NewTicker(15 * time.Second)
	query := `
		SELECT start_time, user_host, query_time, lock_time, rows_sent, rows_examined, sql_text
		FROM slow_log
		ORDER BY start_time DESC;
	`
	manySpaces := regexp.MustCompile("\\s{2,}")

	for {
		select {
		case <-tick.C:
			res, err := db.Query(query)
			PanicIf(err)

			// We might lose some entries here and there, but we should catch the majority
			deleteQuery := `TRUNCATE slow_log;`
			_, err = db.Exec(deleteQuery)
			PanicIf(err)

			var e Entry
			for res.Next() {
				err = res.Scan(&e.StartTime, &e.UserHost, &e.QueryTime, &e.LockTime, &e.RowsSent, &e.RowsExamined, &e.SqlText)
				PanicIf(err)
				e.SqlText = string(manySpaces.ReplaceAll([]byte(e.SqlText), []byte(" ")))

				userHost := strings.Split(e.UserHost, "@")
				for i, v := range userHost {
					userHost[i] = strings.TrimSpace(v)
				}
				userHost[0] = strings.Split(userHost[0], "[")[0]
				userHost[1] = strings.Trim(userHost[1], "[]")

				st, err := time.Parse("2006-01-02 15:04:05.999999", e.StartTime)
				PanicIf(err)
				e.StartTime = fmt.Sprintf("%d", st.UnixNano())

				e.QueryTime = durationAsMill(e.QueryTime)
				e.LockTime = durationAsMill(e.LockTime)

				e.SqlText = strings.ReplaceAll(e.SqlText, `"`, `\"`)

				out := fmt.Sprintf(`service=mysql start_time=%s user=%s host=%s query_time=%s lock_time=%s rows_sent=%s rows_examined=%s sql_text="%s"`, e.StartTime, userHost[0], userHost[1], e.QueryTime, e.LockTime, e.RowsSent, e.RowsExamined, e.SqlText)
				fmt.Println(out)
			}
		}
	}
}

func durationAsMill(in string) string {
	var durMs int64

	qt := strings.Split(in, ":")
	hours, err := strconv.ParseInt(qt[0], 10, 0)
	PanicIf(err)
	durMs += hours * 3600 * 1000
	mins, err := strconv.ParseInt(qt[1], 10, 0)
	PanicIf(err)
	durMs += mins * 60 * 1000
	seconds, err := strconv.ParseFloat(qt[2], 0)
	PanicIf(err)
	durMs += int64(seconds * 1000)

	return fmt.Sprintf("%d", durMs)
}

func PanicIf(err error) {
	if err != nil {
		panic(err)
	}
}
