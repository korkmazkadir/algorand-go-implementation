build:
	go build .

clean:
	rm ./algorand-go-implementation
	rm addressbook.txt
	rm -r output/
	./kill-processes.sh
	rm ./process.pids
