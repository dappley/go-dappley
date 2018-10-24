CXXFLAGS =-g -std=c++0x -I../include -fno-rtti -fPIC -m64 -Wall -Wextra -Wno-unused-parameter -Wno-unused-result -O2 -static-libstdc++
CXX = g++

%.cc.o: %.cc
	$(CXX) $(CXXFLAGS) -c $< -o $<.o

build: lib/blockchain.cc.o engine.cc.o
	$(CXX) -shared -fvisibility=hidden $^ -o ../lib/libdappleyv8.dylib -L../lib -lv8_monolith -lpthread

install: build
	sudo install ../lib/libdappleyv8.dylib /usr/local/lib/
	sudo /sbin/ldconfig
