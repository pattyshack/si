MAKEFLAGS = -s

all:
	go run github.com/pattyshack/gt/tools/lr/bin/generate-lr-parser -o grammar.go grammar.lr

clean:
	rm -f grammar.go
