package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/Tnze/go-mc/save/region"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func main() {
	dry := flag.Bool("dryRun", false, "Trial run that doesn't make any changes")
	verbose := flag.Bool("v", false, "Increase amount of information during the process")
	mode := flag.String("mode", "perChunk", "Mode: \"perChunk\" or \"regionSum\"")
	regionDir := flag.String("path", "", "Path to directory with .mca files")
	newPath := flag.String("newPath", "", "Path where to move the region files")
	minInhabitTime := flag.Int("minInhabitedTicks", 250, "Minimum inhabited ticks to consider as used")
	flag.Parse()

	println("Move Dir:", *newPath)
	println("RegionDir:", *regionDir)
	println("InhabitedMin:", *minInhabitTime)
	println("Verbose:", *verbose)
	println("Mode:", *mode)
	println("Dry-Run:", *dry)

	if len(*newPath) != 0 && !exists(*newPath) {
		log.Fatal("The path to move the .mca files to doesn't exist")
		return
	}

	if len(*regionDir) == 0 {
		log.Fatal("No path to the region files was given")
		return
	} else {
		absPath, err := filepath.Abs(*regionDir)
		if err != nil {
			log.Fatal(*regionDir, " is not a recognized path")
			return
		}
		regionDir = &absPath
	}

	if !exists(*regionDir) {
		log.Fatal(*regionDir, " doesn't exist")
		return
	}

	if !strings.EqualFold(*mode, "perChunk") && !strings.EqualFold(*mode, "regionSum") {
		log.Fatal(*mode, " is not a valid mode")
		return
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, waiting for operations to complete...")
		cancel() // Signal all goroutines to stop
	}()

	err := process(ctx, *dry, *verbose, strings.EqualFold(*mode, "perChunk"), *regionDir, *newPath, int64(*minInhabitTime))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("All operations completed successfully")
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyFileAtomic(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmpDst := dst + ".tmp"
	out, err := os.OpenFile(tmpDst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	if _, err = io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmpDst)
		return err
	}

	if err = out.Sync(); err != nil {
		out.Close()
		os.Remove(tmpDst)
		return err
	}

	if err = out.Close(); err != nil {
		os.Remove(tmpDst)
		return err
	}

	if err = os.Rename(tmpDst, dst); err != nil {
		os.Remove(tmpDst)
		return err
	}

	if err = os.Remove(src); err != nil {
		return err
	}

	return nil
}

func process(ctx context.Context, dry bool, verbose bool, perChunkMode bool, path string, newPath string, minTime int64) (err error) {
	moveRegions := len(newPath) != 0

	regions, err := filepath.Glob(filepath.Join(path, "r.*.*.mca"))
	if err != nil {
		return err
	}

	if len(regions) == 0 {
		log.Println("Couldn't find any region files in", path)
		return nil
	}

	// Sorting by modification time (oldest first) and size (smallest first)
	type fileInfo struct {
		path string
		time time.Time
		size int64
	}
	var files []fileInfo
	for _, f := range regions {
		info, err := os.Stat(f)
		if err != nil {
			if verbose {
				log.Println("Couldn't stat file:", f)
			}
			continue
		}
		files = append(files, fileInfo{f, info.ModTime(), info.Size()})
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].time.Equal(files[j].time) {
			return files[i].size < files[j].size
		}
		return files[i].time.Before(files[j].time)
	})

	// Rewrite regions in sorted order
	regions = nil
	for _, f := range files {
		regions = append(regions, f.path)
	}

	// Channel for collecting results
	type result struct {
		file      string
		chunkMax  int64
		regionSum int64
		err       error
	}
	results := make(chan result, len(regions))
	var wg sync.WaitGroup

	// Limit concurrent goroutines
	maxWorkers := runtime.NumCPU()
	sem := make(chan struct{}, maxWorkers)

	for i, file := range regions {
		select {
		case <-ctx.Done():
			break
		default:
			wg.Add(1)
			sem <- struct{}{} // Acquire semaphore
			go func(file string, index uint64) {
				defer wg.Done()
				defer func() { <-sem }() // Release semaphore

				if verbose {
					log.Println("Processing", file)
				}

				currentRegion, err := region.Open(file)
				if err != nil {
					if verbose {
						log.Println("Couldn't open file at", file, "skipping..")
					}
					results <- result{file: file, err: err}
					return
				}

				var regionSum int64 = 0
				var chunkMax int64 = 0

				for x := 0; x < 32; x++ {
					for z := 0; z < 32; z++ {
						select {
						case <-ctx.Done():
							log.Println("Aborting processing of", file, "due to shutdown signal")
							results <- result{file: file, err: ctx.Err()}
							return
						default:
							if !currentRegion.ExistSector(x, z) {
								continue
							}

							data, err := currentRegion.ReadSector(x, z)
							if err != nil {
								log.Println("Couldn't read sector at x:", x, "z:", z, "from", file)
								results <- result{file: file, err: err}
								return
							}

							var chunk Chunk
							err = chunk.Load(data)
							if err != nil {
								log.Println("Couldn't read chunk at x:", x, "z:", z, "from", file)
								results <- result{file: file, err: err}
								return
							}

							xPos := chunk.XPos + chunk.Level.XPos
							zPos := chunk.ZPos + chunk.Level.ZPos
							inhabitedTime := max(chunk.InhabitedTime, chunk.Level.InhabitedTime)

							if inhabitedTime < 0 {
								log.Println("Skipping", file, "invalid chunk at x:", xPos, "z:", zPos)
								results <- result{file: file, err: fmt.Errorf("invalid chunk at x:%d z:%d", xPos, zPos)}
								return
							}

							if perChunkMode {
								if inhabitedTime > minTime {
									if verbose {
										log.Println("Chunk at x:", xPos, "z:", zPos, "is", inhabitedTime, "ticks old - skipping region")
									}
									results <- result{file: file}
									return
								} else if inhabitedTime > chunkMax {
									chunkMax = inhabitedTime
								}
							} else {
								regionSum += inhabitedTime
								if regionSum > minTime {
									if verbose {
										log.Println(file, "has exceeded inhabit time - skipping. inhabitedTime:", regionSum)
									}
									results <- result{file: file}
									return
								}
							}
						}
					}
				}

				// Logging and file operation
				logStr := "[" + strconv.FormatUint(index+1, 10) + "/" + strconv.Itoa(len(regions)) + "] "
				if dry {
					logStr += "Dry-Run "
				} else if moveRegions && (perChunkMode || regionSum > 0) {
					logStr += "Copying "
				} else if !perChunkMode && regionSum == 0 {
					logStr += "Skipping (Cum. ticks 0) "
				} else {
					logStr += "Deleting "
				}

				info, _ := os.Stat(file)
				logStr += file
				if perChunkMode {
					log.Println(logStr, "- Max. ticks", chunkMax, "-", info.ModTime())
				} else {
					log.Println(logStr, "- Cum. ticks", regionSum, "-", info.ModTime())
				}

				if !dry && (perChunkMode || regionSum > 0) {
					select {
					case <-ctx.Done():
						log.Println("Aborting file operation for", file, "due to shutdown signal")
						results <- result{file: file, err: ctx.Err()}
						return
					default:
						if moveRegions {
							dstPath := filepath.Join(newPath, filepath.Base(file))
							err = copyFileAtomic(file, dstPath)
							if err != nil {
								log.Println("Couldn't copy", file, "->", dstPath, ":", err)
								results <- result{file: file, err: err}
								return
							}
						} else {
							err = os.Remove(file)
							if err != nil && verbose {
								log.Println("Couldn't delete", file)
								results <- result{file: file, err: err}
								return
							}
						}
					}
				}

				results <- result{file: file, chunkMax: chunkMax, regionSum: regionSum}
			}(file, uint64(i))
		}
	}

	// Close results channel after all goroutines finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for res := range results {
		if res.err != nil {
			if verbose {
				log.Println("Error processing", res.file, ":", res.err)
			}
			continue
		}
	}

	return nil
}
