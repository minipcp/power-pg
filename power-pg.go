package main

import (
	"flag"
	"github.com/minipcp/power-pg/proxy"
)


var (
	localHost = flag.String("l", ":9876", "Endereço e porta do listener local")
	remoteHost = flag.String("r", "localhost:5432", "Endereço e porta do servidor PostgreSQL")
)


func main() {
	flag.Parse()
	proxy.Start(localHost, remoteHost, getQueryModificada)
}


func getQueryModificada(queryOriginal string) string {
	if queryOriginal[:5] != "power" {
		return queryOriginal
	}
	return "select * from clientes limit 1;"
}
