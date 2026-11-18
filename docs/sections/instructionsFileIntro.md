[Back to Table of contents](../index.md)  

# Instructions file introduction

## What is an instructions file?

The instructions file is the instructions for manipulating a set of YAML files and documents.  It is a YAML document that contains an array of paths to YAML files, directories, and documents within YAML files to perform operations.  Each instructions file starts with the optional key `commonOverlays`, which is an array, along with the key `yamlFiles` which is also an array.  

The `yamlFiles` key is optional. An instructions file with only `commonOverlays` works as a reusable library of overlays: pass it with `-i` and provide the YAML files from another instructions file, from `-p`/`--path`, or by piping YAML to stdin. If no YAML files are provided by any of these, Yot exits with an error.

Multiple instructions files can be passed by repeating `-i`. They are applied in the order given, and a later file wins when it conflicts with an earlier one. See [Order of operations](orderOfOperations.md#combining-multiple-instructions-files).  


[Back to Table of contents](../index.md)  
[Next Up: Instructions file YAML specification](instructionsFileSpec.md)