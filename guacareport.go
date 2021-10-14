package main

import (
	"database/sql"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
	_ "github.com/go-sql-driver/mysql"
	"github.com/zserge/webview" //version 0.1.1
)

const (
	windowWidth  = 1200
	windowHeight = 800
)

var err error

type Table struct {
	username, remote_host, start_date, end_date, diff_date string
	select_id, history_id                                  int
}

var table []Table

type DataHTML struct {
	date   string
	filter string
	radio  string
}

func (d *DataHTML) AddDate(n string) {
	d.date = n
}

func (d *DataHTML) AddFilter(n string) {
	d.filter = n
}

func (d *DataHTML) AddRadio(n string) {
	d.radio = n
}

var dataHTML DataHTML = DataHTML{radio: "day"}

var indexHTML = `
<!doctype html>
<html>
	<head>
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
	</head>
	<body onload="external.invoke('getsql')">
	
	<table style="width: 100%;">
	<tr>
	<td align="center" style="padding-bottom: 10px;">
		
		<input name="day" type="radio" onchange="radio('day');" checked>День</input>
		<input name="day" type="radio" onchange="radio('month');">Месяц</input>
		<input id="date" placeholder="гггг-мм-дд" style="margin-left: 10px;" onkeydown="if (event.keyCode==13) {update();}"></input>
		<input id="filter_label" placeholder="Фильтр" onkeyup="filter();"></input>
		<button id="save" style="margin-left: 10px;" onclick="external.invoke('save')">Сохранить в Excel</button>
		
	</td>
	</tr>
	
	<tr>
	<td>
		<div id="div_table" style="width: 95%; height: 95%; margin-left: auto; margin-right: auto; overflow: auto;"></div>
    </td>
    </tr>
    </table>
    
    <script>
	
	function update() {
		dataHTML.addDate(document.getElementById('date').value);
		external.invoke('update');
	}
	
	function filter() {
		dataHTML.addFilter(document.getElementById('filter_label').value);
		external.invoke('update');
	}
	
	function radio(s) {
		dataHTML.addRadio(s);
		external.invoke('radio');
	}
	
	</script>
			
	</body>
</html>
`

func handleRPC(w webview.WebView, data string) {
	switch {

	case data == "getsql":

		w.Bind("dataHTML", &dataHTML)

		dataHTML.date = time.Now().Local().Format("2006-01-02")

		err = getsql(w)
		if err == nil {
			w.Eval(`document.getElementById('date').value="` + dataHTML.date + `"`)
		} else {
			w.Eval(`alert("Ошибка: ` + err.Error() + `");`)
		}

	case data == "save":
		file_path := w.Dialog(webview.DialogTypeSave, 0, "Save file", "")

		if strings.Contains(file_path, ".xls") == true {
			file_path = strings.Replace(file_path, ".xlsx", "", -1)
			file_path = strings.Replace(file_path, ".xls", "", -1)
		}

		/*if strings.Contains(file_path, ".xlsx") == false {
			file_path += ".xlsx"
		}*/

		if file_path != "" {
			if strings.Contains(file_path, ".xlsx") == false {
				file_path += ".xlsx"
			}
			err = save(file_path)
			if err != nil {
				w.Eval(`alert("Ошибка: ` + err.Error() + `");`)
			}
		}

	case data == "update":
		w.Bind("dataHTML", &dataHTML)

		if dataHTML.radio == "day" {
			if len(dataHTML.date) == 10 {
				err = getsql(w)
				if err != nil {
					w.Eval(`alert("Ошибка: ` + err.Error() + `");`)
				}
			} else {
				w.Eval(`alert("Дату необходимо ввести в формате гггг-мм-дд");`)
			}
		} else if dataHTML.radio == "month" {
			if len(dataHTML.date) == 7 {
				err = getsql(w)
				if err != nil {
					w.Eval(`alert("Ошибка: ` + err.Error() + `");`)
				}
			} else {
				w.Eval(`alert("Год и месяц необходимо ввести в формате гггг-мм");`)
			}
		}

	case data == "radio":
		w.Bind("dataHTML", &dataHTML)

		if dataHTML.radio == "day" {
			dataHTML.date = time.Now().Local().Format("2006-01-02")
			w.Eval(`document.getElementById('date').placeholder="гггг-мм-дд"`)
		} else if dataHTML.radio == "month" {
			dataHTML.date = time.Now().Local().Format("2006-01")
			w.Eval(`document.getElementById('date').placeholder="гггг-мм"`)
		}

		err = getsql(w)
		if err == nil {
			w.Eval(`document.getElementById('date').value="` + dataHTML.date + `"`)
		} else {
			w.Eval(`alert("Ошибка: ` + err.Error() + `");`)
		}

	}
}

func startServer() string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		defer ln.Close()
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(indexHTML))
		})
		log.Fatal(http.Serve(ln, nil))
	}()
	return "http://" + ln.Addr().String()
}

func main() {

	url := startServer()
	w := webview.New(webview.Settings{
		Width:                  windowWidth,
		Height:                 windowHeight,
		Title:                  "GuacaReport",
		Resizable:              true,
		URL:                    url,
		ExternalInvokeCallback: handleRPC,
	})
	defer w.Exit()

	w.Dispatch(func() {

		w.Bind("dataHTML", &dataHTML)

	})

	w.Run()

}

func getsql(w webview.WebView) error {

	var query string

	var t Table

	db, err := sql.Open("mysql", "user:password@tcp(192.168.1.1:3306)/guacamole")

	if err != nil {
		return err
	}
	defer db.Close()

	if dataHTML.radio == "day" {

		query = `(select username as Пользователь, remote_host as 'Удаленный адрес', start_date as 'Время авторизации', if(end_date is not null, end_date, ''),
       							case when end_date is not null then timediff(end_date, start_date) else timediff(now(), start_date) end as Продолжительность,
       							1 as select_id, history_id
							from guacamole_user_history
							where start_date >= '` + dataHTML.date + ` 00:00:00'
							and start_date <= '` + dataHTML.date + ` 23:59:59'
							UNION
							(select username as Пользователь, '' as 'Удаленный адрес', '' as 'Время авторизации', 'ИТОГО',
       							sec_to_time(sum(time_to_sec(case when end_date is not null then timediff(end_date, start_date) else timediff(now(), start_date) end))) as Продолжительность,
       							2 as select_id, history_id
							from guacamole_user_history
							where start_date >= '` + dataHTML.date + ` 00:00:00'
							and start_date <= '` + dataHTML.date + ` 23:59:59'
							group by username)
							order by Пользователь, select_id, history_id
							;`

	}

	if dataHTML.radio == "month" {

		query = `(select username as Пользователь, remote_host as 'Удаленный адрес', start_date as 'Время авторизации', if(end_date is not null, end_date, ''),
	       						case when end_date is not null then timediff(end_date, start_date) else timediff(now(), start_date) end as Продолжительность,
	       						1 as select_id, history_id
							from guacamole_user_history
							where start_date >= '` + dataHTML.date + `-01 00:00:00'
							and start_date <= '` + dataHTML.date + `-31 23:59:59'
							UNION
							(select username as Пользователь, '' as 'Удаленный адрес', '' as 'Время авторизации', concat('ИТОГО ', date_format(start_date, "%Y-%m-%d")) as 'Время выхода',
       							sec_to_time(sum(time_to_sec(case when end_date is not null then timediff(end_date, start_date) else timediff(now(), start_date) end))) as Продолжительность,
       							2 as select_id, max(history_id)
							from guacamole_user_history
							where start_date >= '` + dataHTML.date + `-01 00:00:00'
							and start_date <= '` + dataHTML.date + `-31 23:59:59'
							group by username, date_format(start_date, "%Y-%m-%d"))
							UNION
							(select username as Пользователь, '' as 'Удаленный адрес', '' as 'Время авторизации', 'ИТОГО ЗА МЕСЯЦ' as 'Время выхода',
       							sec_to_time(sum(time_to_sec(case when end_date is not null then timediff(end_date, start_date) else timediff(now(), start_date) end))) as Продолжительность,
       							3 as select_id, max(history_id)
							from guacamole_user_history
							where start_date >= '` + dataHTML.date + `-01 00:00:00'
							and start_date <= '` + dataHTML.date + `-31 23:59:59'
							group by username)
							order by Пользователь, history_id, select_id
							;`

	}

	rows, err := db.Query(query)

	if err != nil {
		return err
	}
	defer rows.Close()

	htmlTable := ""
	table = nil
	for rows.Next() {
		err := rows.Scan(&t.username, &t.remote_host, &t.start_date, &t.end_date, &t.diff_date, &t.select_id, &t.history_id)
		if err != nil {
			println(err)
			continue
		}

		switch {

		case dataHTML.filter == "":
			table = append(table, t)

		case strings.Contains(strings.ToLower(t.username), strings.ToLower(dataHTML.filter)) == true:
			table = append(table, t)

		}

	}

	currentDate := time.Now().Local().Format("2006-01-02")

	for i := 0; i < len(table); i++ {

		if i > 0 {

			if table[i].end_date == "" && table[i].end_date != table[i-1].end_date && dataHTML.radio == "day" && dataHTML.date == currentDate {
				htmlTable = htmlTable + "<tr style='background: lightyellow;'><td>" + table[i].username + "</td><td>" + table[i].remote_host + "</td><td>" + table[i].start_date + "</td><td>" + table[i].end_date + "</td><td>" + table[i].diff_date + "</td></tr>"
			} else {
				switch {

				case table[i].end_date == "ИТОГО" || strings.Contains(table[i].end_date, "ИТОГО "+dataHTML.date):
					htmlTable = htmlTable + "<tr style='background: lightgray;'><td>" + table[i].username + "</td><td>" + table[i].remote_host + "</td><td>" + table[i].start_date + "</td><td>" + table[i].end_date + "</td><td>" + table[i].diff_date + "</td></tr>"

				case table[i].end_date == "ИТОГО ЗА МЕСЯЦ":
					htmlTable = htmlTable + "<tr style='background: lightblue;'><td>" + table[i].username + "</td><td>" + table[i].remote_host + "</td><td>" + table[i].start_date + "</td><td>" + table[i].end_date + "</td><td>" + table[i].diff_date + "</td></tr>"

				default:
					htmlTable = htmlTable + "<tr><td>" + table[i].username + "</td><td>" + table[i].remote_host + "</td><td>" + table[i].start_date + "</td><td>" + table[i].end_date + "</td><td>" + table[i].diff_date + "</td></tr>"
				}
			}
		} else {
			switch {

			case table[i].end_date == "" && dataHTML.radio == "day" && dataHTML.date == currentDate:
				htmlTable = htmlTable + "<tr style='background: lightyellow;'><td>" + table[i].username + "</td><td>" + table[i].remote_host + "</td><td>" + table[i].start_date + "</td><td>" + table[i].end_date + "</td><td>" + table[i].diff_date + "</td></tr>"
				countOnline++

			default:
				htmlTable = htmlTable + "<tr><td>" + table[i].username + "</td><td>" + table[i].remote_host + "</td><td>" + table[i].start_date + "</td><td>" + table[i].end_date + "</td><td>" + table[i].diff_date + "</td></tr>"

			}
		}
	}

	w.Eval(`document.getElementById('div_table').innerHTML="<a id='update' href=# onclick='update();'>Обновить</></a>"`)
	w.Eval(`document.getElementById('div_table').innerHTML+="<table id='info_table' border='1' width='99%'><tr><th>Пользователь</th><th>Удаленный адрес</th><th>Время авторизации</th><th>Время выхода</th><th>Продолжительность</th></tr>` + htmlTable + `</table>";`)

	return nil

}

func save(file_path string) error {

	f := excelize.NewFile()

	f.SetCellValue("Sheet1", "A1", "Пользователь")
	f.SetCellValue("Sheet1", "B1", "Удаленный адрес")
	f.SetCellValue("Sheet1", "C1", "Время авторизации")
	f.SetCellValue("Sheet1", "D1", "Время выхода")
	f.SetCellValue("Sheet1", "E1", "Продолжительность")

	for i := 0; i < len(table); i++ {
		f.SetCellValue("Sheet1", "A"+strconv.Itoa(i+2), table[i].username)
		f.SetCellValue("Sheet1", "B"+strconv.Itoa(i+2), table[i].remote_host)
		f.SetCellValue("Sheet1", "C"+strconv.Itoa(i+2), table[i].start_date)
		f.SetCellValue("Sheet1", "D"+strconv.Itoa(i+2), table[i].end_date)
		f.SetCellValue("Sheet1", "E"+strconv.Itoa(i+2), table[i].diff_date)
	}

	f.SetColWidth("Sheet1", "A", "E", 30)

	if err := f.SaveAs(file_path); err != nil {
		return (err)
	}

	return nil

}
