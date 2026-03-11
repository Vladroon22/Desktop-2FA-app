.PHONY: app

linux:
	go build -ldflags="-X main.appVersion=1.0.6" -o ./2fa-linux cmd/main.go 

win:
	go build -ldflags="-X main.appVersion=1.0.6" -o ./2fa-windows cmd/main.go 
