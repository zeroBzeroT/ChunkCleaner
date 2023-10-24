# ChunkCleaner
A tool designed to automatically delete chunks/region files with low playtime, optimizing space usage and improving performance.

## Arguments

| Argument          | Default  | Description                                                                                    |
|-------------------|----------|------------------------------------------------------------------------------------------------|
| dryRun            | false    | Trial run that doesn't make any changes                                                        |
| v                 | false    | This option increases the amount of information you are given during the process               |
| mode              | perChunk | The mode by which the inhabited time will be compared to options: \"perChunk\" / \"regionSum\" |
| path              |          | The path of the directory with the .mca files that should be cleaned                           |
| newPath           |          | The path where to move the region files to                                                     |
| minInhabitedTicks | 250      | The value that has to be passed so that a chunk will be seen as \"used\"                       |
