install:V:
	go build -o $HOME/bin/Claude ./cmd/Claude
	go build -o $HOME/bin/Prompt ./cmd/Prompt
	go build -o $HOME/bin/Claude-Reset ./cmd/Claude-Reset
	go build -o $HOME/bin/Claude-Permissions ./cmd/Claude-Permissions

clean:V:
	rm -f $HOME/bin/Claude $HOME/bin/Prompt $HOME/bin/Claude-Reset $HOME/bin/Claude-Permissions