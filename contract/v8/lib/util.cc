#include "util.h"

std::string ReplaceAll(std::string str, const std::string &from,
                       const std::string &to) {
  size_t from_len = from.length(), to_len = to.length();
  size_t start_pos = 0;
  while ((start_pos = str.find(from, start_pos)) != std::string::npos) {
    str.replace(start_pos, from_len, to);
    start_pos += to_len;
  }
  return str;
}
