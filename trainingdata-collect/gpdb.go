package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"net/http"
	"os"
	"scheduling/util"
	"strconv"
	"strings"
	"time"
)

var requestUrl string = "http://localhost:8081"
var locustUrl string = "http://10.96.30.118:8089"
var operationUrl string = "http://localhost:8080"

// 创建一个自定义的 HTTP 客户端
var httpClient *http.Client

type Endpoint struct {
	TraceID   string `json:"traceId"`
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Timestamp int64  `json:"timestamp"`
	Duration  int    `json:"duration"`
}

func CLeanData() {
	db, err := sql.Open("mysql", "root:abc123!@tcp(10.111.157.158:3306)/zipkin")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 清除数据
	db.Exec("DELETE FROM my_spans")
	//db.Exec("DELETE FROM randomdata")

	println("清理")
}
func UpdateThread(attribute string, value int) error {
	ant := 0
	for ant < 10 {
		ant += 1
		// maxThreads
		// 构建请求体
		requestBody, err := json.Marshal(map[string]interface{}{
			"type":      "write",
			"mbean":     "Tomcat:port=8081,type=Connector",
			"attribute": attribute,
			"value":     value,
		})
		if err != nil {
			fmt.Println("Error encoding JSON:", err)
			return err
		}

		// 发送 POST 请求
		url := operationUrl + "/jolokia/write/"
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
		if err != nil {
			fmt.Println("Error creating request:", err)
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error sending request:", err)
			return err
		}
		if resp.StatusCode == 503 {
			return errors.New("错误")
		}
		// 读取响应体
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			continue
		}
		println(string(body))
		defer resp.Body.Close()
		bodstr := string(body)
		// 定义一个结构体来解析JSON
		var data map[string]interface{}

		// 解析JSON字符串
		if err := json.Unmarshal([]byte(bodstr), &data); err != nil {
			fmt.Println("解析JSON失败:", err)
			continue
		}

		// 获取外层的"value"字段的值
		outerValue, ok := data["value"].(float64)
		if !ok {
			fmt.Println("无法获取外层'value'字段的值")
			continue
		}

		if int(outerValue) == value {
			return nil
		}

	}
	return nil
}

func UpdateConnection(value int) error {
	// maxThreads
	// 构建请求体
	requestBody, err := json.Marshal(map[string]interface{}{
		"type":      "write",
		"mbean":     "Tomcat:type=ThreadPool,name=\"http-bio-8081\"",
		"attribute": "maxConnections",
		"value":     value,
	})
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return err
	}

	// 发送 POST 请求
	url := operationUrl + "/jolokia/write/"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return err
	}
	defer resp.Body.Close()

	// 处理响应
	fmt.Println("Response status:", resp.Status)
	if resp.StatusCode == 503 {
		return errors.New("错误")
	}
	return nil
}
func sendOperator(operation string) error {
	// maxThreads
	// 构建请求体
	requestBody, err := json.Marshal(map[string]interface{}{
		"type":      "exec",
		"mbean":     "Tomcat:port=8081,type=Connector",
		"operation": operation,
		"arguments": []string{},
	})
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return err
	}

	// 发送 POST 请求
	url := operationUrl + "/jolokia/"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return err
	}
	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil
	}
	println(string(body))
	defer resp.Body.Close()

	// 处理响应
	fmt.Println("Response status:", resp.Status)
	return nil
}
func RestartService() {
	err := sendOperator("stop")
	if err != nil {
		println(err.Error())
		return
	}
	err = sendOperator("start")
	if err != nil {
		println(err.Error())
	}
}

func ObtainRPS() string {
	// 判断状态
	resp2, err := http.Get(locustUrl + "/stats/requests")
	if err != nil {
		return "-1"
	}
	body2, err := ioutil.ReadAll(resp2.Body)
	if err != nil {
		return "-1"
	}
	text := string(body2)
	// 使用字符串分割获取字段值
	fields := strings.Split(text, ",")
	for _, field := range fields {
		if strings.Contains(field, "total_rps") {
			parts := strings.Split(field, ":")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "-1"
}
func obtainWorks() int {
	// 发送 HTTP 请求获取 RPS
	resp, err := http.Get(locustUrl + "/stats/requests")
	if err != nil {
		fmt.Println("Error obtaining RPS:", err)
		return -1
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return -1
	}

	// 解析 JSON 数据
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return -1
	}

	// 获取 workers 数量
	workers, ok := data["workers"].([]interface{})
	if !ok {
		fmt.Println("Workers not found in response")
		return -1
	}

	return len(workers)
}

func StopLocust(client *kubernetes.Clientset) {

	count := 0
	for {
		if obtainWorks() != 10 {
			util.DeletePodOfDeployment(client, "locust", "testthread")
		}
		time.Sleep(5 * time.Second)
		if count > 10 {
			println("停止失败")
			break
		}
		count += 1
		println("停止中")
		resp, err := httpClient.Get(locustUrl + "/stop")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		_, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == 200 {
			rps := ObtainRPS()
			println(rps)
			if rps == "0" || rps == "0.0" {
				break

			} else {
				StartLocust(1, 1)

				httpClient.Get(locustUrl + "/stop")
				//StopLocust()
				//if ObtainRPS() == "0" {
				//	break
				//}
				if obtainWorks() != 10 {
					util.DeletePodOfDeployment(client, "locust", "testthread")
				}
				time.Sleep(5 * time.Second)

			}
		}
		time.Sleep(1 * time.Second)
	}
	println("停止成功")

}
func StartLocust(spawn_rate int, user_count int) {
	url := locustUrl + "/swarm"
	method := "POST"

	payload := strings.NewReader("user_count=" + strconv.Itoa(user_count) + "&spawn_rate=" + strconv.Itoa(spawn_rate) + "&host=" + requestUrl)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("User-Agent", "Apifox/1.0.0 (https://apifox.com)")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
	println("启动成功")
}
func WriteCsv(data [][]string) {
	// 创建 CSV 文件
	file, err := os.Create("output2-1.csv")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// 创建 CSV Writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入 CSV 标题行
	headers := []string{"users", "threads", "gp", "all", "qos"}
	writer.Write(headers)

	// 将数据写入 CSV 文件
	for _, row := range data {
		err := writer.Write(row)
		if err != nil {
			fmt.Println("Error writing row:", err)
			return
		}
	}

	fmt.Println("CSV 文件已成功创建！")
}
func WriteCsv2(data [][]string) {
	// 创建 CSV 文件
	file, err := os.Create("output2-2.csv")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// 创建 CSV Writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入 CSV 标题行
	headers := []string{"users", "threads", "gp", "all"}
	writer.Write(headers)

	// 将数据写入 CSV 文件
	for _, row := range data {
		err := writer.Write(row)
		if err != nil {
			fmt.Println("Error writing row:", err)
			return
		}
	}

	fmt.Println("CSV 文件已成功创建！")
}

// 获取某个pod
func obtainPodIp(client *kubernetes.Clientset) string {
	for {
		pod, err := util.ObtainPodByName("testone-pod", "testthread", client)
		if err == nil {
			if pod.Status.PodIP != "" {
				println(pod.Status.PodIP)
				return pod.Status.PodIP
			}

		}
		time.Sleep(1 * time.Second)
	}

	return ""
}
func ObtainRequestSituation2(t int64, qos float64) (int, int) {
	//http://localhost:9411/zipkin/api/v2/traces?endTs=1709820692280&lookback=200&limit=10
	queryUrl := fmt.Sprintf("http://10.108.85.190:9411/zipkin/api/v2/traces?endTs=%d&lookback=1500&limit=15000", t)

	//url := "http://localhost:9411/zipkin/api/v2/traces?endTs=" + strconv.FormatInt(time.Now().UnixMilli(), 10) + "&lookback=500&limit=15000"
	println(queryUrl) // 发起 GET 请求
	response, err := http.Get(queryUrl)
	if err != nil {
		fmt.Println("Error:", err)
		return 0, 0
	}
	defer response.Body.Close()

	// 读取响应的内容
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return 0, 0
	}

	// 打印响应内容

	var endpoints [][]Endpoint
	err = json.Unmarshal([]byte(string(body)), &endpoints)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return 0, 0
	}
	all := 0
	res := 0
	// 打印解析后的数据
	for _, endpoint := range endpoints {
		for _, en := range endpoint {
			if float64(en.Duration)/1000 <= qos {
				res += 1
			}
			all += 1
		}
	}
	return res, all
}
func ObtainRequestSituationDB(pt int64, t int64, qos float64) (int, int) {
	dbQuery := fmt.Sprintf("select duration from my_spans where start_ts <=%d and start_ts>=%d", t, t-1000)
	// 设置MySQL连接信息
	db, err := sql.Open("mysql", "root:abc123!@tcp(10.111.157.158:3306)/zipkin")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 执行查询语句
	rows, err := db.Query(dbQuery)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	all := 0
	res := 0
	// 遍历结果集并输出每一行数据
	for rows.Next() {
		var duration string
		if err := rows.Scan(&duration); err != nil {
			log.Fatal(err)
		}
		f, err := strconv.ParseFloat(duration, 64)
		if err != nil {
			fmt.Println("转换失败:", err)
			continue
		}
		if f <= qos {
			res += 1
		}
		all += 1
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	return res, all
}

func main() {
	httpClient = &http.Client{
		Timeout: 5 * time.Second, // 设置超时时间为 5 秒
	}
	client := util.GenerateClient()
	// 1. 首先启动一个测试
	// 2. 识别关键路劲和关键服务
	// 3. 从1个到10个pod，分别获取不同pod配置的下的GP响应
	// 4. 分别记录当前启动的参数，总请求数量，不同请求数量记录到一个文件中
	// 在给定的负载下面，探究pod个数与gp的关系
	var data [][]string
	//var data2 [][]string

	preTime := time.Now().UnixMilli()

	for i := 50; i < 301; i += 20 {
		var startTime []int64
		var threadNum []int
		//lastTime := time.Now().UnixMilli()
		//startOneTime := time.Now().UnixMilli()
		for {
			name, err := util.ObtainPodByName("testone-pod", "testthread", client)
			println(name)
			if err != nil {
				break
			}
		}
		file := util.CreatePodTemplateByFile("D:\\项目\\goland\\scheduling\\pod.yaml", fmt.Sprintf("%dm", 4000), client)
		pod, err := client.CoreV1().Pods(file.ObjectMeta.Namespace).Create(context.TODO(), &v1.Pod{
			ObjectMeta: file.ObjectMeta,
			Spec:       file.Spec,
		}, metav1.CreateOptions{})
		if err != nil {
			panic(err.Error())
		}

		fmt.Printf("Pod %s created\n", pod.Name)
		ip := obtainPodIp(client)
		//requestUrl = "http://10.102.181.250:8085"
		requestUrl = "http://" + ip + ":8081"
		operationUrl = "http://" + ip + ":8084"

		for {
			if obtainWorks() != 10 {
				util.DeletePodOfDeployment(client, "locust", "testthread")
				util.DeletePodOfDeployment(client, "locust-slave", "testthread")

			} else {
				break
			}
			time.Sleep(10 * time.Second)
		}
		StartLocust(i, i)
		for {

			err := UpdateConnection(10000)
			if err == nil {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		for j := 0; j <= 200; j += 1 {

			UpdateThread("maxThreads", j+1)

			for k := 0; k < 10; k++ {
				time.Sleep(100 * time.Millisecond)
				nowTimestamp := time.Now().UnixMilli()
				startTime = append(startTime, nowTimestamp)
				threadNum = append(threadNum, j)
			}
		}

		StopLocust(client)
		time.Sleep(3 * time.Minute)
		tmpPt := preTime
		for k, t := range startTime {
			gp2, all2 := ObtainRequestSituationDB(tmpPt, t, 400)
			fmt.Printf("通过db查询：thread--%d,gp--%d，all--%d\n", threadNum[k], gp2, all2)
			data = append(data, []string{
				strconv.Itoa(i + 1), strconv.Itoa(threadNum[k]), strconv.Itoa(gp2), strconv.Itoa(all2), "400",
			})
			tmpPt = t

		}
		util.DeletePodByName("testone-pod", "testthread", client)

		WriteCsv(data)

		CLeanData()
	}

}
