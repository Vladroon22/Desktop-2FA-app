.PHONY: 

tests:
	go test -v ./...

linux:
	go build -ldflags="-X main.AppVersion=1.0.7" -o ./2fa-linux cmd/main.go 

win:
	go build -ldflags="-X main.AppVersion=1.0.7" -o ./2fa-windows.exe cmd/main.go 
