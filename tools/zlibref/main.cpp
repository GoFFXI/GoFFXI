#include <cstddef>
#include <cstdint>
#include <cstring>
#include <algorithm>
#include <fstream>
#include <iomanip>
#include <iostream>
#include <string>
#include <vector>

namespace {

struct JumpEntry {
    const void* ptr;
};

constexpr size_t CompressedSize(size_t bits) {
    return (bits + 7) / 8;
}

inline uint32_t JumpBit(const uint8_t* table, uint32_t i) {
    return (table[i / 8] >> (i & 7)) & 1U;
}

int32_t CompressSub(const uint8_t* pattern, uint32_t read, uint32_t elem, int8_t* out, uint32_t out_sz) {
    if (CompressedSize(elem) > sizeof(uint32_t)) {
        return -1;
    }

    if (CompressedSize(read + elem) > out_sz) {
        return -1;
    }

    for (uint32_t i = 0; i < elem; ++i) {
        const uint8_t shift    = (read + i) & 7U;
        const uint32_t index   = (read + i) / 8U;
        const uint32_t invMask = ~(1U << shift);
        const uint8_t bit      = JumpBit(pattern, i) << shift;
        auto*        dst       = reinterpret_cast<uint8_t*>(out + index);
        *dst                   = static_cast<uint8_t>((invMask & *dst) + bit);
    }
    return 0;
}

class ZlibReference {
public:
    bool Init(const std::string& resourceDir) {
        if (!LoadTable(resourceDir + "/compress.dat", enc_)) {
            std::cerr << "failed to load compress.dat from " << resourceDir << "\n";
            return false;
        }

        std::vector<uint32_t> dec;
        if (!LoadTable(resourceDir + "/decompress.dat", dec)) {
            std::cerr << "failed to load decompress.dat from " << resourceDir << "\n";
            return false;
        }

        PopulateJumpTable(dec);
        return true;
    }

    int32_t Compress(const std::vector<uint8_t>& input, std::vector<uint8_t>& out) const {
        if (enc_.empty()) {
            return -1;
        }

        uint32_t read  = 0;
        const uint32_t maxBits = (static_cast<uint32_t>(out.size()) - 1U) * 8U;

        for (size_t i = 0; i < input.size(); ++i) {
            const int8_t symbol = static_cast<int8_t>(input[i]);
            const uint32_t elem = enc_[symbol + 0x180];
            if (elem + read < maxBits) {
                const uint32_t index = symbol + 0x80;
                uint32_t pattern     = enc_[index];
                uint8_t  bytes[sizeof(pattern)];
                std::memcpy(bytes, &pattern, sizeof(bytes));
                if (CompressSub(bytes, read, elem, reinterpret_cast<int8_t*>(out.data() + 1), static_cast<uint32_t>(out.size() - 1)) != 0) {
                    return -1;
                }
                read += elem;
            } else if (input.size() + 1 >= out.size()) {
                const size_t zeroLen   = (out.size() / 4) + (input.size() & 3U);
                const size_t midLen    = input.size() / 4U;
                const size_t finalLen  = input.size() & 3U;
                std::fill_n(out.begin(), std::min(zeroLen, out.size()), static_cast<uint8_t>(0));
                if (midLen > 0 && out.size() > 1) {
                    const size_t limit = std::min(midLen, out.size() - 1);
                    std::fill_n(out.begin() + 1, limit, static_cast<uint8_t>(input.size()));
                }
                const size_t offset = 1 + midLen;
                if (finalLen > 0 && offset < out.size()) {
                    const size_t limit = std::min(finalLen, out.size() - offset);
                    std::fill_n(out.begin() + offset, limit, static_cast<uint8_t>((input.size() + 1) * 8));
                }
                return static_cast<int32_t>(input.size());
            } else {
                return -1;
            }
        }

        out[0] = 1;
        return static_cast<int32_t>(read + 8);
    }

private:
    bool LoadTable(const std::string& path, std::vector<uint32_t>& out) const {
        std::ifstream file(path, std::ios::binary | std::ios::ate);
        if (!file) {
            return false;
        }

        const std::streamsize size = file.tellg();
        if (size <= 0 || size % static_cast<std::streamsize>(sizeof(uint32_t)) != 0) {
            return false;
        }

        file.seekg(0, std::ios::beg);
        out.resize(static_cast<size_t>(size) / sizeof(uint32_t));
        file.read(reinterpret_cast<char*>(out.data()), size);
        return file.good();
    }

    void PopulateJumpTable(const std::vector<uint32_t>& dec) {
        jump_.resize(dec.size());
        const uint32_t base = dec[0] - sizeof(uint32_t);
        for (size_t i = 0; i < dec.size(); ++i) {
            if (dec[i] > 0xFF) {
                jump_[i].ptr = jump_.data() + (dec[i] - base) / sizeof(base);
            } else {
                jump_[i].ptr = reinterpret_cast<void*>(static_cast<std::uintptr_t>(dec[i]));
            }
        }
    }

    std::vector<uint32_t> enc_;
    std::vector<JumpEntry> jump_;
};

std::vector<uint8_t> ReadBytes(const std::string& path) {
    std::ifstream file(path, std::ios::binary | std::ios::ate);
    if (!file) {
        return {};
    }

    const std::streamsize size = file.tellg();
    if (size < 0) {
        return {};
    }

    file.seekg(0, std::ios::beg);
    std::vector<uint8_t> data(static_cast<size_t>(size));
    file.read(reinterpret_cast<char*>(data.data()), size);
    if (!file.good()) {
        return {};
    }
    return data;
}

} // namespace

int main(int argc, char** argv) {
    if (argc < 3) {
        std::cerr << "Usage: " << argv[0] << " <resource_dir> <payload_file>\n";
        return 1;
    }

    const std::string resourceDir = argv[1];
    const std::string payloadPath = argv[2];

    ZlibReference codec;
    if (!codec.Init(resourceDir)) {
        return 1;
    }

    auto payload = ReadBytes(payloadPath);
    if (payload.empty()) {
        std::cerr << "payload is empty or missing: " << payloadPath << "\n";
        return 1;
    }

    std::vector<uint8_t> out(payload.size() * 2 + 64);
    const int32_t bits = codec.Compress(payload, out);
    if (bits < 0) {
        std::cerr << "compression failed\n";
        return 1;
    }

    const size_t bytes = CompressedSize(static_cast<size_t>(bits));
    std::cout << "bits=" << bits << "\n";
    std::cout << "bytes=" << bytes << "\n";
    std::cout << "hex=" << std::uppercase << std::hex << std::setfill('0');
    for (size_t i = 0; i < bytes; ++i) {
        std::cout << std::setw(2) << static_cast<int>(out[i]);
    }
    std::cout << std::dec << "\n";
    return 0;
}
