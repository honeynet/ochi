# Query DSL is vaguely based on these rules:

initial query 
   : booleanclause and query : booleandclause or query : booleanclause 

```
query
   : booleanClause booleanSuffixClause || "NOT" booleanClause booleanSuffixClause

searchclause
   : payload searchOperator string

booleanSuffixClause:
   emptyString || OR query || AND query

booleanClause
   : binaryClause || unaryClause

binaryClause
   : portItemClause binaryOperator integer || ipItemClause binaryOperator ipv4 || searchClause

binaryOperator
   : "eq" || "==" || "ne" || "!="

portItemClause
   : "tcp.port" || "udp.port"

ipItemClause
   : "ip.src" || "ip.dst"

integer:
   : regex([0-9]+)

ipv4:
   : regex([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})

searchClause: 
   : payload searchOperator string

searchOperators: 
   "contains" || "matches" || "~"

payload:
   : "payload"

string:
   : regex([\S]+)
```


# Examples (not all supported yet)

`ip.src eq 192.168.1.12 and ip.port == 8080`  
`ip.src eq 192.168.1.12 or udp.port == 12`  
`ip.src != 192.168.1.12 or udp.port == 12`  
`ip.src != 192.168.1.12 or udp.port == 12`  
`not ip.src != 192.168.1.12`  
`not (ip.src != 192.168.1.12)`
