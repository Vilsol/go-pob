// Command logo regenerates every shipped icon from assets/logo/go-pob-logo.svg: a lossless svgo minify (proved
// pixel-identical in headless Chromium), the web favicons and PWA icons, and the desktop png / ico / icns. PNGs are
// rendered by Chromium, then go through pngquant and oxipng; all four tools come from one pinned nixpkgs revision. assets/logo/outputs.sha256
// records the inputs and outputs of the last run; --check verifies it without the tools.
package main

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

const (
	nixpkgs  = "github:NixOS/nixpkgs/44a91898084f46797b5fac650c7e8c9ac38c43d4"
	source   = "assets/logo/go-pob-logo.svg"
	stamp    = "assets/logo/outputs.sha256"
	svgoCfg  = "scripts/logo/svgo.config.mjs"
	renderJS = "frontend/scripts/logo-render.mjs"
	minified = "frontend/static/favicon.svg"
)

var (
	// everything that decides the output bytes, besides the pinned tool versions named in this file
	inputs = []string{source, svgoCfg, renderJS, "scripts/logo/main.go"}
	tools  = []string{"chromium", "svgo", "pngquant", "oxipng"}
	// the tile's outermost gradient stop, behind the logo wherever an icon must be opaque
	background = color.NRGBA{R: 0x0e, G: 0x0b, B: 0x09, A: 0xff}
	proofSizes = []int{16, 32, 64, 180, 512, 2048}
)

type raster struct {
	size int
	// opaque: the logo over the background; pad: the logo in the centre 80% (a maskable icon's safe zone)
	opaque, pad bool
}

func (r raster) inner() int {
	if !r.pad {
		return r.size
	}
	n := r.size * 4 / 5
	return n + (r.size-n)%2
}

func (r raster) name() string {
	return fmt.Sprintf("%d%s%s", r.size, map[bool]string{true: "-opaque"}[r.opaque], map[bool]string{true: "-pad"}[r.pad])
}

func plain(n int) raster { return raster{size: n} }

var (
	pngs = map[string]raster{
		"frontend/static/favicon.png":           plain(64),
		"frontend/static/icon-192.png":          plain(192),
		"frontend/static/icon-512.png":          plain(512),
		"frontend/static/apple-touch-icon.png":  {size: 180, opaque: true},
		"frontend/static/icon-maskable-512.png": {size: 512, opaque: true, pad: true},
		"desktop/icon.png":                      plain(512),
	}
	icos = map[string][]int{
		"frontend/static/favicon.ico":    {16, 32, 48},
		"desktop/build/windows/icon.ico": {16, 24, 32, 48, 64, 128, 256},
	}
	// Apple's icns PNG types and their pixel sizes
	icnsTypes = []struct {
		typ  string
		size int
	}{{"ic11", 32}, {"ic12", 64}, {"ic07", 128}, {"ic13", 256}, {"ic08", 256}, {"ic14", 512}, {"ic09", 512}, {"ic10", 1024}}
	icnsPath = "desktop/build/darwin/icons.icns"
)

func main() {
	check := flag.Bool("check", false, "verify "+stamp+" against the committed inputs and outputs, without the tools")
	review := flag.String("review", "", "also write every unquantised render and final PNG to this directory")
	flag.Parse()
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "logo: unexpected argument %q\n", flag.Arg(0))
		os.Exit(2)
	}
	var err error
	if *check {
		err = verify()
	} else {
		err = regenerate(context.Background(), *review)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "logo:", err)
		os.Exit(1)
	}
}

func regenerate(ctx context.Context, review string) error {
	out, err := generate(ctx, review)
	if err != nil {
		return err
	}
	paths := sortedKeys(out)
	written := 0
	for _, p := range paths {
		if old, _ := os.ReadFile(p); bytes.Equal(old, out[p]) { //nolint:gosec // the generator's own output list
			continue
		}
		written++
		if err := os.WriteFile(p, out[p], 0o644); err != nil { //nolint:gosec // committed assets are world-readable
			return fmt.Errorf("write: %w", err)
		}
	}
	sums, err := hashes(append(append([]string{}, inputs...), paths...))
	if err != nil {
		return err
	}
	if err := os.WriteFile(stamp, []byte(sums), 0o644); err != nil { //nolint:gosec // committed assets are world-readable
		return fmt.Errorf("write: %w", err)
	}
	fmt.Printf("logo: %d outputs, %d rewritten\n", len(paths), written)
	return nil
}

// verify re-hashes every file the stamp names; a moved input means the outputs were not regenerated from it, a
// moved output means it was edited by hand.
func verify() error {
	f, err := os.Open(stamp)
	if err != nil {
		return fmt.Errorf("stamp: %w", err)
	}
	defer f.Close()
	var files []string
	want := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		sum, path, ok := strings.Cut(sc.Text(), "  ")
		if !ok {
			return fmt.Errorf("stamp: bad line %q", sc.Text())
		}
		files = append(files, path)
		want[path] = sum
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("stamp: %w", err)
	}
	for _, in := range inputs {
		if want[in] == "" {
			return fmt.Errorf("stamp: no line for input %s", in)
		}
	}
	got, err := hashes(files)
	if err != nil {
		return err
	}
	var stale []string
	for line := range strings.Lines(got) {
		sum, path, _ := strings.Cut(strings.TrimSuffix(line, "\n"), "  ")
		if want[path] != sum {
			stale = append(stale, path)
		}
	}
	if len(stale) > 0 {
		return fmt.Errorf("%d file(s) differ from %s, run `mise run logo`: %s", len(stale), stamp, strings.Join(stale, ", "))
	}
	fmt.Printf("logo: %d files match %s\n", len(files), stamp)
	return nil
}

func hashes(paths []string) (string, error) {
	var b strings.Builder
	for _, p := range paths {
		data, err := os.ReadFile(p) //nolint:gosec // paths from the generator or its stamp
		if err != nil {
			return "", fmt.Errorf("hash: %w", err)
		}
		sum := sha256.Sum256(data)
		fmt.Fprintf(&b, "%s  %s\n", hex.EncodeToString(sum[:]), p)
	}
	return b.String(), nil
}

type pipeline struct {
	ctx    context.Context //nolint:containedctx // one run of a command-line tool
	bin    map[string]string
	tmp    string
	review string
	render map[int][]byte    // the minified svg rendered by Chromium, per size
	final  map[string][]byte // quantised PNG per raster name
}

func generate(ctx context.Context, review string) (map[string][]byte, error) {
	bin, err := resolveTools(ctx)
	if err != nil {
		return nil, err
	}
	tmp, err := os.MkdirTemp("", "logo-")
	if err != nil {
		return nil, fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)
	if review != "" {
		if err := os.MkdirAll(review, 0o750); err != nil {
			return nil, fmt.Errorf("review dir: %w", err)
		}
	}
	p := &pipeline{ctx: ctx, bin: bin, tmp: tmp, review: review, final: map[string][]byte{}}

	srcSVG, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("source: %w", err)
	}
	minSVG, err := p.run("svgo", "--config", svgoCfg, "-i", source, "-o", "-")
	if err != nil {
		return nil, err
	}
	minPath := filepath.Join(tmp, "min.svg")
	if err := os.WriteFile(minPath, minSVG, 0o600); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}
	fmt.Printf("svg: %d B -> %d B\n", len(srcSVG), len(minSVG))

	sizes := map[int]bool{}
	for _, n := range proofSizes {
		sizes[n] = true
	}
	for _, r := range pngs {
		sizes[r.inner()] = true
	}
	for _, s := range icos {
		for _, n := range s {
			sizes[n] = true
		}
	}
	for _, e := range icnsTypes {
		sizes[e.size] = true
	}
	ref, err := p.chromium(source, "src", proofSizes)
	if err != nil {
		return nil, err
	}
	if p.render, err = p.chromium(minPath, "min", sortedKeys(sizes)); err != nil {
		return nil, err
	}
	for _, n := range proofSizes {
		d, err := diff(ref[n], p.render[n])
		if err != nil {
			return nil, err
		}
		fmt.Printf("proof %4dpx: max channel diff %d\n", n, d.max)
		if d.max != 0 {
			return nil, fmt.Errorf("the minified svg renders differently at %dpx (max channel diff %d)", n, d.max)
		}
	}

	out := map[string][]byte{minified: minSVG}
	for _, path := range sortedKeys(pngs) {
		if out[path], err = p.png(pngs[path]); err != nil {
			return nil, err
		}
	}
	for _, path := range sortedKeys(icos) {
		if out[path], err = p.ico(icos[path]); err != nil {
			return nil, err
		}
	}
	if out[icnsPath], err = p.icns(); err != nil {
		return nil, err
	}
	return out, nil
}

func resolveTools(ctx context.Context) (map[string]string, error) {
	args := make([]string, 0, 3+len(tools))
	args = append(args, "build", "--no-link", "--print-out-paths")
	for _, t := range tools {
		args = append(args, nixpkgs+"#"+t)
	}
	cmd := exec.CommandContext(ctx, "nix", args...)
	cmd.Stderr = os.Stderr
	res, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nix build %v (the renderer and the PNG tools come from nixpkgs): %w", tools, err)
	}
	bin := map[string]string{}
	for _, dir := range strings.Fields(string(res)) {
		for _, t := range tools {
			if _, err := os.Stat(filepath.Join(dir, "bin", t)); err == nil {
				bin[t] = filepath.Join(dir, "bin", t)
			}
		}
	}
	for _, t := range tools {
		if bin[t] == "" {
			return nil, fmt.Errorf("nix built no bin/%s", t)
		}
	}
	return bin, nil
}

func (p *pipeline) run(tool string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(p.ctx, p.bin[tool], args...) //nolint:gosec // a pinned tool from the nix store
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	res, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w\n%s", tool, strings.Join(args, " "), err, stderr.String())
	}
	return res, nil
}

func (p *pipeline) chromium(svg, tag string, sizes []int) (map[int][]byte, error) {
	dir := filepath.Join(p.tmp, tag)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("render dir: %w", err)
	}
	args := []string{renderJS, svg, dir}
	for _, n := range sizes {
		args = append(args, fmt.Sprint(n))
	}
	cmd := exec.CommandContext(p.ctx, "node", args...)
	cmd.Env = append(os.Environ(), "LOGO_CHROMIUM="+p.bin["chromium"])
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("chromium render: %w", err)
	}
	out := map[int][]byte{}
	for _, n := range sizes {
		b, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("%d.png", n))) //nolint:gosec // our temp dir
		if err != nil {
			return nil, fmt.Errorf("chromium render: %w", err)
		}
		out[n] = b
		if err := p.keep(fmt.Sprintf("render-%s-%d.png", tag, n), b); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (p *pipeline) keep(name string, b []byte) error {
	if p.review == "" {
		return nil
	}
	if err := os.WriteFile(filepath.Join(p.review, name), b, 0o600); err != nil { //nolint:gosec // the directory --review names
		return fmt.Errorf("review: %w", err)
	}
	return nil
}

// png quantises one render (flattened onto the background for an opaque icon), then recompresses it losslessly.
func (p *pipeline) png(r raster) ([]byte, error) {
	if b, ok := p.final[r.name()]; ok {
		return b, nil
	}
	raw := p.render[r.inner()]
	if r.opaque {
		var err error
		if raw, err = flatten(raw, r.size); err != nil {
			return nil, err
		}
	}
	in := filepath.Join(p.tmp, r.name()+".png")
	if err := os.WriteFile(in, raw, 0o600); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}
	quant, err := p.run("pngquant", "--speed", "1", "--quality", "80-100", "--strip", "--output", "-", "--", in)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(in, quant, 0o600); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}
	final, err := p.run("oxipng", "--opt", "max", "--strip", "all", "--zopfli", "--stdout", in)
	if err != nil {
		return nil, err
	}
	d, err := diff(raw, final)
	if err != nil {
		return nil, err
	}
	fmt.Printf("png %-16s %7d -> pngquant %6d -> oxipng %6d B; vs render: max %3d mean %.3f psnr %.1f dB\n",
		r.name(), len(raw), len(quant), len(final), d.max, d.mean, d.psnr)
	p.final[r.name()] = final
	if err := p.keep("final-"+r.name()+".png", final); err != nil {
		return nil, err
	}
	return final, nil
}

// flatten centres the rendered logo on an opaque size×size canvas of the background colour.
func flatten(raw []byte, size int) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	canvas := image.NewNRGBA(image.Rect(0, 0, size, size))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(background), image.Point{}, draw.Src)
	off := (size - img.Bounds().Dx()) / 2
	draw.Draw(canvas, img.Bounds().Add(image.Pt(off, off)), img, img.Bounds().Min, draw.Over)
	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}
	return buf.Bytes(), nil
}

// ico writes a PNG-in-ICO container (Windows Vista and later), one entry per size.
func (p *pipeline) ico(sizes []int) ([]byte, error) {
	var head, body bytes.Buffer
	le16 := func(v ...int) {
		for _, x := range v {
			head.Write(binary.LittleEndian.AppendUint16(nil, uint16(x&0xffff))) //nolint:gosec // masked to 16 bits
		}
	}
	le16(0, 1, len(sizes))
	off := 6 + 16*len(sizes)
	for _, n := range sizes {
		b, err := p.png(plain(n))
		if err != nil {
			return nil, err
		}
		dim := byte(n % 256) //nolint:gosec // 0 means 256
		head.Write([]byte{dim, dim, 0, 0})
		le16(1, 32, len(b), len(b)>>16, off+body.Len(), (off+body.Len())>>16)
		body.Write(b)
	}
	return append(head.Bytes(), body.Bytes()...), nil
}

// icns writes Apple's icon container with PNG entries.
func (p *pipeline) icns() ([]byte, error) {
	var body bytes.Buffer
	for _, e := range icnsTypes {
		b, err := p.png(plain(e.size))
		if err != nil {
			return nil, err
		}
		body.WriteString(e.typ)
		body.Write(be32(8 + len(b)))
		body.Write(b)
	}
	return append(append([]byte("icns"), be32(8+body.Len())...), body.Bytes()...), nil
}

func be32(v int) []byte { return binary.BigEndian.AppendUint32(nil, uint32(v)) } //nolint:gosec // container sizes fit 32 bits

type delta struct {
	max        int
	mean, psnr float64
}

// diff compares two PNGs channel by channel on premultiplied 8-bit RGBA, so colour under full transparency is ignored.
func diff(a, b []byte) (delta, error) {
	ia, err := png.Decode(bytes.NewReader(a))
	if err != nil {
		return delta{}, fmt.Errorf("decode: %w", err)
	}
	ib, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return delta{}, fmt.Errorf("decode: %w", err)
	}
	if ia.Bounds() != ib.Bounds() {
		return delta{}, fmt.Errorf("diff: sizes differ")
	}
	var d delta
	var sum, sq float64
	n := 0
	for y := ia.Bounds().Min.Y; y < ia.Bounds().Max.Y; y++ {
		for x := ia.Bounds().Min.X; x < ia.Bounds().Max.X; x++ {
			r1, g1, b1, a1 := ia.At(x, y).RGBA()
			r2, g2, b2, a2 := ib.At(x, y).RGBA()
			w := [4]uint32{r2, g2, b2, a2}
			for i, v := range [4]uint32{r1, g1, b1, a1} {
				e := int(v>>8) - int(w[i]>>8)
				if e < 0 {
					e = -e
				}
				d.max = max(d.max, e)
				sum += float64(e)
				sq += float64(e * e)
				n++
			}
		}
	}
	d.mean = sum / float64(n)
	d.psnr = math.Inf(1)
	if sq > 0 {
		d.psnr = 10 * math.Log10(255*255/(sq/float64(n)))
	}
	return d, nil
}

func sortedKeys[K cmp.Ordered, V any](m map[K]V) []K {
	k := make([]K, 0, len(m))
	for n := range m {
		k = append(k, n)
	}
	slices.Sort(k)
	return k
}
