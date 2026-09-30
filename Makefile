# Makefile
BINARY=jdo

build:
	go build -o $(BINARY) .

clean:
	rm -f $(BINARY) *.test *.out

.PHONY: build clean
