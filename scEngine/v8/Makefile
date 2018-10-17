all:
	g++ -fPIC -shared -I../include engine.cc -o ../lib/libdappleyv8.so -lv8_monolith -L../lib -pthread -std=c++0x
	sudo install ../lib/libdappleyv8.so /usr/local/lib/
	sudo /sbin/ldconfig
