BIN=$HOME/bin

all:V: Claude Prompt Claude-Reset Claude-Permissions

Claude:V: cmd/Claude/main.go internal/acme/client.go
	cd cmd/Claude && go build -o $BIN/Claude .

Prompt:V: cmd/Prompt/main.go internal/acme/client.go internal/context/manager.go
	cd cmd/Prompt && go build -o $BIN/Prompt .

Claude-Reset:V: cmd/Claude-Reset/main.go internal/acme/client.go internal/context/manager.go
	cd cmd/Claude-Reset && go build -o $BIN/Claude-Reset .

Claude-Permissions:V: cmd/Claude-Permissions/main.go internal/acme/client.go internal/context/manager.go
	cd cmd/Claude-Permissions && go build -o $BIN/Claude-Permissions .

clean:V:
	rm -f $BIN/Claude $BIN/Prompt $BIN/Claude-Reset $BIN/Claude-Permissions

install:V: all