CXXFLAGS = -g -std=gnu++14 -fno-exceptions -nostdinc++ -isystem../include/third_party/libc++/include -isystem../include/third_party/libc++abi/include  -I../include -fno-rtti -fPIC -m64 -Wall -Wextra -Wno-unused-parameter -Wno-unused-result -O2 
CXX = g++

.PHONY: clean install

%.cc.o: %.cc
	$(CXX) $(CXXFLAGS) -c $< -o $<.o

build: lib/blockchain.cc.o engine.cc.o lib/file.cc.o lib/load_lib.cc.o lib/load_sc.cc.o lib/storage.cc.o lib/reward_distributor.cc.o
	$(CXX) -shared -fvisibility=hidden $^ -o ../lib/libdappleyv8.so -L../lib -lv8 -lv8_libbase -lv8_libplatform -licui18n -licuuc -lc++ -lpthread

install:
	sudo install ../lib/*.so /usr/local/lib/
	sudo /sbin/ldconfig

clean:
	rm **/*.o
	rm *.o
	rm ../lib/libdappleyv8.so
