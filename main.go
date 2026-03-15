// Command gomapper generates type-safe struct mapping functions from Go struct definitions.
//
// Usage:
//
//	//go:generate gomapper -src User -dst UserDTO
//	//go:generate gomapper -pairs User:UserDTO,Order:OrderDTO -bidirectional
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/KARTIKrocks/gomapper/internal/generator"
	"github.com/KARTIKrocks/gomapper/internal/loader"
	"github.com/KARTIKrocks/gomapper/internal/matcher"
	"golang.org/x/tools/go/packages"
)

type pair struct{ Src, Dst string }

func main() {
	var (
		src           = flag.String("src", "", "source type name")
		dst           = flag.String("dst", "", "destination type name")
		pairs         = flag.String("pairs", "", "comma-separated Src:Dst pairs (alternative to -src/-dst)")
		output        = flag.String("output", "mapper_gen.go", "output file name")
		mode          = flag.String("mode", "func", `generation mode: "func" (default), "register", or "both"`)
		bidirectional = flag.Bool("bidirectional", false, "generate both S→D and D→S mappings")
		tagKey        = flag.String("tag", "map", "struct tag key for field renaming")
		strict        = flag.Bool("strict", false, "fail if any destination field is unmapped")
		ci            = flag.Bool("ci", false, "case-insensitive field name matching")
		nilSafe       = flag.Bool("nil-safe", false, "generate nil checks for pointer dereferences")
		verbose       = flag.Bool("v", false, "verbose: print field matching decisions")
	)
	flag.Parse()

	typePairs := parseTypePairs(*pairs, *src, *dst)

	// Expand bidirectional pairs.
	if *bidirectional {
		typePairs = expandBidirectional(typePairs)
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("getwd: %v", err)
	}
	pkg, err := loader.Load(dir)
	if err != nil {
		log.Fatalf("loading package: %v", err)
	}

	genPairs := matchAll(pkg, typePairs, *tagKey, *strict, *ci, *verbose)

	data := generator.Data{
		PkgName: pkg.Name,
		Pairs:   genPairs,
		NilSafe: *nilSafe,
	}

	genMode := generator.Mode(*mode)
	switch genMode {
	case generator.ModeRegister, generator.ModeFunc, generator.ModeBoth:
	default:
		log.Fatalf("invalid -mode %q; must be \"register\", \"func\", or \"both\"", *mode)
	}
	if err := generator.WriteFile(data, genMode, *output); err != nil {
		log.Fatalf("generating: %v", err)
	}

	if *verbose {
		fmt.Printf("wrote %s (%d pair(s))\n", *output, len(genPairs))
	}
}

// parseTypePairs parses -pairs or -src/-dst flags into a slice of pairs.
func parseTypePairs(pairsFlag, src, dst string) []pair {
	switch {
	case pairsFlag != "":
		var typePairs []pair
		for p := range strings.SplitSeq(pairsFlag, ",") {
			parts := strings.SplitN(strings.TrimSpace(p), ":", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				log.Fatalf("invalid pair format %q; expected Src:Dst", p)
			}
			typePairs = append(typePairs, pair{parts[0], parts[1]})
		}
		return typePairs
	case src != "" && dst != "":
		return []pair{{src, dst}}
	default:
		fmt.Fprintln(os.Stderr, "usage: gomapper -src Type -dst Type")
		fmt.Fprintln(os.Stderr, "       gomapper -pairs Src:Dst[,Src:Dst,...]")
		flag.PrintDefaults()
		os.Exit(1)
		return nil
	}
}

// expandBidirectional adds reverse pairs for each input pair.
func expandBidirectional(typePairs []pair) []pair {
	var expanded []pair
	for _, p := range typePairs {
		expanded = append(expanded, p, pair{p.Dst, p.Src})
	}
	return expanded
}

// matchAll performs field matching for all pairs using the already-loaded package.
func matchAll(pkg *packages.Package, typePairs []pair, tagKey string, strict, ci, verbose bool) []generator.PairData {
	cfg := matcher.Config{
		TagKey:          tagKey,
		Strict:          strict,
		CaseInsensitive: ci,
		Verbose:         verbose,
	}

	var genPairs []generator.PairData
	for _, p := range typePairs {
		srcInfo, err := loader.LookupStruct(pkg, p.Src)
		if err != nil {
			log.Fatalf("source type: %v", err)
		}
		dstInfo, err := loader.LookupStruct(pkg, p.Dst)
		if err != nil {
			log.Fatalf("destination type: %v", err)
		}

		result, err := matcher.Match(srcInfo, dstInfo, cfg)
		if err != nil {
			log.Fatalf("matching %s → %s: %v", p.Src, p.Dst, err)
		}

		genPairs = append(genPairs, generator.PairData{
			SrcType:              p.Src,
			DstType:              p.Dst,
			Mappings:             result.Mappings,
			Unmapped:             result.Unmapped,
			NestedDstAssignments: result.NestedDstAssignments,
		})
	}

	return genPairs
}
