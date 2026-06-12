## b64

a fast base64 encoder/decoder that runs in your terminal. handles standard base64, base64url, base32, and base16 (hex).

## install

```bash
go install github.com/ufraaan/b64@latest
```

or build from source:

```bash
git clone https://github.com/ufraaan/b64
cd b64
go build -o /usr/local/bin/b64 .
```

## usage

```bash
# encode a file
b64 encode image.png > image.b64

# decode a file
b64 decode image.b64 > image.png

# pipe stuff through it
cat somefile | b64 encode | b64 decode > original

# auto detect mode (no subcommand needed)
b64 somefile              # encodes if it looks binary, decodes if it looks like base64
```

## flags

| flag | what it does |
|---|---|
| `-w <n>` | wrap output every N columns (default 76, 0 = no wrap) |
| `--url` | use url safe alphabet (- instead of +, _ instead of /) |
| `--raw` | url safe alphabet without padding |
| `--base32` | use base32 instead of base64 |
| `--base16` | use hex instead of base64 |
| `--ignore-garbage` | skip random characters during decode |
| `--check` | check if input is valid without decoding it |

## examples

```bash
# url safe encoding
echo "some data" | b64 --url encode

# no line wrapping
b64 -w 0 encode bigfile.bin

# validate a base64 file
b64 --check < file.b64

# base32 encode
echo "hello" | b64 --base32 encode
# NBSWY3DP

# base16 (hex) encode
echo "hello" | b64 --base16 encode
# 68656C6C6F

# ignore garbage in otherwise valid base64
printf "aGVs\nbG8=" | b64 --ignore-garbage decode
```

## how auto detect works

when you run `b64` without saying encode or decode, it looks at the first 512 bytes of your input. if it sees binary data (anything outside printable ascii), it encodes. if the data looks like valid base64 (has digits, = signs, or +/ characters), it decodes. if the data is just letters, it plays it safe and encodes instead.

you can always override with `b64 encode` or `b64 decode` if the auto mode guesses wrong.

## streaming

the tool never loads the whole file into memory. it processes data in small chunks. for encoding it reads 3 bytes at a time and writes 4 characters. for decoding it reads 4 characters and writes 3 bytes. this means you can run it on files of any size without running out of memory.

## performance

runs at about 1.8 GB/s for encoding and 870 MB/s for decoding on an Apple M5. tested with 64KB blocks.

## what's in the box

the repo is split into a few internal packages:

- `internal/base64/` - the main base64 implementation. has separate files for the encoding type, encode logic, decode logic, streaming, and validation.
- `internal/base32/` - base32 (RFC 4648)
- `internal/base16/` - hex encoding

the `main.go` ties everything together into the CLI tool.

## benchmarks (Apple M5, 64KB blocks)

| operation | throughput |
|---|---|
| encode (std) | ~1.9 GB/s |
| decode (std) | ~792 MB/s |
| encode (url) | ~2.0 GB/s |
| decode (url) | ~826 MB/s |
| stream encode | ~246 MB/s |
| stream decode | ~378 MB/s |

## tests

```bash
go test ./...
```

the test suite covers roundtrip encoding/decoding for all variants, streaming with various sizes, wrapped output, garbage handling, input validation, and cross validation against your system's `base64` command.

## benchmarks

```bash
go test ./internal/base64 -bench=. -benchtime=1s
```
