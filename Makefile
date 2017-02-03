all:	clean eel-test eel-build eel-install

eel-test:
	cd test && go test -v

eel-build:
	go build
	cp eel bin/eel

eel-install:
	go install

clean:
	rm -f bin/eel bin/eel.pid $$GOPATH/bin/eel eel.log eel
