.PHONY: build build-scout build-resolver clean

BINDIR := bin

build: build-scout build-resolver

$(BINDIR):
	mkdir -p $(BINDIR)

build-scout: $(BINDIR)
	go build -o $(BINDIR)/scout ./cmd/scout

build-resolver: $(BINDIR)
	go build -o $(BINDIR)/resolver ./cmd/resolver

clean:
	rm -rf $(BINDIR)
