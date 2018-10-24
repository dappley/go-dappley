CXXFLAGS = -g -std=gnu++14 -fno-exceptions -nostdinc++ -isystem../include/third_party/libc++/include -isystem../include/third_party/libc++abi/include  -I../include -fno-rtti -fPIC -m64 -Wall -Wextra -Wno-unused-parameter -Wno-unused-result -O2 
CXX = g++

.PHONY: clean

%.cc.o: %.cc
	$(CXX) $(CXXFLAGS) -c $< -o $<.o

build: lib/blockchain.cc.o engine.cc.o
	$(CXX) -shared -fvisibility=hidden $^ -o ../lib/libdappleyv8.so -L../lib -lv8_monolith -lc++ -lpthread

install: build
	sudo install ../lib/*.so /usr/local/lib/
	sudo /sbin/ldconfig

clean:
	rm *.o
	rm ../lib/libdappleyv8.so
