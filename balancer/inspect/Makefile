all:
	$(CC) -g -fPIC -c -o test1.o test1.c -ldl
	$(CC) -g -fPIC -shared -o libtest1.so test1.o -ldl

	$(CC) -g -fPIC -c -o test2.o test2.c -ldl
	$(CC) -g -fPIC -shared -o libtest2.so test2.o -ldl

	$(CC) -g -fPIC -c -o test3.o test3.c -ldl
	$(CC) -g -fPIC -shared -o libtest3.so test3.o -ldl
