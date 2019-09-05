ARCH = $(shell arch)
PLATFORM = m64
ifneq ($(findstring arm, $(ARCH)),)
	PLATFORM = marm
endif

ifeq ($(PLATFORM), m64)
	CXXFLAGS = -g -std=c++11 -fno-exceptions -nostdinc++ -isystem../include/third_party/libc++/include -isystem../include/third_party/libc++abi/include  -I../include/x86_64 -fno-rtti -fPIC -$(PLATFORM) -Wall -Wextra -Wno-unused-parameter -Wno-unused-result -O2
endif
ifeq ($(PLATFORM), marm)
	CXXFLAGS = -g -std=c++11 -fno-exceptions -nostdinc++ -isystem../include/third_party/libc++/include -isystem../include/third_party/libc++abi/include  -I../include/arm -fno-rtti -fPIC -$(PLATFORM) -Wall -Wextra -Wno-unused-parameter -Wno-unused-result -O2
endif

CXX = g++
UNAME = $(shell uname)
ifeq ($(UNAME), Linux)
	FILESUFFIX = so
	LDCONFIG = sudo /sbin/ldconfig
endif
ifeq ($(UNAME), Darwin)
	FILESUFFIX = dylib
	LDCONFIG =
endif

.PHONY: clean install

LIB_SRC_DIR := lib
LIB_SRC_FILES := $(wildcard $(LIB_SRC_DIR)/*.cc)
LIB_OBJ_FILES := $(patsubst $(LIB_SRC_DIR)/%.cc,$(LIB_SRC_DIR)/%.cc.o,$(LIB_SRC_FILES))

%.cc.o: %.cc
	$(CXX) $(CXXFLAGS) -c $< -o $<.o

build: engine.cc.o thread_engine.cc.o $(LIB_OBJ_FILES)
ifeq ($(PLATFORM), m64)
	$(CXX) -shared -fvisibility=hidden $^ -o ../lib/libdappleyv8.$(FILESUFFIX) -L../lib/x86_64 -lv8 -lv8_libbase -lv8_libplatform -licui18n -licuuc -lc++ -lpthread
endif
ifeq ($(PLATFORM), marm)
	$(CXX) -shared -fvisibility=hidden $^ -o ../lib/libdappleyv8.$(FILESUFFIX) -L../lib/arm -lv8 -lv8_libbase -lv8_libplatform -licui18n -licuuc -lc++ -lpthread
endif

install: build
	sudo install ../lib/*.$(FILESUFFIX) /usr/local/lib/
	$(LDCONFIG)

clean:
	rm **/*.o
	rm *.o
	rm ../lib/libdappleyv8.$(FILESUFFIX)
