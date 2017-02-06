# No fuss

1. `go get -u github.com/Comcast/eel`

2. change to the project directory

3. `GOOS=linux GOARCH=arm GOARM=7 go build`

4. `scp binary, config-eel, config-handlers to the device`
