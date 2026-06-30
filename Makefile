APP=Falcon

CORE=build/falcon-core
EDGE=build/falcon-edge

.PHONY: all core edge clean

all: core edge

core:
	go build -o $(CORE) ./cmd/falcon-core

edge:
	go build -o $(EDGE) ./cmd/falcon-edge

clean:
	rm -rf build/*
