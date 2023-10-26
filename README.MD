# ChunkCleaner
A tool designed to automatically delete region files (.mca) with low playtime, optimizing space usage and improving performance.

![logo](https://github.com/zeroBzeroT/ChunkCleaner/blob/master/logo.svg?raw=true)

[![discord](https://img.shields.io/discord/843551077759844362?logo=discord)](https://discord.gg/7tW8ZAtGr5)
[![reddit](https://img.shields.io/reddit/subreddit-subscribers/0b0t)](https://old.reddit.com/r/0b0t/)
![last commit](https://img.shields.io/github/last-commit/zeroBzeroT/ChunkCleaner)
![code size](https://img.shields.io/github/languages/code-size/zeroBzeroT/ChunkCleaner)
[![downloads](https://img.shields.io/github/downloads/zeroBzeroT/ChunkCleaner/total)](https://github.com/zeroBzeroT/ChunkCleaner/releases)

## Considerations
In some chunks in Minecraft, the "InhabitedTime" variable does not update as expected, resulting in slower increments than anticipated. This issue has been observed in multiple versions of the game, including 1.12.2, 1.17.1 and 1.18.2. See https://bugs.mojang.com/browse/MC-127407.

## Arguments

| Argument             | Default    | Description                                                                                    |
|----------------------|------------|------------------------------------------------------------------------------------------------|
| `-h`                 |            | See the arguments and their usage                                                              |
| `-dryRun`            | `false`    | Trial run that doesn't make any changes                                                        |
| `-minInhabitedTicks` | `250`      | The value that has to be passed so that a chunk will be seen as \"used\"                       |
| `-mode`              | `perChunk` | The mode by which the inhabited time will be compared to options: \"perChunk\" / \"regionSum\" |
| `-newPath`           |            | The path where to move the region files to (if not provides .mca files will be deleted)        |
| `-path`              |            | The path of the directory with the .mca files that should be cleaned                           |
| `-v`                 | `false`    | This option increases the amount of information you are given during the process               |

## TODO

- Add `LightPopulated` and `TerrainPopulated` as options

## Warranty

The Software is provided "as is" and without warranties of any kind, express or implied, including but not limited to the warranties of merchantability, fitness for a particular purpose, and non-infringement. In no event shall the Authors or copyright owners be liable for any claims, damages or other liability, whether in an action in contract, tort or otherwise, arising from, out of or in connection with the Software or the use or other dealings in the Software.
