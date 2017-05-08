package main

//check_consul_service.go
//Nagios plugins style check for services in consul
//eddie.stecker my first Go program

import (
	"flag"
	"fmt"
	consulapi "github.com/hashicorp/consul/api"
	"io"
	"io/ioutil"
	"os"
)

// Trying to map consul status to icinga 0-3 status
// TODO test maintenance
var statusMap = map[string]int{
	"passing":  0,
	"warning":  1,
	"critical": 2,
	"unknown":  3,
	"maintenance": 3,
	"error": 3,
}

func CheckNode(out io.Writer, health consulapi.Health, queryOptions consulapi.QueryOptions, argNode *string) {
	//node check, only critical or OK exit codes
	fmt.Fprintln(out, "Only querying", *argNode, "for all services.")
	healthCheck, queryMeta, queryError := health.Node(*argNode, &queryOptions)
	if queryError != nil {
		panic(queryError)
	}
	fmt.Fprintln(out, "Request Time:", queryMeta.RequestTime)
	if len(healthCheck) == 0 {
		fmt.Fprintln(os.Stdout, "The node:", *argNode, "could not be found.")
		os.Exit(statusMap["unknown"])  //Exiting unknown as it's not really a failing node, just unknown
	}
	for _, check := range healthCheck {
		if check.Status != "passing" {
			fmt.Fprintln(os.Stdout, check.Name, check.Status) //short-circuit exit, may have other errors
			os.Exit(statusMap["critical"])               //CRITICAL
		}
	}
	fmt.Fprintln(os.Stdout, "Node", *argNode, "is passing all checks.")
	os.Exit(statusMap["passing"]) //OK

}

func CheckService(out io.Writer, argService *string, health consulapi.Health, argTag *string, queryOptions consulapi.QueryOptions, argCrit *float64, argWarn *float64) {
	// Service check on all nodes
	// By default, the service on all nodes need to be passing
	fmt.Fprintln(out, "Querying all nodes of service:", *argService)
	// TODO verify tags
	serviceEntry, queryMeta, queryError := health.Service(*argService, *argTag, false, &queryOptions)
	if queryError != nil {
		panic(queryError)
	}
	fmt.Fprintln(out, "Request Time:", queryMeta.RequestTime)
	nodeStatus := make(map[string]int) // Map to later calculate how many nodes are passing
	for _, se := range serviceEntry {
		// Loop through results
		fmt.Fprintln(out, "Node:", *se.Node, "Service:", *se.Service)
		nodeStatus[se.Node.Node] = statusMap["passing"] // TODO start as critical?
		for _, check := range se.Checks {
			fmt.Fprintln(out, check.Status)
			nodeStatus[se.Node.Node] = nodeStatus[se.Node.Node] + statusMap[check.Status]
		}
	}
	// Now calculate if a node is passing or not
	totalNodes := len(nodeStatus)
	totalPassing := 0
	totalFail := 0
	if totalNodes == 0 {
		//Quick exit at this point
		fmt.Fprintln(os.Stdout, "ERROR Zero nodes were found")
		os.Exit(statusMap["unknown"])
	}
	for i := range nodeStatus {
		if nodeStatus[i] == 0 {
			// passing
			totalPassing = totalPassing + 1
		} else {
			//if not passing, then it's failing
			totalFail = totalFail + 1
		}
	}
	passingPercent := float64(totalPassing) / float64(totalNodes) * 100
	fmt.Fprintln(os.Stdout, "Total nodes:", totalNodes, "Passing nodes:", totalPassing, "Failing nodes:", totalFail, "Passing percent:", passingPercent)

	if passingPercent < *argCrit {
		// Not enough passing nodes
		os.Exit(statusMap["critical"])
	} else if passingPercent >= *argCrit && passingPercent <= *argWarn {
		os.Exit(statusMap["warning"])
	} else {
		os.Exit(statusMap["passing"])
	}
}

func CheckNodeService(out io.Writer, argService *string, health consulapi.Health, queryOptions consulapi.QueryOptions, argNode *string) {
	//Query a node for one specific service
	fmt.Fprintln(out, "Only querying", *argNode, "for the service", *argService)
	healthCheck, queryMeta, queryError := health.Node(*argNode, &queryOptions)
	if queryError != nil {
		panic(queryError)
	}
	fmt.Fprintln(out, "Request Time:", queryMeta.RequestTime)
	serviceFound := false //To keep track of a match
	for _, check := range healthCheck {
		fmt.Fprintln(out, check.CheckID, check.Status)
		if check.ServiceName == *argService {
			serviceFound = true
			if check.Node == *argNode && check.Status != "passing" {
				// Then there's a failing health check, lets exit now
				fmt.Fprintln(os.Stdout, check.Name, check.Status)
				os.Exit(statusMap["critical"])
			}
		}
	}
	if serviceFound {
		// All status checks were passing, so exit passing
		fmt.Fprintln(os.Stdout, "Service", *argService, "is passing on", *argNode)
		os.Exit(statusMap["passing"]) //OK
	} else {
		// Service not running on the node?
		fmt.Fprintln(os.Stdout, "The service", *argService, "was not found on", *argNode)
		os.Exit(statusMap["error"])  //Service does not exist
	}
}

func main() {
	var out io.Writer = ioutil.Discard //Set out to discard as default
	config := consulapi.DefaultConfig()
	client, err := consulapi.NewClient(config)
	if err != nil {
		panic(err)
	}

	health := client.Health()

	argDebug := flag.Bool("debug", false, "Enable debug logging")
	//argServer TODO Server address, but for now default is localhost
	//argPort TODO?
	//argTLS TODO?
	argService := flag.String("service", "", "Name of service in consul to check")
	argNode := flag.String("node", "", "Node you want to check the service on, otherwise check aggregate")
	argTag := flag.String("tag", "", "Services with this tag, must also have a service argument.")
	argWarn := flag.Float64("warn", 100, "Warn if less than this percent of passing nodes. Used with 'service'.")
	argCrit := flag.Float64("crit", 100, "Critical if less than this percent of passing nodes. Used with 'service'.")
	argDC := flag.String("dc", "", "The Consul datacenter which the machine is in")
	flag.Parse()


	queryOptions := consulapi.QueryOptions{AllowStale: true}
	if *argDC != "" {
		queryOptions = consulapi.QueryOptions{AllowStale: true, Datacenter: *argDC}
	}


	if *argDebug == true {
		//Setup debug mode
		out = os.Stdout
		visitor := func(a *flag.Flag) {
			fmt.Fprintln(out, "Arg:", a.Name, "value =", a.Value)
		}
		flag.VisitAll(visitor)
	}
	if *argService == "" && *argNode == "" {
		//No args, lets exit and print help
		fmt.Fprintln(os.Stdout, "Need some arguments to run. Try \"-service consul\"")
		flag.PrintDefaults()
		os.Exit(statusMap["error"])
	} else if *argWarn < *argCrit {
		fmt.Fprintln(os.Stdout, "Critical percent", *argCrit, "is higher than warning,", *argWarn, "that does not make sense.")
		flag.PrintDefaults()
		os.Exit(statusMap["error"])
	} else if *argService != "" && *argNode == "" {
		//Check a service which may be on multiple nodes. Calculate passing based on total % passing
		CheckService(out, argService, *health, argTag, queryOptions, argCrit, argWarn)
	} else if *argService != "" && *argNode != "" {
		//Check a specific node for a specific service
		if *argWarn != 100 || *argCrit != 100 {
			fmt.Fprintln(os.Stdout, "Setting a warning or critical precentage does not make sense with both service and node.")
			flag.PrintDefaults()
			os.Exit(statusMap["error"])
		}
		CheckNodeService(out, argService, *health, queryOptions, argNode)
	} else if *argNode != "" && *argService == "" {
		//Check a node only, thus, it includes all services
		CheckNode(out, *health, queryOptions, argNode)
	}
}
