install:V:
	go build -o $HOME/bin/Claude .

clean:V:
	rm -f $HOME/bin/Claude
