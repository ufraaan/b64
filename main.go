package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/ufraaan/b64/internal/base16"
	"github.com/ufraaan/b64/internal/base32"
	"github.com/ufraaan/b64/internal/base64"
)

type encodingKind int

const (
	encBase64 encodingKind = iota
	encBase64URL
	encBase32
	encBase16
)

type config struct {
	kind          encodingKind
	mode          string
	file          string
	wrap          int
	raw           bool
	ignoreGarbage bool
	check         bool
}

func main() {
	cfg, shell := parseFlags()
	if shell != "" {
		writeCompletion(shell)
		return
	}
	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() (config, string) {
	wrap := flag.Int("w", 76, "wrap encoded output at COLUMNS (0 = no wrap)")
	url := flag.Bool("url", false, "use URL-safe alphabet")
	raw := flag.Bool("raw", false, "URL-safe alphabet without padding")
	ignoreGarbage := flag.Bool("ignore-garbage", false, "skip non-alphabet characters during decode")
	check := flag.Bool("check", false, "validate input format without decoding")
	useBase32 := flag.Bool("base32", false, "use base32 encoding")
	useBase16 := flag.Bool("base16", false, "use base16 (hex) encoding")
	flag.Parse()

	args := flag.Args()

	cfg := config{
		wrap:          *wrap,
		raw:           *raw,
		ignoreGarbage: *ignoreGarbage,
		check:         *check,
	}

	switch {
	case *useBase32:
		cfg.kind = encBase32
	case *useBase16:
		cfg.kind = encBase16
	case *url || *raw:
		cfg.kind = encBase64URL
	}

	if len(args) > 0 {
		switch args[0] {
		case "encode", "decode":
			cfg.mode = args[0]
			if len(args) > 1 {
				cfg.file = args[1]
			}
		case "completion":
			if len(args) > 1 {
				return cfg, args[1]
			}
			return cfg, "bash"
		default:
			cfg.file = args[0]
		}
	}

	return cfg, ""
}

func run(cfg config) error {
	r, closer, err := openInput(cfg.file)
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer.Close()
	}

	header, r, err := peekReader(r, 512)
	if err != nil {
		return err
	}

	if cfg.mode == "" {
		cfg.mode = detectMode(header, cfg.kind)
	}

	if cfg.check {
		return checkStream(cfg, r)
	}

	switch cfg.mode {
	case "encode":
		return encodeStream(cfg, r, os.Stdout)
	case "decode":
		return decodeStream(cfg, r, os.Stdout)
	default:
		return fmt.Errorf("unknown mode: %s", cfg.mode)
	}
}

func openInput(name string) (io.Reader, io.Closer, error) {
	if name == "" || name == "-" {
		return os.Stdin, nil, nil
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, nil, err
	}
	return f, f, nil
}

func peekReader(r io.Reader, size int) ([]byte, io.Reader, error) {
	buf := make([]byte, size)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, nil, err
	}
	buf = buf[:n]
	rest := io.MultiReader(bytes.NewReader(buf), r)
	return buf, rest, nil
}

func detectMode(header []byte, kind encodingKind) string {
	if len(header) == 0 {
		return "encode"
	}

	// Binary bytes (non-printable, non-whitespace) → definitely data to encode
	for _, b := range header {
		if b < 32 && b != '\t' && b != '\n' && b != '\r' {
			return "encode"
		}
		if b > 126 {
			return "encode"
		}
	}

	// Strip whitespace
	var cleaned []byte
	for _, b := range header {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			cleaned = append(cleaned, b)
		}
	}

	if len(cleaned) < 4 {
		return "encode"
	}

	// Check for strong signals that this is encoded text
	hasEquals := false
	hasSpecial := false
	hasDigit := false
	allUpper := true
	allLower := true

	for _, b := range cleaned {
		switch {
		case b == '=':
			hasEquals = true
		case b == '+' || b == '/' || b == '-' || b == '_':
			hasSpecial = true
		case b >= '0' && b <= '9':
			hasDigit = true
		case b >= 'A' && b <= 'Z':
			allLower = false
		case b >= 'a' && b <= 'z':
			allUpper = false
		}
	}

	// Strong signals → definitely base64-ish
	if hasEquals || hasSpecial {
		return "decode"
	}

	// Digits suggest base64 (plain text rarely has many digits)
	if hasDigit {
		return "decode"
	}

	// Base32 is always uppercase; hex is 0-9 A-F
	if kind == encBase32 && allUpper && !allLower {
		return "decode"
	}
	if kind == encBase16 {
		// Hex has even length and only 0-9 A-F a-f
		valid := true
		for _, b := range cleaned {
			if !((b >= '0' && b <= '9') || (b >= 'A' && b <= 'F') || (b >= 'a' && b <= 'f')) {
				valid = false
				break
			}
		}
		if valid && len(cleaned)%2 == 0 {
			return "decode"
		}
	}

	// All-letter input is ambiguous: default to encode
	// (user can always override with explicit 'decode' command)
	return "encode"
}

func encodingFor(cfg config) string {
	switch cfg.kind {
	case encBase64URL:
		if cfg.raw {
			return "base64url-nopad"
		}
		return "base64url"
	case encBase32:
		return "base32"
	case encBase16:
		return "base16"
	default:
		return "base64"
	}
}

func checkStream(cfg config, r io.Reader) error {
	encName := encodingFor(cfg)
	buf := make([]byte, 32*1024)

	for {
		n, err := r.Read(buf)
		if n > 0 {
			switch cfg.kind {
			case encBase64, encBase64URL:
				enc := pickBase64(cfg)
				if err := enc.Check(buf[:n]); err != nil {
					return fmt.Errorf("%s: %w", encName, err)
				}
			case encBase32:
				enc := base32.StdEncoding
				if err := enc.Check(buf[:n]); err != nil {
					return fmt.Errorf("%s: %w", encName, err)
				}
			case encBase16:
				enc := base16.StdEncoding
				if err := enc.Check(buf[:n]); err != nil {
					return fmt.Errorf("%s: %w", encName, err)
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stderr, "%s: input is valid\n", encName)
	return nil
}

func pickBase64(cfg config) *base64.Encoding {
	if cfg.kind == encBase64URL {
		if cfg.raw {
			return base64.RawURLEncoding
		}
		return base64.URLEncoding
	}
	return base64.StdEncoding
}

func encodeStream(cfg config, r io.Reader, w io.Writer) error {
	encName := encodingFor(cfg)
	buf := make([]byte, 32*1024)

	switch cfg.kind {
	case encBase64, encBase64URL:
		enc := pickBase64(cfg)
		encw := base64.NewEncoder(enc, w)
		if cfg.wrap > 0 {
			encw.WithWrap(cfg.wrap)
		}
		if _, err := io.CopyBuffer(encw, r, buf); err != nil {
			return fmt.Errorf("%s encode: %w", encName, err)
		}
		if err := encw.Close(); err != nil {
			return fmt.Errorf("%s encode: %w", encName, err)
		}

	case encBase32:
		encw := base32.NewEncoder(base32.StdEncoding, w)
		if cfg.wrap > 0 {
			encw.WithWrap(cfg.wrap)
		}
		if _, err := io.CopyBuffer(encw, r, buf); err != nil {
			return fmt.Errorf("%s encode: %w", encName, err)
		}
		if err := encw.Close(); err != nil {
			return fmt.Errorf("%s encode: %w", encName, err)
		}

	case encBase16:
		encw := base16.NewEncoder(base16.StdEncoding, w)
		if _, err := io.CopyBuffer(encw, r, buf); err != nil {
			return fmt.Errorf("%s encode: %w", encName, err)
		}
	}

	return nil
}

func decodeStream(cfg config, r io.Reader, w io.Writer) error {
	encName := encodingFor(cfg)
	buf := make([]byte, 32*1024)

	switch cfg.kind {
	case encBase64, encBase64URL:
		enc := pickBase64(cfg)
		decr := base64.NewDecoder(enc, r)
		if cfg.ignoreGarbage {
			decr.WithIgnoreGarbage(true)
		}
		if _, err := io.CopyBuffer(w, decr, buf); err != nil {
			return fmt.Errorf("%s decode: %w", encName, err)
		}

	case encBase32:
		dec, err := base32.StdEncoding.Decode(readAll(r))
		if err != nil {
			return fmt.Errorf("%s decode: %w", encName, err)
		}
		if _, err := w.Write(dec); err != nil {
			return err
		}

	case encBase16:
		dec, err := base16.StdEncoding.Decode(readAll(r))
		if err != nil {
			return fmt.Errorf("%s decode: %w", encName, err)
		}
		if _, err := w.Write(dec); err != nil {
			return err
		}
	}

	return nil
}

func readAll(r io.Reader) []byte {
	data, _ := io.ReadAll(r)
	return data
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: b64 [flags] [command] [file]

Commands:
  encode              Encode input (default if input looks binary)
  decode              Decode input (default if input looks like base64)
  completion <shell>  Generate shell completions (bash, zsh, fish)
  (no command)        Auto-detect encode vs decode

Flags:
  -w <cols>           Wrap encoded output at column width (default: 76, 0 = no wrap)
  --url               Use URL-safe alphabet (- instead of +, _ instead of /)
  --raw               URL-safe alphabet without padding
  --ignore-garbage    Skip non-alphabet characters during decode
  --check             Validate input format without decoding
  --base32            Use base32 encoding
  --base16            Use base16 (hex) encoding
  --help              Show this help

Examples:
  b64 encode file.txt
  b64 decode file.b64
  b64 -w 0 --url < input > output
  b64 --check < file.b64
  b64 --base32 < input
  b64 completion bash  # print bash completions
  eval "$(b64 completion zsh)"  # source zsh completions
`)
	}
}

func writeCompletion(shell string) {
	switch shell {
	case "bash":
		fmt.Print(bashCompletion)
	case "zsh":
		fmt.Print(zshCompletion)
	case "fish":
		fmt.Print(fishCompletion)
	default:
		die("unknown shell: %s (use bash, zsh, or fish)", shell)
	}
}

const bashCompletion = `_b64() {
	local cur prev opts
	COMPREPLY=()
	cur="${COMP_WORDS[COMP_CWORD]}"
	prev="${COMP_WORDS[COMP_CWORD-1]}"

	opts="encode decode completion -w --url --raw --ignore-garbage --check --base32 --base16 --help"

	case "${prev}" in
	encode|decode)
		_filedir
		return 0
		;;
	completion)
		COMPREPLY=($(compgen -W "bash zsh fish" -- "${cur}"))
		return 0
		;;
	-w)
		COMPREPLY=()
		return 0
		;;
	*)
		COMPREPLY=($(compgen -W "${opts}" -- "${cur}"))
		return 0
		;;
	esac
} && complete -F _b64 b64
`

const zshCompletion = `#compdef b64

_b64() {
	local -a opts
	opts=(
		'(-w)'{-w,}'[wrap encoded output at COLUMNS (0 = no wrap)]:columns'
		'--url[use URL-safe alphabet]'
		'--raw[URL-safe alphabet without padding]'
		'--ignore-garbage[skip non-alphabet characters during decode]'
		'--check[validate input format without decoding]'
		'--base32[use base32 encoding]'
		'--base16[use base16 (hex) encoding]'
		'--help[show help]'
		'encode:encode input: _files'
		'decode:decode input: _files'
		'completion:generate completions:(bash zsh fish)'
	)
	_describe -t commands 'b64' opts
}

_b64 "$@"
`

const fishCompletion = `complete -c b64 -f
complete -c b64 -n "__fish_use_subcommand" -a encode -d "Encode input"
complete -c b64 -n "__fish_use_subcommand" -a decode -d "Decode input"
complete -c b64 -n "__fish_use_subcommand" -a completion -d "Generate shell completions"
complete -c b64 -s w -l wrap -r -d "Wrap encoded output at COLUMNS (0 = no wrap)"
complete -c b64 -l url -d "Use URL-safe alphabet"
complete -c b64 -l raw -d "URL-safe alphabet without padding"
complete -c b64 -l ignore-garbage -d "Skip non-alphabet characters during decode"
complete -c b64 -l check -d "Validate input format without decoding"
complete -c b64 -l base32 -d "Use base32 encoding"
complete -c b64 -l base16 -d "Use base16 (hex) encoding"
complete -c b64 -l help -d "Show help"
`

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
