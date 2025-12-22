.PHONY: build build-pico2 flash-pico2 clean

build:
	go build -o receiver .

build-pico2:
	tinygo build -o /dev/null -target=pico2 .

flash-pico2:
	tinygo flash -target=pico2 .

clean:
	rm -f receiver
