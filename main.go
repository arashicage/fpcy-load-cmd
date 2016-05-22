package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"common/goracle"
	"common/goracle/connect"
	"common/ini"
	"common/utils"
	"github.com/garyburd/redigo/redis"
)

var logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)
var version = "version info : 1.0.0.0 \n" + "compiled with:" + " go1.5.4.linux-amd64"

func main() {

	fmt.Println(version)

	f := "load.conf"

	if len(os.Args) > 1 {
		f = os.Args[1]
	}

	cfg := ini.DumpAll(f)

	proxy := make(map[string]string)
	proxy["01"] = cfg["proxy:"+"01"]
	proxy["02"] = cfg["proxy:"+"02"]
	proxy["03"] = cfg["proxy:"+"03"]
	proxy["04"] = cfg["proxy:"+"04"]
	proxy["05"] = cfg["proxy:"+"05"]
	proxy["06"] = cfg["proxy:"+"06"]
	proxy["07"] = cfg["proxy:"+"07"]
	proxy["08"] = cfg["proxy:"+"08"]
	proxy["09"] = cfg["proxy:"+"09"]
	proxy["10"] = cfg["proxy:"+"10"]
	proxy["11"] = cfg["proxy:"+"11"]
	proxy["12"] = cfg["proxy:"+"12"]

	uid := cfg["DEFAULT:"+"uid"]
	sql := cfg["options:"+"sql"]
	col := cfg["options:"+"col"]
	key := cfg["options:"+"key"]
	del := cfg["options:"+"del"]
	batch, err := strconv.Atoi(cfg["DEFAULT:"+"batch"])
	if err != nil {
		logger.Println("无效的批量大小")
		return
	}

	// 获取数据加载的配置信息 fplx sjlx sql_text col
	conf, err := goracle.DumpTable(uid, sql, col, key, del)
	if err != nil {
		return
	}

	tasks, err := goracle.DumpTable(cfg["DEFAULT:"+"uid"], cfg["tasks:"+"sql"], cfg["tasks:"+"col"], cfg["tasks:"+"key"], cfg["tasks:"+"del"])
	if err != nil {
		return
	}

	logger.Println("待加载任务数量:", len(tasks))

	for _, v := range tasks {

		loadParallel(conf, v, proxy, uid, batch)

	}

}

func loadParallel(conf map[string]map[string]string, v map[string]string, proxy map[string]string, uid string, batch int) error {

	taskSQL := conf[v["fplx_dm"]+"_"+v["sjlx_dm"]]["sql_text"] + " where kpyf = '" + v["kpyf"] + "' and tslsh='" + v["tslsh"] + "'"

	logger.Println("------------------ 任务信息 ------------------")
	logger.Println("推送流水:", v["tslsh"])
	logger.Println("开票月份:", v["kpyf"])
	logger.Println("发票类型:", v["fplx_dm"])
	logger.Println("数据类型:", v["sjlx_dm"])
	logger.Println("记录数量:", v["fpsl"])

	// 通过 taskSQL 取出数据, 根据 taskType 决定加载什么, 调用加载,日志写到 taskName 中
	url := proxy[utils.SubStr(v["kpyf"], 4, 2)]
	logger.Println("目标代理:", url)

	err := loadTask(uid, taskSQL, conf[v["fplx_dm"]+"_"+v["sjlx_dm"]]["cols"], conf[v["fplx_dm"]+"_"+v["sjlx_dm"]]["sjlx_dm"], url, batch)
	if err != nil {
		updTask(uid, "update fpcy_sjjzjk set jzzt_dm='3',jzsj=sysdate,servername='x' where 1 = 1 "+" and tslsh = "+v["tslsh"]+" and fplx_dm = "+v["fplx_dm"]+" and sjlx_dm = "+v["sjlx_dm"]+" and kpyf = "+v["kpyf"]+" and mode_ = "+v["mode_"])
		return err
	} else {
		updTask(uid, "update fpcy_sjjzjk set jzzt_dm='2',jzsj=sysdate,servername='x' where 1 = 1 "+" and tslsh = "+v["tslsh"]+" and fplx_dm = "+v["fplx_dm"]+" and sjlx_dm = "+v["sjlx_dm"]+" and kpyf = "+v["kpyf"]+" and mode_ = "+v["mode_"])
	}

	logger.Println("")

	return nil
}

func updTask(uid, sql string) error {

	conn, err := connect.GetRawConnection(uid)
	if err != nil {
		logger.Printf("连接数据库发生错误.\n连接信息为: %s\n错误信息为: %s", uid, strings.Split(err.Error(), "\n")[1])
		return err
	}
	defer conn.Close()

	cur := conn.NewCursor()
	defer cur.Close()

	err = cur.Execute(sql, nil, nil)
	if err != nil {
		logger.Printf("执行sql 语句发生错误.\nsql 语句为: %s\n错误信息为: %s", sql, strings.Split(err.Error(), "\n")[1])
		return err
	}

	err = cur.Execute("commit", nil, nil)
	if err != nil {
		logger.Printf("执行sql 语句发生错误.\nsql 语句为: %s\n错误信息为: %s", sql, strings.Split(err.Error(), "\n")[1])
		return err
	}

	return nil
}

func loadTask(uid, sql, cols, taskType, url string, batch int) error {

	conn, err := connect.GetRawConnection(uid)
	if err != nil {
		logger.Printf("连接数据库发生错误.\n连接信息为: %s\n错误信息为: %s", uid, strings.Split(err.Error(), "\n")[1])
		return err
	}
	defer conn.Close()

	cur := conn.NewCursor()
	defer cur.Close()

	err = cur.Execute(sql, nil, nil)

	// logger.Println(sql)
	if err != nil {
		logger.Printf("执行sql 语句发生错误.\nsql 语句为: %s\n错误信息为: %s", sql, strings.Split(err.Error(), "\n")[1])
		return err
	}

	// 获取sql 的列别名
	columns, err := goracle.GetColumns(cur)
	if err != nil {
		logger.Printf("获取列信息发生错误.\n错误信息为: %s", strings.Split(err.Error(), "\n")[1])
		return err
	}

	// records 为全部记录 records[i][j]=v
	records, err := cur.FetchMany(batch) //[][]interface{}
	if err != nil {
		logger.Printf("加载失败.\n错误信息为: %s", err.Error())
		return err
	}

	for err == nil && len(records) > 0 {

		if taskType == "01" {

			err = LoadingFPXX(records, url, strings.Split(cols, ","), columns)
			if err != nil {
				break
			}
		} else if taskType == "02" {
			err = LoadingHWXX(records, url, strings.Split(cols, ","), columns)
			if err != nil {
				break
			}
		}

		records, err = cur.FetchMany(batch)
		if err != nil {
			logger.Printf("加载失败.\n错误信息为: %s", err.Error())
			return err
		}
	}
	if err != nil {
		return err
	}

	return nil
}

func LoadingFPXX(records [][]interface{}, url string, fields []string, columns []goracle.Column) error {

	N := len(columns)

	rs, err := redis.Dial("tcp", url)
	if err != nil {
		logger.Printf("连接 redis 发生错误", err, url)
		return err
	}

	for _, row := range records { //每行
		argx := make([]interface{}, N*2, N*2)
		idx := 0
		for i, col := range row { //每列
			r := ""
			if col == nil {
				r = ""
			} else {
				r = strings.TrimSuffix(strings.TrimPrefix(columns[i].String(col), `"`), `"`)
			}
			argx[idx] = fields[i] //fieldnames
			idx++
			argx[idx] = r //field value
			idx++

		}
		//    1                            2n-2   2n-1
		//key xx field1 xx1 field2 xx2 ... expire nnn
		rs.Send("HMSET", argx[1:N*2-2]...)
		rs.Send("EXPIRE", argx[1], argx[N*2-1])
	}

	err = rs.Flush()
	if err != nil {
		logger.Printf("ERROR flush 出错\n", err)
		return err
	} else {
		for i := 0; i < len(records)*2; i++ {
			_, err = rs.Receive()
			if err != nil {
				break
			}
		}
		if err != nil {
			logger.Println("┌━ 本次 pipeline:", len(records))
			logger.Printf("└━ 获取 reply FAIL.本任务加载失败.错误信息:\n%s", err)
		} else {
			logger.Println("┌━ 本次 pipeline:", len(records))
			logger.Printf("└━ 获取 reply OK.本批次加载成功.\n")
		}

	}

	rs.Close()
	return err

}

func LoadingHWXX(records [][]interface{}, url string, fields []string, columns []goracle.Column) error {

	N := len(columns)

	rs, err := redis.Dial("tcp", url)
	if err != nil {
		logger.Printf("连接 redis 发生错误", err, url)
		return err
	}

	for _, row := range records { //每行
		argx := make([]interface{}, N, N)
		idx := 0
		for i, col := range row { //每列
			r := ""
			if col == nil {
				r = ""
			} else {
				r = strings.TrimSuffix(strings.TrimPrefix(columns[i].String(col), `"`), `"`)
			}
			argx[idx] = r //field value
			idx++

		}

		//key val
		rs.Send("HMSET", argx...)
	}

	err = rs.Flush()
	if err != nil {
		logger.Printf("ERROR flush 出错\n", err)
		return err
	} else {
		for i := 0; i < len(records); i++ {
			_, err = rs.Receive()
			if err != nil {
				break
			}
		}
		if err != nil {
			logger.Println("┌━ 本次 pipeline:", len(records))
			logger.Printf("└━ 获取 reply FAIL.本任务加载失败错误信息:\n%s", err)
		} else {
			logger.Println("┌━ 本次 pipeline:", len(records))
			logger.Printf("└━ 获取 reply OK.本批次加载成功.\n")
		}

	}

	rs.Close()
	return err
}
