CXXFLAGS = -g -std=gnu++14 -fno-exceptions -nostdinc++ -isystem../../include/third_party/libc++/include -isystem../../include/third_party/libc++abi/include  -I../../include -fno-rtti -fPIC -m64 -Wall -Wextra -Wno-unused-parameter -Wno-unused-result -O2
CXX = g++

.PHONY: clean install

LIB_SRC_DIR := ../lib
LIB_SRC_FILES := $(wildcard $(LIB_SRC_DIR)/*.cc)
LIB_OBJ_FILES := $(patsubst $(LIB_SRC_DIR)/%.cc,$(LIB_SRC_DIR)/%.cc.o,$(LIB_SRC_FILES))


%.cc.o: %.cc
	$(CXX) $(CXXFLAGS) -c $< -o $<.o

build: ../engine.cc.o $(LIB_OBJ_FILES)
	$(CXX) -shared -fvisibility=hidden $^ -o ../../lib/libdappleyv8.dylib -L../../lib -lv8 -lv8_libbase -lv8_libplatform -licui18n -licuuc -lc++ -lpthread

install: build
	sudo install ../../lib/*.dylib /usr/local/lib/

clean:
	rm ../**/*.o
	rm ../*.o
	rm ../../lib/libdappleyv8.dylib
