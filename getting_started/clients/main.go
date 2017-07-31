package main

import (
	"fmt"
	"go-manager"
	"go-manager/elastic"
	"go-manager/gateway"
	"io"
	"net/http"
)

func main() {
	//
	// MANAGER
	//
	manager, _ := pm.NewManager()

	//
	// GATEWAY
	//
	var headers map[string]string
	var body io.Reader
	configGateway := gateway.NewConfig("http://localhost:8081")
	gateway, _ := manager.NewGateway(configGateway)
	manager.AddGateway("gateway_1", gateway)
	manager.GetGateway("gateway_1")
	status, bytes, err := manager.RequestGateway("gateway_1", http.MethodGet, "/example/123456789", headers, body)
	fmt.Println("STATUS:", status, "RESPONSE:", string(bytes), "err:", err)

	//
	// ELASTIC CLIENT
	//
	configElasticClient := elastic.NewConfig("http://localhost:9200")
	elasticClient := manager.NewElasticClient(configElasticClient)
	manager.AddElasticClient("elastic_1", elasticClient)
	response, err := elasticClient.Search("index", "type", "body")
	fmt.Println("RESPONSE:", response, "ERROR:", err)
}
