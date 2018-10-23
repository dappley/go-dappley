CXXFLAGS =-g -std=c++0x -I../include -fno-rtti -fPIC -m64 -Wall -Wextra -Wno-unused-parameter -Wno-unused-result -O2 -static-libstdc++
CXX = g++-4.8

%.cc.o: %.cc
	$(CXX) $(CXXFLAGS) -c $< -o $<.o

build: engine.cc.o
	$(CXX) -shared -fvisibility=hidden $^ -o ../lib/libdappleyv8.so -L../lib -lv8_monolith -lpthread

install: build
	sudo install ../lib/libdappleyv8.so /usr/local/lib/
	sudo /sbin/ldconfig
