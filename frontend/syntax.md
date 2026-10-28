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
   : portItemClause binaryOperator integer || ipItemClause binaryOperator ipv4 || searchClause || stringItemClause binaryOperator string

binaryOperator
   : "eq" || "==" || "ne" || "!="

portItemClause
   : "tcp.port" || "udp.port"

ipItemClause
   : "ip.src"

stringItemClause
   : "end.reason" || "handler" || "transport" || "scanner"

integer:
   : regex([0-9]+)

ipv4:
   : regex([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})

searchClause: 
   : payload searchOperator string

searchOperators: 
   "contains" || "matches" || "~"

   "contains" is a substring test. "matches" and "~" test a JavaScript regular
   expression against the decoded payload; a pattern that does not compile is
   reported like a syntax error.

   A pattern is used as written, so a backslash reaches the expression intact:
   "\.php" matches a literal dot and "\d+" matches digits. Matching is case
   sensitive; a leading "(?i)", "(?s)" or "(?m)" sets the corresponding flag, as
   in "(?i)wget". Only the first 16 KiB of a payload is tested, which keeps a
   slow pattern from locking up the page.

payload:
   : "payload"

string:
   : regex(\"(?:[^"\\]|\\.)*\")
```


# Examples (not all supported yet)

`ip.src eq 192.168.1.12 and ip.port == 8080`  
`ip.src eq 192.168.1.12 or udp.port == 12`  
`ip.src != 192.168.1.12 or udp.port == 12`  
`ip.src != 192.168.1.12 or udp.port == 12`  
`not ip.src != 192.168.1.12`  
`not (ip.src != 192.168.1.12)`  
`handler eq "http"`  
`transport eq "udp"`  
`scanner eq "censys"`  
`not scanner eq "censys"`  
`transport ne "tcp" and payload contains "something"`  
`payload contains "GET / HTTP/1.1"`  
`payload contains "192.168.1.1"`  
`payload matches "wget|curl"`  
`payload ~ "^GET /[^ ]*\.php"`  
`payload matches "(?i)wget"`  
`end.reason == "connection reset by peer"`

