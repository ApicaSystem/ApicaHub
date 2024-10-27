package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func main() {

	if len(os.Args) < 4 {
		fmt.Println("Error: Please provide input, output filenames and title as arguments.")
		os.Exit(1)
	}

	file1 := os.Args[1]
	file2 := os.Args[2]
	title := os.Args[3]

	err := checkFileExists(file1)
	if err != nil {
		fmt.Println("Error: Input file does not exist.")
		os.Exit(1)
	}

	convertToDataExplorer(file1, file2, title)
}

func checkFileExists(filename string) error {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return err
	}
	return nil
}

func convertToDataExplorer(inputFile string, outputFile string, dashboardTitle string) {
	inputData, _ := os.ReadFile(inputFile)
	var d map[string]interface{}
	json.Unmarshal(inputData, &d)

	outputJson := make(map[string]interface{})
	header := make(map[string]interface{})
	outputJson["header"] = header
	header["dateTimeRange"] = true
	header["dropdowns"] = []interface{}{}

	tabs := []interface{}{}
	tab := make(map[string]interface{})
	tabs = append(tabs, tab)
	tab["key"] = dashboardTitle
	tab["order"] = "1"
	tab["title"] = dashboardTitle
	tab["type"] = "metrics"

	order := 1

	queryList := []map[string]interface{}{}
	for _, row := range d["rows"].([]interface{}) {
		for _, panel := range row.(map[string]interface{})["panels"].([]interface{}) {
			for _, target := range panel.(map[string]interface{})["targets"].([]interface{}) {
				title := ""

				if len(panel.(map[string]interface{})["targets"].([]interface{})) > 1 {
					legendSplit := strings.Split(target.(map[string]interface{})["legendFormat"].(string), " - ")
					title = panel.(map[string]interface{})["title"].(string) + " - " + legendSplit[len(legendSplit)-1] + " query"
				} else {
					title = panel.(map[string]interface{})["title"].(string) + " query"
				}

				query := make(map[string]interface{})
				query["name"] = title
				if panel.(map[string]interface{})["lines"] != nil && panel.(map[string]interface{})["lines"].(bool) {
					query["chart_type"] = "line"
				}
				query["data_source_name"] = "Apica Monitoring" //

				options := make(map[string]interface{})
				query["options"] = options
				options["description"] = title
				options["order"] = order
				order += 1
				plot := make(map[string]interface{})
				options["plot"] = plot
				plot["errorColumn"] = ""
				plot["groupBy"] = ""
				plot["x"] = "Timestamp"
				plot["xLabel"] = "Timestamp"
				plot["y"] = []string{"value"}
				plot["yLabel"] = "value"
				options["upperLimit"] = ""
				queryExpr := target.(map[string]interface{})["expr"].(string)
				queryExpr = strings.ReplaceAll(queryExpr, "$namespace", "{{namespace}}")
				if queryExpr != strings.ReplaceAll(queryExpr, ",service=~\"$service\"", "") {
					queryExpr = strings.ReplaceAll(queryExpr, ",service=~\"$service\"", "")
					queryExpr = strings.ReplaceAll(queryExpr, ", quantile=", " quantile=")
					queryExpr = strings.ReplaceAll(queryExpr, ",quantile=", "quantile=")
				} else if queryExpr != strings.ReplaceAll(queryExpr, ", service=~\"$service\"", "") {
					queryExpr = strings.ReplaceAll(queryExpr, ", service=~\"$service\"", "")
					queryExpr = strings.ReplaceAll(queryExpr, ", quantile=", " quantile=")
					queryExpr = strings.ReplaceAll(queryExpr, ",quantile=", "quantile=")
				}
				queryExpr += "&duration=1h&step=5m"
				query["query"] = queryExpr
				query["schema"] = ""

				queryList = append(queryList, query)
			}
		}
	}

	tab["queriesList"] = queryList

	outputJson["tabs"] = tabs

	byteData, _ := json.MarshalIndent(outputJson, "", "    ")

	f, err := os.Create(outputFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	fileData := strings.ReplaceAll(string(byteData), "\\u0026", "&")
	_, err = f.WriteString(fileData)
	if err != nil {
		fmt.Println(err)
	}

	err = f.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
}
