.PHONY: build build-scout build-resolver clean

BINDIR := pq/bin

build: build-scout build-resolver

$(BINDIR):
	mkdir -p $(BINDIR)

build-scout: $(BINDIR)
	go -C pq build -o bin/scout ./cmd/scout

build-resolver: $(BINDIR)
	go -C pq build -o bin/resolver ./cmd/resolver

clean:
	rm -rf $(BINDIR)
