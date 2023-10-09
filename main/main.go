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
	newPath := flag.String("newPath", "", "The path where to move the region files to")
	minInhabitTime := flag.Int("minInhabitedTicks", 250, "The value that has to be passed so that a chunk will be seen as \"used\"")
	flag.Parse()

	println("Move Dir:", *newPath)
	println("RegionDir:", *regionDir)
	println("InhabitedMin:", *minInhabitTime)
	println("Verbose:", *verbose)
	println("Mode:", *mode)

	if len(*newPath) != 0 && !exists(*newPath) {
		log.Fatal("The path to move the .mca files to doesn't exist")
		return
	}

	if len(*regionDir) == 0 {
		log.Fatal("No path to the region files was given, see -h for flags and their usage")
		return
	} else {
		absPath, err := filepath.Abs(*regionDir)
		if err != nil {
			log.Fatal(*regionDir, "is not a recognized path")
			return
		}
		println(*regionDir)
		regionDir = &absPath
		println(absPath)
	}

	if !exists(*regionDir) {
		log.Fatal(*regionDir, "doesn't exist")
		return
	}

	if !strings.EqualFold(*mode, "perChunk") && !strings.EqualFold(*mode, "regionSum") {
		log.Fatal(*mode, "is not a valid comparison mode, must be on of \"perChunk\" or \"regionSum\"")
		return
	}

	err := process(*verbose, strings.EqualFold(*mode, "perChunk"), *regionDir, *newPath, int64(*minInhabitTime))
	if err != nil {
		log.Fatal(err)
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

func process(verbose bool, perChunkMode bool, path string, newPath string, minTime int64) (err error) {
	moveRegions := len(newPath) != 0

	regions, err := filepath.Glob(path + string(os.PathSeparator) + "r.*.*.mca")
	if err != nil {
		return err
	}

	if len(regions) == 0 {
		log.Println("Couldn't find any region files in", path)
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
				log.Println("Couldn't open file at", file, "skipping..")
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
						log.Println("Couldn't read sector at x:", x, "z:", z, "from", file, "- skipping region as used..")
					}
					continue regionLoop
				}

				var chunk Chunk
				err = chunk.Load(data)
				if err != nil {
					log.Println("Couldn't read chunk at x:", x, "z:", z, "from", file, "- skipping region as used..")
				}

				if perChunkMode {
					if chunk.Level.InhabitedTime > minTime {
						log.Println("Chunk at x:", chunk.XPos, "y:", chunk.YPos, "z:", chunk.ZPos, "is", chunk.Level.InhabitedTime, "ticks old, skipping region as used..")
						continue regionLoop
					}
				} else {
					regionSum += chunk.Level.InhabitedTime
					if regionSum > minTime {
						if verbose {
							log.Println(file, "has exceeded the minimum of inhabitant time, skipping region as used..")
						}
						continue regionLoop
					}
				}
			}
		}

		if verbose {
			log.Println("Handling", file, "..")
		}

		if moveRegions {
			err = os.Rename(file, filepath.Join(newPath, filepath.Base(file)))
			if err != nil && verbose {
				log.Println("Couldn't move", file)
			}
		} else {
			err = os.Remove(file)
			if err != nil && verbose {
				log.Println("Couldn't delete", file)
			}
		}
	}

	return nil
}
