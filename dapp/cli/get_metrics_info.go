package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getMetricsInfoCommandHandler(ctx context.Context, c interface{}, flags cmdFlags) {
	var flag bool = true
	file, err := os.Create("metricsInfo_result.csv")
	if err != nil {
		log.Println(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	writer.Comma = ','
	tick := time.NewTicker(time.Duration(5000) * time.Millisecond)
	for {
		select {
		case <-tick.C:
			metricsServiceRequest := rpcpb.MetricsServiceRequest{}
			metricsInfoResponse, err := c.(rpcpb.MetricServiceClient).RpcGetMetricsInfo(ctx, &metricsServiceRequest)
			if err != nil {
				switch status.Code(err) {
				case codes.Unavailable:
					fmt.Println("Error: server is not reachable!")
				default:
					fmt.Println("Error: ", err.Error())
				}
				return
			}
			fmt.Println("metricsInfo: ", metricsInfoResponse.Data)

			m, ok := gjson.Parse(metricsInfoResponse.Data).Value().(map[string]interface{})
			if !ok {
				fmt.Println("Parsed data is not json")
				continue
			}
			var titleStr []string
			var metricsInfostr []string
			metricsInfoMap := make(map[string]string)
			for key, value := range m {
				if value != nil {
					childValue := value.(map[string]interface{})
					for cKey, cValue := range childValue {
						if cKey == "txRequestSend" || cKey == "txRequestSendFromMiner" {
							grandChildValue := cValue.(map[string]interface{})
							for gcKey, gcValue := range grandChildValue {
								if gcValue != nil {
									switch v := gcValue.(type) {
									case float64:
										metricsInfoMap[key+":"+cKey+":"+gcKey] = strconv.Itoa(int(v))
									}
								}
							}
						} else {
							switch v := cValue.(type) {
							case float64:
								metricsInfoMap[key+":"+cKey] = strconv.Itoa(int(v))
							}
						}
					}
				}
			}
			var keys []string
			for key := range metricsInfoMap {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for i := 0; i < len(keys); i++ {
				value := metricsInfoMap[keys[i]]
				metricsInfostr = append(metricsInfostr, value)
				titleStr = append(titleStr, keys[i])
			}
			var strArray [][]string
			if flag == true {
				strArray = append(strArray, titleStr)
			}
			strArray = append(strArray, metricsInfostr)
			flag = false
			writer.WriteAll(strArray)
			writer.Flush()
		}
	}
}
