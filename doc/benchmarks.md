# go version go1.4.2 darwin/amd64

./eel/test$ export GOMAXPROCS=1

./eel/test$ go test -bench=.

PASS

Test | Count | Time
---|---|---
BenchmarkRawTransformation	|   30000	  |   49158 ns/op
BenchmarkCanonicalizeEvent	  |  1000	 |  1921845 ns/op
BenchmarkTransformationByExample|	    1000	  | 1809619 ns/op
BenchmarkNamedTransformation	|    1000	 |  1855053 ns/op
BenchmarkMessageGeneration	|    1000	 |  1954922 ns/op
BenchmarkJavaScript	  |  1000	  | 2294415 ns/op
BenchmarkMatchByExample	|    1000	  | 1783525 ns/op
BenchmarkCustomProperties	|    1000	|   1803217 ns/op
BenchmarkStringOps	 |   1000	|   1745312 ns/op
BenchmarkContains	 |   1000	 |  2113395 ns/op
BenchmarkCase	  |  1000	  | 1924249 ns/op
BenchmarkRegex	|    1000	|   1739750 ns/op

ok 	28.714s

./eel/test$ export GOMAXPROCS=8

./eel/test$ go test -bench=.

PASS

Test | Count | Time
---|---|---
BenchmarkRawTransformation-8	|   30000	   |  59814 ns/op
BenchmarkCanonicalizeEvent-8	 |   1000	 |  1453160 ns/op
BenchmarkTransformationByExample-8 |	    1000|	   1402856 ns/op
BenchmarkNamedTransformation-8	 |   1000	 |  1384841 ns/op
BenchmarkMessageGeneration-8	 |   1000	  | 1395114 ns/op
BenchmarkJavaScript-8	 |   1000	|   1362980 ns/op
BenchmarkMatchByExample-8	   | 1000	 |  1361960 ns/op
BenchmarkCustomProperties-8	  |  1000	|   1367858 ns/op
BenchmarkStringOps-8	  |  1000	  | 1337911 ns/op
BenchmarkContains-8	 |   1000	 |  1457167 ns/op
BenchmarkCase-8	 |   1000	 |  1378816 ns/op
BenchmarkRegex-8	  |  1000	 |  1352709 ns/op

ok 22.808s

# go version go1.5.2 darwin/amd64

./eel/test$ export GOMAXPROCS=1

./eel/test$ go test -bench=.

PASS

Test | Count | Time
---|---|---
BenchmarkRawTransformation     | 	   30000	 |    39634 ns/op
BenchmarkCanonicalizeEvent     | 	    1000	 |  1890510 ns/op
BenchmarkTransformationByExample	|    1000	  | 1783343 ns/op
BenchmarkNamedTransformation  |  	    1000	 |  1809167 ns/op
BenchmarkMessageGeneration    |  	    1000	|   1895474 ns/op
BenchmarkJavaScript            | 	    1000	|   2132872 ns/op
BenchmarkMatchByExample       |  	    1000	|   1769751 ns/op
BenchmarkCustomProperties       |	    1000	 |  1776089 ns/op
BenchmarkStringOps             | 	    1000	 |  1747321 ns/op
BenchmarkContains             |  	    1000	 |  2000132 ns/op
BenchmarkCase                |   	    1000	  | 1859262 ns/op
BenchmarkRegex                  |	    1000	|   1779854 ns/op

ok 27.652s

./eel/test$ export GOMAXPROCS=8

./eel/test$ go test -bench=.

PASS

Test | Count | Time
---|---|---
BenchmarkRawTransformation-8      |	   30000	 |    42007 ns/op
BenchmarkCanonicalizeEvent-8      |	    1000	 |  1483847 ns/op
BenchmarkTransformationByExample-8	|    1000	 |  1428444 ns/op
BenchmarkNamedTransformation-8    |	    1000	 |  1413214 ns/op
BenchmarkMessageGeneration-8      |	    1000	|   1427241 ns/op
BenchmarkJavaScript-8          |   	    1000	 |  1388611 ns/op
BenchmarkMatchByExample-8        | 	    1000	 |  1405158 ns/op
BenchmarkCustomProperties-8    |   	    1000	 |  1410230 ns/op
BenchmarkStringOps-8          |    	    1000	 |  1412329 ns/op
BenchmarkContains-8           |    	    1000	 |  1513613 ns/op
BenchmarkCase-8              |     	    1000	 |  1461517 ns/op
BenchmarkRegex-8              |    	    1000	 |  1458123 ns/op

ok 22.675s
