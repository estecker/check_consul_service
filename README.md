# check_consul_service
Icinga style check for services in Consul. Two great use cases for this program are dependency chaining from within Consul, and for monitoring/alerting from Icinga. 

Program exit codes follow the API specified at https://docs.icinga.com/latest/en/pluginapi.html#returncode. 


Options
```
  -crit float
    	Critical if less than this percent of passing nodes. Used with 'service'. (default 100)
  -dc string
    	The Consul datacenter which the machine is in
  -debug
    	Enable debug logging
  -node string
    	Node you want to check the service on, otherwise check aggregate
  -service string
    	Name of service in consul to check
  -tag string
    	Services with this tag, must also have a service argument.
  -warn float
    	Warn if less than this percent of passing nodes. Used with 'service'. (default 100)
 ```
      
[![CircleCI](https://circleci.com/gh/estecker/check_consul_service.svg?style=svg)](https://circleci.com/gh/estecker/check_consul_service)
