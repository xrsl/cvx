.PHONY: build install clean

build:
	go build -o cvx .

install: build
	mkdir -p ~/.local/bin
	cp cvx ~/.local/bin/
	@echo "Installed to ~/.local/bin/cvx"

clean:
	rm -f cvx
