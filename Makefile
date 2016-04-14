all:	clean eel-test eel-build eel-install

eel-test:
	cd eel/test && go test -v

eel-build:
	cd eel/eelsys && go build

eel-install:
	cd eel/eelsys && go install

clean:
	rm -f eelsys/eelsys $$GOPATH/bin/eelsys eel.log
