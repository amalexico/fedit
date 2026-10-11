build:
	go build -o fedit .

test:
	go test ./...

clean:
	rm -f fedit
