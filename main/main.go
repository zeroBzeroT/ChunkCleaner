package main

import (
	"flag"
	"github.com/Tnze/go-mc/save/region"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	verbose := flag.Bool("v", false, "verbose - log everything")
	mode := flag.String("mode", "perChunk", "The mode by which the inhabited time will be compared to options: \"perChunk\"/\"regionSum\"")
	regionDir := flag.String("path", "", "The path of the directory with the .mca files that should be cleaned")
	minInhabitTime := flag.Int("minInhabitedTicks", 250, "The value that has to be passed so that a chunk will be seen as \"used\"")
	flag.Parse()

	println("RegionDir:", *regionDir)
	println("InhabitedMin:", *minInhabitTime)
	println("Verbose:", *verbose)
	println("Mode:", *mode)

	if len(*regionDir) == 0 {
		println("No path to the region files was given, see -h for flags and their usage")
		return
	}

	if !exists(*regionDir) {
		println(*regionDir, " doesn't exist")
		return
	}

	if !strings.EqualFold(*mode, "perChunk") && !strings.EqualFold(*mode, "regionSum") {
		println(*mode, "is not a valid comparison mode, must be on of \"perChunk\" or \"regionSum\"")
		return
	}

	err := process(*verbose, strings.EqualFold(*mode, "perChunk"), *regionDir, int64(*minInhabitTime))
	if err != nil {
		println(err)
		return
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}

	return false
}

func process(verbose bool, perChunkMode bool, path string, minTime int64) (err error) {
	regions, err := filepath.Glob(path + string(os.PathSeparator) + "r.*.*.mca")
	if err != nil {
		return err
	}

	if len(regions) == 0 {
		println("Couldn't find any region files in", path)
		return nil
	}

regionLoop:
	for _, file := range regions {
		if verbose {
			log.Println("Processing", file)
		}
		currentRegion, err := region.Open(file)
		if err != nil {
			if verbose {
				println("Couldn't open file at", file, "skipping..")
			}
			continue
		}

		var regionSum int64 = 0

		for x := 0; x < 32; x++ {
			for z := 0; z < 32; z++ {
				if !currentRegion.ExistSector(x, z) {
					continue
				}

				data, err := currentRegion.ReadSector(x, z)
				if err != nil {
					if verbose {
						println("Couldn't read sector at x:", x, "z:", z, "from", file, "- skipping region as used..")
					}
					continue regionLoop
				}

				var chunk Chunk
				err = chunk.Load(data)
				if err != nil {
					println("Couldn't read chunk at x:", x, "z:", z, "from", file, "- skipping region as used..")
				}

				if perChunkMode {
					if chunk.Level.InhabitedTime > minTime {
						println("Chunk at x:", chunk.XPos, "y:", chunk.YPos, "z:", chunk.ZPos, "is", chunk.Level.InhabitedTime, "ticks old, skipping region as used..")
						continue regionLoop
					}
				} else {
					regionSum += chunk.Level.InhabitedTime
					if regionSum > minTime {
						if verbose {
							println(file, "has exceeded the minimum of inhabitant time, skipping region as used..")
						}
						continue regionLoop
					}
				}
			}
		}

		if verbose {
			println("Deleting", file, "..")
		}

		err = os.Remove(file)
		if err != nil && verbose {
			println("Couldn't delete", file)
		}
	}

	return nil
}
