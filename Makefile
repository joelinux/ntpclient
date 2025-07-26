
all: ntpclient ntpclient.exe

ntpclient: ntpclient.go
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ntpclient	

ntpclient.exe: ntpclient.go
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ntpclient.exe
clean:
	rm -f ntpclient ntpclient.exe
fmt:
	go fmt
